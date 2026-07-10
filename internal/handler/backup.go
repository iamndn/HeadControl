package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"headcontrol/internal/model"
	"io"
	"net/http"
	"os/exec"
	"time"
)

// ExportBackup generates a JSON backup file for download
func (h *Handler) ExportBackup(w http.ResponseWriter, r *http.Request) {
	backup := model.BackupData{
		ExportedAt: time.Now().Format(time.RFC3339),
		Version:    "1.0",
	}

	// 1. Local settings (mask API key for security)
	if settings, err := h.store.GetSettings(); err == nil && settings != nil {
		maskedKey := ""
		if len(settings.APIKey) > 8 {
			maskedKey = settings.APIKey[:4] + "****" + settings.APIKey[len(settings.APIKey)-4:]
		}
		backup.Settings = &model.Settings{
			BaseURL:   settings.BaseURL,
			APIKey:    maskedKey,
			CreatedAt: settings.CreatedAt,
			UpdatedAt: settings.UpdatedAt,
		}
	}

	// 2. Users list from Headscale API
	client, err := h.getClient()
	if err == nil && client != nil {
		if users, apiErr := client.ListUsers(); apiErr == nil {
			backup.Users = users
		}
	}

	// 3. ACL Policy
	cmd := exec.Command("headscale", "--config", ConfigPath, "policy", "show")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if cmd.Run() == nil {
		backup.Policy = stdout.String()
	}

	// 4. Audit logs
	if auditLogs, err := h.getAuditLogs(); err == nil {
		backup.AuditLogs = auditLogs
	}

	// Serialize
	jsonData, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		h.renderToast(w, "Failed to generate backup: "+err.Error(), "error")
		return
	}

	filename := fmt.Sprintf("headcontrol-backup-%s.json", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(jsonData)))
	w.Write(jsonData)

	h.LogAuditEvent(r, "Export Backup", "Downloaded system backup file")
}

// ImportBackup handles uploading and restoring a backup file
func (h *Handler) ImportBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		h.renderToast(w, "Failed to parse upload: "+err.Error(), "error")
		return
	}

	file, _, err := r.FormFile("backup_file")
	if err != nil {
		h.renderToast(w, "No file uploaded.", "error")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		h.renderToast(w, "Failed to read file.", "error")
		return
	}

	var backup model.BackupData
	if err := json.Unmarshal(data, &backup); err != nil {
		h.renderToast(w, "Invalid backup file format: "+err.Error(), "error")
		return
	}

	// Restore local settings (only BaseURL — never overwrite API key from backup since it's masked)
	if backup.Settings != nil && backup.Settings.BaseURL != "" {
		existing, _ := h.store.GetSettings()
		if existing != nil {
			// Only update BaseURL, keep existing API key
			h.store.SaveSettings(backup.Settings.BaseURL, existing.APIKey)
		}
	}

	h.LogAuditEvent(r, "Import Backup", fmt.Sprintf("Restored backup from %s (version %s)", backup.ExportedAt, backup.Version))
	h.renderToast(w, fmt.Sprintf("Backup restored successfully! (exported: %s)", backup.ExportedAt), "success")
}
