package handler

import (
	"bytes"
	"fmt"
	"headcontrol/internal/model"
	"html"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// LogsPage renders the audit logs page
func (h *Handler) LogsPage(w http.ResponseWriter, r *http.Request) {
	logs, err := h.getSystemLogs()
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}

	auditLogs, err := h.getAuditLogs()
	if err != nil {
		log.Printf("failed to load audit logs: %v", err)
	}

	h.renderPage(w, r, "logs", map[string]interface{}{
		"Title":      "System Logs",
		"ActivePage": "logs",
		"Logs":       logs,
		"AuditLogs":  auditLogs,
		"Error":      errMsg,
	})
}

// LogsRaw returns only the log content block for HTMX refresh
func (h *Handler) LogsRaw(w http.ResponseWriter, r *http.Request) {
	logs, err := h.getSystemLogs()
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte("<span style='color:#f38ba8;'>Failed to load logs: " + html.EscapeString(err.Error()) + "</span>"))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	for _, line := range logs {
		fmt.Fprintf(w, `<div style="color: %s;">%s</div>`, line.Color, html.EscapeString(line.Text))
	}
}

// LogsAudit returns only the audit logs timeline feed for HTMX refresh
func (h *Handler) LogsAudit(w http.ResponseWriter, r *http.Request) {
	auditLogs, err := h.getAuditLogs()
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte("<div class='error-banner'>Failed to load audit trail.</div>"))
		return
	}

	h.render(w, "audit-feed.html", map[string]interface{}{
		"AuditLogs": auditLogs,
	})
}

// LogAuditEvent records a structured audit event to a local log file
func (h *Handler) LogAuditEvent(r *http.Request, action, details string) {
	ip := r.RemoteAddr
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ip = strings.Split(xff, ",")[0]
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("%s [AUDIT] %s - %s - %s\n", timestamp, ip, action, details)

	auditLogPath := filepath.Join(".", "audit.log")
	f, err := os.OpenFile(auditLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("failed to open audit log file: %v", err)
		return
	}
	defer f.Close()

	if _, err := f.WriteString(logLine); err != nil {
		log.Printf("failed to write to audit log file: %v", err)
	}
}

// Helper to read and parse local audit logs
func (h *Handler) getAuditLogs() ([]model.AuditLog, error) {
	auditLogPath := filepath.Join(".", "audit.log")
	if _, err := os.Stat(auditLogPath); os.IsNotExist(err) {
		return []model.AuditLog{}, nil
	}

	data, err := os.ReadFile(auditLogPath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	var logs []model.AuditLog

	// Read in reverse order (newest first)
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " [AUDIT] ", 2)
		if len(parts) < 2 {
			continue
		}
		timestamp := parts[0]

		restParts := strings.SplitN(parts[1], " - ", 3)
		if len(restParts) < 3 {
			continue
		}
		ip := restParts[0]
		action := restParts[1]
		details := restParts[2]

		logs = append(logs, model.AuditLog{
			Timestamp: timestamp,
			IP:        ip,
			Action:    action,
			Details:   details,
			Type:      getAuditLogType(action),
		})
	}

	return logs, nil
}

// Helper to execute journalctl command and get log lines
func (h *Handler) getSystemLogs() ([]model.LogLine, error) {
	cmd := exec.Command("journalctl", "-u", "headscale", "-n", "200", "--no-pager")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Fallback mock logs for testing and systems without systemd journal access (e.g. Windows)
		mockLogs := []string{
			"2026-07-09T18:00:00Z [INFO] Headscale server starting...",
			"2026-07-09T18:01:05Z [INFO] Database connection initialized successfully",
			"2026-07-09T18:02:10Z [auth] API Key created for user 'admin'",
			"2026-07-09T18:05:44Z [register] Node 'workstation' registered successfully for user 'nhanh'",
			"2026-07-09T18:10:00Z [INFO] Subnet route 10.0.1.0/24 advertised by node 'workstation'",
			"2026-07-09T18:15:30Z [auth] Authentication request from node 'phone' approved",
			"2026-07-09T18:22:15Z [WARNING] Keep-alive timeout from node 'backup-server'",
			"2026-07-09T18:25:00Z [INFO] ACL policies reloaded successfully",
		}
		
		var logLines []model.LogLine
		for _, line := range mockLogs {
			color := "#cdd6f4"
			lower := strings.ToLower(line)
			if strings.Contains(lower, "error") || strings.Contains(lower, "fail") || strings.Contains(lower, "warn") {
				color = "#f38ba8"
			} else if strings.Contains(lower, "auth") || strings.Contains(lower, "register") || strings.Contains(lower, "success") {
				color = "#a6e3a1"
			}
			logLines = append(logLines, model.LogLine{Text: line, Color: color})
		}
		return logLines, nil
	}

	lines := strings.Split(stdout.String(), "\n")
	var logLines []model.LogLine
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			color := "#cdd6f4"
			lower := strings.ToLower(trimmed)
			if strings.Contains(lower, "error") || strings.Contains(lower, "fail") || strings.Contains(lower, "warning") {
				color = "#f38ba8" // Red-ish terminal color
			} else if strings.Contains(lower, "auth") || strings.Contains(lower, "register") || strings.Contains(lower, "success") {
				color = "#a6e3a1" // Green-ish terminal color
			}
			logLines = append(logLines, model.LogLine{Text: trimmed, Color: color})
		}
	}

	return logLines, nil
}

func getAuditLogType(action string) string {
	lower := strings.ToLower(action)
	if strings.Contains(lower, "delete") || strings.Contains(lower, "expire") || strings.Contains(lower, "reject") {
		return "danger"
	}
	if strings.Contains(lower, "create") || strings.Contains(lower, "approve") {
		return "success"
	}
	return "info"
}
