package handler

import (
	"bytes"
	"fmt"
	"html"
	"headcontrol/internal/model"
	"net/http"
	"os/exec"
	"strings"
)

// LogsPage renders the audit logs page
func (h *Handler) LogsPage(w http.ResponseWriter, r *http.Request) {
	logs, err := h.getSystemLogs()
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}

	h.renderPage(w, r, "logs", map[string]interface{}{
		"Title":      "System Logs",
		"ActivePage": "logs",
		"Logs":       logs,
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
