package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"headcontrol/internal/model"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// PreAuthKeysPage renders the split-pane Pre-Auth Keys page
func (h *Handler) PreAuthKeysPage(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil || client == nil {
		h.renderPageWithError(w, r, "Pre-Auth Keys", "keys", "Failed to load settings.")
		return
	}

	users, apiErr := client.ListUsers()
	if apiErr != nil {
		h.renderPageWithError(w, r, "Pre-Auth Keys", "keys", apiErr.Error())
		return
	}

	// Select first user's ID by default if available
	var defaultUser string
	if len(users) > 0 {
		defaultUser = users[0].ID
	}

	var keys []model.PreAuthKey
	if defaultUser != "" {
		keys, _ = h.listKeys(defaultUser)
	}

	h.renderPage(w, r, "keys", map[string]interface{}{
		"Title":        "Pre-Auth Keys",
		"ActivePage":   "keys",
		"Users":        users,
		"SelectedUser": defaultUser,
		"Keys":         keys,
	})
}

// PreAuthKeysTable handles partial updates for listing keys for a specific user
func (h *Handler) PreAuthKeysTable(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user")
	if user == "" {
		h.renderPartialError(w, "User is required")
		return
	}

	keys, err := h.listKeys(user)
	if err != nil {
		h.renderPartialError(w, err.Error())
		return
	}

	h.render(w, "keys-table.html", map[string]interface{}{
		"Keys":         keys,
		"SelectedUser": user,
	})
}

// CreatePreAuthKey handles POST /api/keys/create
func (h *Handler) CreatePreAuthKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := r.FormValue("user")
	reusable := r.FormValue("reusable") == "true"
	expirationStr := r.FormValue("expiration")
	tagsCSV := r.FormValue("tags")

	if user == "" {
		h.renderToast(w, "User is required.", "error")
		return
	}

	// Validate duration
	duration, err := time.ParseDuration(expirationStr)
	if err != nil {
		h.renderToast(w, "Invalid expiration format: "+err.Error(), "error")
		return
	}

	client, err := h.getClient()
	if err != nil || client == nil {
		h.renderToast(w, "Failed to load settings.", "error")
		return
	}

	// Resolve user identifier to CLI-compatible User ID
	users, apiErr := client.ListUsers()
	if apiErr != nil {
		h.renderToast(w, apiErr.Error(), "error")
		return
	}

	var targetUser *model.User
	for i := range users {
		if users[i].ID == user || users[i].Name == user {
			targetUser = &users[i]
			break
		}
	}

	if targetUser == nil {
		h.renderToast(w, "User not found.", "error")
		return
	}

	// CLI Arguments using the numeric user ID to prevent strconv.ParseUint parsing errors
	args := []string{"--config", ConfigPath, "preauthkeys", "create", "-u", targetUser.ID, "--expiration", duration.String()}
	if reusable {
		args = append(args, "--reusable")
	}
	if tagsCSV != "" {
		var tagList []string
		for _, t := range strings.Split(tagsCSV, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tagList = append(tagList, t)
			}
		}
		if len(tagList) > 0 {
			args = append(args, "--tags", strings.Join(tagList, ","))
		}
	}

	// CLI execution using safe variadic arguments
	cmd := exec.Command("headscale", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		h.renderToast(w, fmt.Sprintf("Failed to create key: %s", stderr.String()), "error")
		return
	}

	newKey := strings.TrimSpace(stdout.String())
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	
	// Returns a success toast, renders the new key panel in the placeholder, and triggers a table refresh
	fmt.Fprintf(w, `
		<div class="toast toast-success" hx-swap-oob="beforeend:#toast-container">Pre-Auth Key created successfully!</div>
		<div id="new-key-display" class="settings-section" style="margin-top:16px; border-color:var(--green); background:var(--green-bg); position:relative;" hx-swap-oob="outerHTML">
			<h4 style="color:var(--success); text-transform:uppercase; font-size:0.875rem; margin-bottom:8px;">New Key Created</h4>
			<code style="font-family:'JetBrains Mono', monospace; font-size:1rem; font-weight:600; word-break:break-all; display:block; padding:8px; background:var(--surface); border:2px solid var(--border); border-radius:4px; margin-bottom:8px;">%s</code>
			<p class="text-muted" style="font-size:0.75rem;">Make sure to copy this key now. You won't be able to see it again.</p>
		</div>
		<script>HC.refreshKeys('%s');</script>
	`, html.EscapeString(newKey), html.EscapeString(targetUser.ID))
}

// ExpirePreAuthKey handles POST /api/keys/expire
func (h *Handler) ExpirePreAuthKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := r.FormValue("user")
	key := r.FormValue("key")

	if user == "" || key == "" {
		h.renderToast(w, "User and key are required.", "error")
		return
	}

	client, err := h.getClient()
	if err != nil || client == nil {
		h.renderToast(w, "Failed to load settings.", "error")
		return
	}

	// Resolve user identifier to numeric User ID
	users, apiErr := client.ListUsers()
	if apiErr != nil {
		h.renderToast(w, apiErr.Error(), "error")
		return
	}

	var targetUser *model.User
	for i := range users {
		if users[i].ID == user || users[i].Name == user {
			targetUser = &users[i]
			break
		}
	}

	if targetUser == nil {
		h.renderToast(w, "User not found.", "error")
		return
	}

	// CLI execution using key string as a positional argument (no -u or -k flags in newer Headscale versions)
	cmd := exec.Command("headscale", "--config", ConfigPath, "preauthkeys", "expire", key)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		h.renderToast(w, fmt.Sprintf("Failed to expire key: %s", stderr.String()), "error")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
		<div class="toast toast-success" hx-swap-oob="beforeend:#toast-container">Key expired successfully!</div>
		<script>HC.refreshKeys('%s');</script>
	`, html.EscapeString(targetUser.ID))
}

// Helper to query pre-auth keys via headscale CLI
func (h *Handler) listKeys(user string) ([]model.PreAuthKey, error) {
	client, err := h.getClient()
	if err != nil || client == nil {
		return nil, fmt.Errorf("failed to load settings")
	}

	// 1. Resolve user ID or Name to target User Name (string username) for list filtering
	users, apiErr := client.ListUsers()
	if apiErr != nil {
		return nil, apiErr
	}

	var targetUser *model.User
	for i := range users {
		if users[i].ID == user || users[i].Name == user {
			targetUser = &users[i]
			break
		}
	}

	if targetUser == nil {
		return nil, fmt.Errorf("user not found")
	}

	// 2. Fetch keys globally using CLI (does not support filter flags in newer versions)
	cmd := exec.Command("headscale", "--config", ConfigPath, "preauthkeys", "list", "-o", "json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("headscale preauthkeys list error: %s: %w", stderr.String(), err)
	}

	outputBytes := stdout.Bytes()
	if len(bytes.TrimSpace(outputBytes)) == 0 {
		return []model.PreAuthKey{}, nil
	}

	var rawKeys []struct {
		ID        interface{} `json:"id"`
		Key       string      `json:"key"`
		User      struct {
			ID   interface{} `json:"id"`
			Name string      `json:"name"`
		} `json:"user"`
		Reusable  bool `json:"reusable"`
		Ephemeral bool `json:"ephemeral"`
		Used      bool `json:"used"`
		Expiration struct {
			Seconds int64 `json:"seconds"`
			Nanos   int32 `json:"nanos"`
		} `json:"expiration"`
		CreatedAt struct {
			Seconds int64 `json:"seconds"`
			Nanos   int32 `json:"nanos"`
		} `json:"created_at"`
	}

	if err := json.Unmarshal(outputBytes, &rawKeys); err != nil {
		return nil, fmt.Errorf("failed to parse keys JSON: %w (raw: %s)", err, stdout.String())
	}

	var keys []model.PreAuthKey
	for _, raw := range rawKeys {
		var expStr string
		if raw.Expiration.Seconds > 0 {
			expStr = time.Unix(raw.Expiration.Seconds, int64(raw.Expiration.Nanos)).Format(time.RFC3339)
		}
		var createdStr string
		if raw.CreatedAt.Seconds > 0 {
			createdStr = time.Unix(raw.CreatedAt.Seconds, int64(raw.CreatedAt.Nanos)).Format(time.RFC3339)
		}

		keys = append(keys, model.PreAuthKey{
			ID:         fmt.Sprintf("%v", raw.ID),
			Key:        raw.Key,
			User:       raw.User.Name,
			Reusable:   raw.Reusable,
			Ephemeral:  raw.Ephemeral,
			Used:       raw.Used,
			Expiration: expStr,
			CreatedAt:  createdStr,
		})
	}

	// 3. Filter keys matching the resolved username (e.g. key.User == targetUser.Name)
	var filtered []model.PreAuthKey
	for _, k := range keys {
		if k.User == targetUser.Name {
			filtered = append(filtered, k)
		}
	}

	return filtered, nil
}
