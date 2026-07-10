package handler

import (
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const sessionCookieName = "hc_session"
const sessionDuration = 7 * 24 * time.Hour // 7 days

// LoginPage renders the login form
func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
	// If no admins exist yet, redirect to register
	if !h.store.HasAdmins() {
		http.Redirect(w, r, "/register", http.StatusFound)
		return
	}

	// If already logged in, redirect to dashboard
	if sess := h.getSession(r); sess != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	h.renderAuthPage(w, "login.html", nil)
}

// RegisterPage renders the first-admin registration form
func (h *Handler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	// Only allow registration if no admins exist
	if h.store.HasAdmins() {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	h.renderAuthPage(w, "register.html", nil)
}

// HandleLogin processes login form submission
func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	if username == "" || password == "" {
		h.renderAuthPage(w, "login.html", map[string]interface{}{
			"Error":    "Username and password are required.",
			"Username": username,
		})
		return
	}

	admin, err := h.store.GetAdminByUsername(username)
	if err != nil || admin == nil {
		h.renderAuthPage(w, "login.html", map[string]interface{}{
			"Error":    "Invalid username or password.",
			"Username": username,
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err != nil {
		h.renderAuthPage(w, "login.html", map[string]interface{}{
			"Error":    "Invalid username or password.",
			"Username": username,
		})
		return
	}

	// Create session
	token, err := h.store.CreateSession(admin.ID, sessionDuration)
	if err != nil {
		h.renderAuthPage(w, "login.html", map[string]interface{}{
			"Error": "Failed to create session. Please try again.",
		})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(sessionDuration.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/", http.StatusFound)
}

// HandleRegister processes first-admin registration
func (h *Handler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Only allow if no admins exist
	if h.store.HasAdmins() {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	if username == "" || password == "" {
		h.renderAuthPage(w, "register.html", map[string]interface{}{
			"Error":    "Username and password are required.",
			"Username": username,
		})
		return
	}

	if len(username) < 3 {
		h.renderAuthPage(w, "register.html", map[string]interface{}{
			"Error":    "Username must be at least 3 characters.",
			"Username": username,
		})
		return
	}

	if len(password) < 6 {
		h.renderAuthPage(w, "register.html", map[string]interface{}{
			"Error":    "Password must be at least 6 characters.",
			"Username": username,
		})
		return
	}

	if password != confirmPassword {
		h.renderAuthPage(w, "register.html", map[string]interface{}{
			"Error":    "Passwords do not match.",
			"Username": username,
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		h.renderAuthPage(w, "register.html", map[string]interface{}{
			"Error": "Internal error. Please try again.",
		})
		return
	}

	if err := h.store.CreateAdmin(username, string(hash)); err != nil {
		h.renderAuthPage(w, "register.html", map[string]interface{}{
			"Error":    "Failed to create admin account: " + err.Error(),
			"Username": username,
		})
		return
	}

	// Auto-login after registration
	admin, _ := h.store.GetAdminByUsername(username)
	if admin != nil {
		token, err := h.store.CreateSession(admin.ID, sessionDuration)
		if err == nil {
			http.SetCookie(w, &http.Cookie{
				Name:     sessionCookieName,
				Value:    token,
				Path:     "/",
				MaxAge:   int(sessionDuration.Seconds()),
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
		}
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

// HandleLogout clears the session
func (h *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		h.store.DeleteSession(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	http.Redirect(w, r, "/login", http.StatusFound)
}

// RequireAuth middleware checks for valid session
func (h *Handler) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If no admins exist, allow access (will be redirected to register via setup flow)
		if !h.store.HasAdmins() {
			next(w, r)
			return
		}

		sess := h.getSession(r)
		if sess == nil {
			if h.isHTMX(r) {
				w.Header().Set("HX-Redirect", "/login")
				w.WriteHeader(200)
			} else {
				http.Redirect(w, r, "/login", http.StatusFound)
			}
			return
		}

		next(w, r)
	}
}

// getSession retrieves the current session from cookie
func (h *Handler) getSession(r *http.Request) *struct{ Username string } {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil
	}

	sess, err := h.store.GetSession(cookie.Value)
	if err != nil || sess == nil {
		return nil
	}

	return &struct{ Username string }{Username: sess.Username}
}

// GetCurrentUsername returns the logged-in admin username for templates
func (h *Handler) GetCurrentUsername(r *http.Request) string {
	sess := h.getSession(r)
	if sess != nil {
		return sess.Username
	}
	return ""
}

// renderAuthPage renders a standalone auth page (no sidebar layout)
func (h *Handler) renderAuthPage(w http.ResponseWriter, name string, data map[string]interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if data == nil {
		data = map[string]interface{}{}
	}
	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		// Fallback: render inline HTML if template fails
		errMsg := ""
		if e, ok := data["Error"]; ok {
			errMsg = fmt.Sprintf(`<div style="color:#ef4444;margin-bottom:16px;font-weight:600;">%s</div>`, html.EscapeString(e.(string)))
		}
		w.Write([]byte(fmt.Sprintf(`<!DOCTYPE html><html><head><title>HeadControl</title></head><body style="display:flex;justify-content:center;align-items:center;min-height:100vh;background:#1a1a2e;font-family:sans-serif;">
			<div style="background:white;padding:40px;border-radius:8px;width:400px;">
				<h2>HeadControl</h2>%s<p>Template error: %s</p>
			</div></body></html>`, errMsg, html.EscapeString(err.Error()))))
	}
}
