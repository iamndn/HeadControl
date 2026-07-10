package handler

import (
	"bytes"
	"fmt"
	"headcontrol/internal/model"
	"html"
	"net/http"
	"os/exec"
	"strings"
)

func (h *Handler) NodesPage(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil || client == nil {
		h.renderPageWithError(w, r, "Nodes", "nodes", "Failed to load settings.")
		return
	}

	nodes, apiErr := client.ListNodes()
	if apiErr != nil {
		h.renderPageWithError(w, r, "Nodes", "nodes", apiErr.Error())
		return
	}

	h.renderPage(w, r, "nodes", map[string]interface{}{
		"Title":      "Nodes",
		"ActivePage": "nodes",
		"Nodes":      nodes,
	})
}

func (h *Handler) NodesTable(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil || client == nil {
		h.renderPartialError(w, "Failed to load settings.")
		return
	}

	nodes, apiErr := client.ListNodes()
	if apiErr != nil {
		h.renderPartialError(w, apiErr.Error())
		return
	}

	h.render(w, "nodes-content.html", map[string]interface{}{
		"Title":      "Nodes",
		"ActivePage": "nodes",
		"Nodes":      nodes,
	})
}

func (h *Handler) NodeDetail(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("id")
	if nodeID == "" {
		h.renderPartialError(w, "Node ID is required.")
		return
	}

	client, err := h.getClient()
	if err != nil || client == nil {
		h.renderPartialError(w, "Failed to load settings.")
		return
	}

	node, apiErr := client.GetNode(nodeID)
	if apiErr != nil {
		h.renderPartialError(w, apiErr.Error())
		return
	}

	h.render(w, "node-detail.html", map[string]interface{}{"Node": node})
}

func (h *Handler) RenameNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}

	nodeID := r.FormValue("nodeId")
	newName := r.FormValue("newName")

	if nodeID == "" || newName == "" {
		h.renderToast(w, "Node ID and new name are required.", "error")
		return
	}

	client, err := h.getClient()
	if err != nil || client == nil {
		h.renderToast(w, "Failed to load settings.", "error")
		return
	}

	if _, apiErr := client.RenameNode(nodeID, newName); apiErr != nil {
		h.renderToast(w, apiErr.Error(), "error")
		return
	}

	h.LogAuditEvent(r, "Rename Device", "Renamed device ID '"+nodeID+"' to '"+newName+"'")
	h.renderToast(w, "Node renamed to '"+newName+"' successfully!", "success")
}

func (h *Handler) ExpireNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}

	nodeID := r.FormValue("nodeId")
	if nodeID == "" {
		h.renderToast(w, "Node ID is required.", "error")
		return
	}

	client, err := h.getClient()
	if err != nil || client == nil {
		h.renderToast(w, "Failed to load settings.", "error")
		return
	}

	if _, apiErr := client.ExpireNode(nodeID); apiErr != nil {
		h.renderToast(w, apiErr.Error(), "error")
		return
	}

	h.LogAuditEvent(r, "Expire Device", "Expired device ID: "+nodeID)
	h.renderToast(w, "Node expired successfully!", "success")
}

func (h *Handler) DeleteNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}

	nodeID := r.FormValue("nodeId")
	if nodeID == "" {
		h.renderToast(w, "Node ID is required.", "error")
		return
	}

	client, err := h.getClient()
	if err != nil || client == nil {
		h.renderToast(w, "Failed to load settings.", "error")
		return
	}

	if apiErr := client.DeleteNode(nodeID); apiErr != nil {
		h.renderToast(w, apiErr.Error(), "error")
		return
	}

	h.LogAuditEvent(r, "Delete Device", "Deleted device ID: "+nodeID)
	h.renderToast(w, "Node deleted successfully!", "success")
}

func (h *Handler) SetNodeTags(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}

	nodeID := r.FormValue("nodeId")
	if nodeID == "" {
		h.renderToast(w, "Node ID is required.", "error")
		return
	}

	tags := splitCSV(r.FormValue("tags"))

	client, err := h.getClient()
	if err != nil || client == nil {
		h.renderToast(w, "Failed to load settings.", "error")
		return
	}

	if _, apiErr := client.SetNodeTags(nodeID, tags); apiErr != nil {
		h.renderToast(w, apiErr.Error(), "error")
		return
	}

	h.LogAuditEvent(r, "Update Tags", fmt.Sprintf("Updated device ID '%s' tags to: %s", nodeID, strings.Join(tags, ",")))
	h.renderToast(w, "Tags updated successfully!", "success")
}

func (h *Handler) SetNodeRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}

	nodeID := r.FormValue("nodeId")
	if nodeID == "" {
		h.renderToast(w, "Node ID is required.", "error")
		return
	}

	routes := splitCSV(r.FormValue("routes"))

	client, err := h.getClient()
	if err != nil || client == nil {
		h.renderToast(w, "Failed to load settings.", "error")
		return
	}

	if _, apiErr := client.SetApprovedRoutes(nodeID, routes); apiErr != nil {
		h.renderToast(w, apiErr.Error(), "error")
		return
	}

	h.renderToast(w, "Routes approved successfully!", "success")
}

func splitCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// EditNodeNameForm returns the HTMX input form for inline editing.
func (h *Handler) EditNodeNameForm(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("id")
	if nodeID == "" {
		http.Error(w, "Node ID is required", http.StatusBadRequest)
		return
	}

	client, err := h.getClient()
	if err != nil || client == nil {
		http.Error(w, "Failed to load settings", http.StatusInternalServerError)
		return
	}

	node, apiErr := client.GetNode(nodeID)
	if apiErr != nil {
		http.Error(w, apiErr.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
		<form hx-post="/nodes/rename-inline?id=%s" hx-target="#node-name-cell-%s" hx-swap="innerHTML" style="margin:0; display:flex; gap:8px; align-items:center;">
			<input type="text" name="newName" value="%s" class="form-input" style="padding:4px 8px; font-size:0.875rem; width:150px; border:2px solid var(--border); box-shadow:2px 2px 0 var(--border);" required autofocus onfocus="this.select()">
			<button type="submit" style="display:none;"></button>
		</form>
	`, nodeID, nodeID, html.EscapeString(node.GivenName))
}

// RenameNodeInline renames a node via headscale CLI and returns updated row/cell inner HTML.
func (h *Handler) RenameNodeInline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	nodeID := r.URL.Query().Get("id")
	newName := r.FormValue("newName")

	if nodeID == "" || newName == "" {
		http.Error(w, "Node ID and new name are required", http.StatusBadRequest)
		return
	}

	client, err := h.getClient()
	if err != nil || client == nil {
		http.Error(w, "Failed to load settings", http.StatusInternalServerError)
		return
	}

	node, apiErr := client.GetNode(nodeID)
	if apiErr != nil {
		http.Error(w, apiErr.Error(), http.StatusInternalServerError)
		return
	}

	// Safe variadic CLI command execution
	cmd := exec.Command("headscale", "--config", ConfigPath, "nodes", "rename", node.Name, newName)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := cmd.Run(); err != nil {
		// Revert to original text and display error toast
		nameSpan := ""
		if node.Name != "" && node.Name != node.GivenName {
			nameSpan = fmt.Sprintf(`<br><span class="text-muted" style="font-size:0.75rem;">%s</span>`, html.EscapeString(node.Name))
		}
		fmt.Fprintf(w, `<strong>%s</strong>%s
		<div class="toast toast-error" hx-swap-oob="beforeend:#toast-container">Failed to rename: %s</div>`,
			html.EscapeString(node.GivenName), nameSpan, html.EscapeString(strings.TrimSpace(stderr.String())))
		return
	}

	// Fetch updated node info
	updatedNode, apiErr := client.GetNode(nodeID)
	if apiErr != nil {
		updatedNode = &model.Node{ID: nodeID, GivenName: newName, Name: node.Name}
	}

	nameSpan := ""
	if updatedNode.Name != "" && updatedNode.Name != updatedNode.GivenName {
		nameSpan = fmt.Sprintf(`<br><span class="text-muted" style="font-size:0.75rem;">%s</span>`, html.EscapeString(updatedNode.Name))
	}

	h.LogAuditEvent(r, "Rename Device (Inline)", "Renamed device ID '"+nodeID+"' inline to '"+newName+"'")
	fmt.Fprintf(w, `<strong>%s</strong>%s
	<div class="toast toast-success" hx-swap-oob="beforeend:#toast-container">Device renamed to '%s' successfully!</div>`,
		html.EscapeString(updatedNode.GivenName), nameSpan, html.EscapeString(updatedNode.GivenName))
}

// ConfigureNode handles renaming a node and editing its tags in a single operation
func (h *Handler) ConfigureNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	nodeID := r.FormValue("nodeId")
	newName := r.FormValue("newName")
	tagsCSV := r.FormValue("tags")

	if nodeID == "" {
		h.renderToast(w, "Node ID is required.", "error")
		return
	}

	client, err := h.getClient()
	if err != nil || client == nil {
		h.renderToast(w, "Failed to load settings.", "error")
		return
	}

	node, apiErr := client.GetNode(nodeID)
	if apiErr != nil {
		h.renderToast(w, apiErr.Error(), "error")
		return
	}

	// 1. Rename node if name changed
	if newName != "" && newName != node.GivenName {
		if _, apiErr := client.RenameNode(nodeID, newName); apiErr != nil {
			h.renderToast(w, apiErr.Error(), "error")
			return
		}
	}

	// 2. Update tags
	var tagList []string
	for _, t := range strings.Split(tagsCSV, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			if !strings.HasPrefix(t, "tag:") {
				t = "tag:" + t
			}
			tagList = append(tagList, t)
		}
	}

	if _, apiErr := client.SetNodeTags(nodeID, tagList); apiErr != nil {
		h.renderToast(w, apiErr.Error(), "error")
		return
	}

	h.LogAuditEvent(r, "Configure Device", fmt.Sprintf("Configured device ID '%s': name='%s', tags='%s'", nodeID, newName, tagsCSV))
	h.renderToast(w, "Device configured successfully!", "success")
}
