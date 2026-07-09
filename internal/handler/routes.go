package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"headcontrol/internal/model"
	"net/http"
	"os/exec"
)

// RoutesPage handles rendering of the routes tab page or content.
func (h *Handler) RoutesPage(w http.ResponseWriter, r *http.Request) {
	routes, err := h.listRoutes()
	if err != nil {
		h.renderPageWithError(w, r, "Routes", "routes", err.Error())
		return
	}

	h.renderPage(w, r, "routes", map[string]interface{}{
		"Title":      "Routes",
		"ActivePage": "routes",
		"Routes":     routes,
	})
}

// RoutesTable handles HTMX partial refresh for routes.
func (h *Handler) RoutesTable(w http.ResponseWriter, r *http.Request) {
	routes, err := h.listRoutes()
	if err != nil {
		h.renderPartialError(w, err.Error())
		return
	}

	h.render(w, "routes-content.html", map[string]interface{}{
		"Title":      "Routes",
		"ActivePage": "routes",
		"Routes":     routes,
	})
}

// ApproveRoute handles POST /api/routes/approve
func (h *Handler) ApproveRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	routeIDStr := r.FormValue("id")
	if routeIDStr == "" {
		h.renderToast(w, "Route ID is required.", "error")
		return
	}

	// CLI execution - Safe variadic arguments to prevent injection
	cmd := exec.Command("headscale", "--config", ConfigPath, "routes", "enable", "-r", routeIDStr)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		h.renderToast(w, fmt.Sprintf("Failed to approve route: %s", stderr.String()), "error")
		return
	}

	h.renderToast(w, "Route approved successfully!", "success")
}

// RejectRoute handles POST /api/routes/reject (disable route)
func (h *Handler) RejectRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	routeIDStr := r.FormValue("id")
	if routeIDStr == "" {
		h.renderToast(w, "Route ID is required.", "error")
		return
	}

	// CLI execution - Safe variadic arguments to prevent injection
	cmd := exec.Command("headscale", "--config", ConfigPath, "routes", "disable", "-r", routeIDStr)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		h.renderToast(w, fmt.Sprintf("Failed to reject route: %s", stderr.String()), "error")
		return
	}

	h.renderToast(w, "Route rejected/disabled successfully!", "success")
}

// Helper to query routes via headscale CLI
func (h *Handler) listRoutes() ([]model.Route, error) {
	cmd := exec.Command("headscale", "--config", ConfigPath, "routes", "list", "-o", "json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("headscale routes list error: %s: %w", stderr.String(), err)
	}

	// Check if output is empty
	outputBytes := stdout.Bytes()
	if len(bytes.TrimSpace(outputBytes)) == 0 {
		return []model.Route{}, nil
	}

	var routes []model.Route
	if err := json.Unmarshal(outputBytes, &routes); err != nil {
		return nil, fmt.Errorf("failed to parse routes JSON: %w (raw: %s)", err, stdout.String())
	}

	return routes, nil
}
