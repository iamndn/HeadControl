package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"headcontrol/internal/model"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
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

	routeID, err := strconv.ParseUint(routeIDStr, 10, 64)
	if err != nil {
		h.renderToast(w, "Invalid route ID format.", "error")
		return
	}

	// 1. Fetch all routes
	routes, err := h.listRoutes()
	if err != nil {
		h.renderToast(w, "Failed to fetch routes: "+err.Error(), "error")
		return
	}

	// 2. Find target route
	var targetRoute *model.Route
	for i := range routes {
		if routes[i].ID == routeID {
			targetRoute = &routes[i]
			break
		}
	}

	if targetRoute == nil {
		h.renderToast(w, "Route not found.", "error")
		return
	}

	if targetRoute.Node == nil {
		h.renderToast(w, "Route node not found.", "error")
		return
	}

	nodeID := targetRoute.Node.ID

	// 3. Accumulate all currently enabled routes + target route for this node
	var approvedPrefixes []string
	approvedPrefixes = append(approvedPrefixes, targetRoute.Prefix)
	for _, rt := range routes {
		if rt.Node != nil && rt.Node.ID == nodeID && rt.Enabled && rt.ID != routeID {
			// Deduplicate just in case
			isDup := false
			for _, p := range approvedPrefixes {
				if p == rt.Prefix {
					isDup = true
					break
				}
			}
			if !isDup {
				approvedPrefixes = append(approvedPrefixes, rt.Prefix)
			}
		}
	}

	// 4. Call headscale nodes approve-routes
	prefixesStr := strings.Join(approvedPrefixes, ",")
	cmd := exec.Command("headscale", "--config", ConfigPath, "nodes", "approve-routes", "-i", nodeID, "-r", prefixesStr)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		h.renderToast(w, fmt.Sprintf("Failed to approve route: %s", stderr.String()), "error")
		return
	}

	h.LogAuditEvent(r, "Approve Route", fmt.Sprintf("Approved route ID '%d' (%s) for node ID '%s'", routeID, targetRoute.Prefix, nodeID))
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

	routeID, err := strconv.ParseUint(routeIDStr, 10, 64)
	if err != nil {
		h.renderToast(w, "Invalid route ID format.", "error")
		return
	}

	// 1. Fetch all routes
	routes, err := h.listRoutes()
	if err != nil {
		h.renderToast(w, "Failed to fetch routes: "+err.Error(), "error")
		return
	}

	// 2. Find target route
	var targetRoute *model.Route
	for i := range routes {
		if routes[i].ID == routeID {
			targetRoute = &routes[i]
			break
		}
	}

	if targetRoute == nil {
		h.renderToast(w, "Route not found.", "error")
		return
	}

	if targetRoute.Node == nil {
		h.renderToast(w, "Route node not found.", "error")
		return
	}

	nodeID := targetRoute.Node.ID

	// 3. Accumulate currently enabled routes EXCLUDING the rejected one
	var approvedPrefixes []string
	for _, rt := range routes {
		if rt.Node != nil && rt.Node.ID == nodeID && rt.Enabled && rt.ID != routeID {
			// Deduplicate
			isDup := false
			for _, p := range approvedPrefixes {
				if p == rt.Prefix {
					isDup = true
					break
				}
			}
			if !isDup {
				approvedPrefixes = append(approvedPrefixes, rt.Prefix)
			}
		}
	}

	// 4. Call headscale nodes approve-routes
	prefixesStr := strings.Join(approvedPrefixes, ",")
	cmd := exec.Command("headscale", "--config", ConfigPath, "nodes", "approve-routes", "-i", nodeID, "-r", prefixesStr)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		h.renderToast(w, fmt.Sprintf("Failed to reject route: %s", stderr.String()), "error")
		return
	}

	h.LogAuditEvent(r, "Reject Route", fmt.Sprintf("Rejected/Disabled route ID '%d' (%s) for node ID '%s'", routeID, targetRoute.Prefix, nodeID))
	h.renderToast(w, "Route rejected/disabled successfully!", "success")
}

// Helper to query routes via headscale CLI (using nodes list-routes)
func (h *Handler) listRoutes() ([]model.Route, error) {
	cmd := exec.Command("headscale", "--config", ConfigPath, "nodes", "list-routes", "-o", "json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("headscale nodes list-routes error: %s: %w", stderr.String(), err)
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
