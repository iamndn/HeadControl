package main

import (
	"flag"
	"headcontrol/internal/handler"
	"headcontrol/internal/store"
	"log"
	"net/http"
)

func main() {
	port := flag.String("port", "8080", "Server port")
	dbPath := flag.String("db", "headcontrol.db", "SQLite database path")
	flag.Parse()

	s, err := store.New(*dbPath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer s.Close()

	h, err := handler.New(s, "templates")
	if err != nil {
		log.Fatalf("templates: %v", err)
	}

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Auth routes (no auth required) — handle both GET (page) and POST (form submit)
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.HandleLogin(w, r)
		} else {
			h.LoginPage(w, r)
		}
	})
	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.HandleRegister(w, r)
		} else {
			h.RegisterPage(w, r)
		}
	})
	http.HandleFunc("/logout", h.HandleLogout)

	// Setup (auth required but no settings needed)
	http.HandleFunc("/setup", h.RequireAuth(h.SetupPage))
	http.HandleFunc("/api/test-connection", h.RequireAuth(h.TestConnection))
	http.HandleFunc("/api/save-settings", h.RequireAuth(h.SaveSettings))

	// Pages (auth + setup required)
	http.HandleFunc("/", h.RequireAuth(h.RequireSetup(h.DashboardPage)))
	http.HandleFunc("/users", h.RequireAuth(h.RequireSetup(h.UsersPage)))
	http.HandleFunc("/nodes", h.RequireAuth(h.RequireSetup(h.NodesPage)))
	http.HandleFunc("/settings", h.RequireAuth(h.RequireSetup(h.SettingsPage)))
	http.HandleFunc("/policy", h.RequireAuth(h.RequireSetup(h.GetPolicyHandler)))
	http.HandleFunc("/policy/save", h.RequireAuth(h.RequireSetup(h.SavePolicyHandler)))
	http.HandleFunc("/routes", h.RequireAuth(h.RequireSetup(h.RoutesPage)))

	http.HandleFunc("/dashboard/summary", h.RequireAuth(h.RequireSetup(h.DashboardSummary)))
	http.HandleFunc("/users/table", h.RequireAuth(h.RequireSetup(h.UsersTable)))
	http.HandleFunc("/nodes/table", h.RequireAuth(h.RequireSetup(h.NodesTable)))
	http.HandleFunc("/nodes/detail", h.RequireAuth(h.RequireSetup(h.NodeDetail)))
	http.HandleFunc("/routes/table", h.RequireAuth(h.RequireSetup(h.RoutesTable)))
	http.HandleFunc("/nodes/edit", h.RequireAuth(h.RequireSetup(h.EditNodeNameForm)))
	http.HandleFunc("/nodes/rename-inline", h.RequireAuth(h.RequireSetup(h.RenameNodeInline)))
	http.HandleFunc("/keys", h.RequireAuth(h.RequireSetup(h.PreAuthKeysPage)))
	http.HandleFunc("/keys/table", h.RequireAuth(h.RequireSetup(h.PreAuthKeysTable)))
	http.HandleFunc("/logs", h.RequireAuth(h.RequireSetup(h.LogsPage)))
	http.HandleFunc("/logs/raw", h.RequireAuth(h.RequireSetup(h.LogsRaw)))
	http.HandleFunc("/logs/audit", h.RequireAuth(h.RequireSetup(h.LogsAudit)))

	// API endpoints (auth + setup required)
	http.HandleFunc("/api/users/create", h.RequireAuth(h.RequireSetup(h.CreateUser)))
	http.HandleFunc("/api/users/rename", h.RequireAuth(h.RequireSetup(h.RenameUser)))
	http.HandleFunc("/api/users/delete", h.RequireAuth(h.RequireSetup(h.DeleteUser)))

	http.HandleFunc("/api/nodes/rename", h.RequireAuth(h.RequireSetup(h.RenameNode)))
	http.HandleFunc("/api/nodes/configure", h.RequireAuth(h.RequireSetup(h.ConfigureNode)))
	http.HandleFunc("/api/nodes/expire", h.RequireAuth(h.RequireSetup(h.ExpireNode)))
	http.HandleFunc("/api/nodes/delete", h.RequireAuth(h.RequireSetup(h.DeleteNode)))
	http.HandleFunc("/api/nodes/tags", h.RequireAuth(h.RequireSetup(h.SetNodeTags)))
	http.HandleFunc("/api/nodes/routes", h.RequireAuth(h.RequireSetup(h.SetNodeRoutes)))

	http.HandleFunc("/api/update-settings", h.RequireAuth(h.RequireSetup(h.UpdateSettings)))
	http.HandleFunc("/api/routes/approve", h.RequireAuth(h.RequireSetup(h.ApproveRoute)))
	http.HandleFunc("/api/routes/reject", h.RequireAuth(h.RequireSetup(h.RejectRoute)))
	http.HandleFunc("/api/keys/create", h.RequireAuth(h.RequireSetup(h.CreatePreAuthKey)))
	http.HandleFunc("/api/keys/expire", h.RequireAuth(h.RequireSetup(h.ExpirePreAuthKey)))

	// Backup endpoints
	http.HandleFunc("/api/backup/export", h.RequireAuth(h.RequireSetup(h.ExportBackup)))
	http.HandleFunc("/api/backup/import", h.RequireAuth(h.RequireSetup(h.ImportBackup)))

	log.Printf("HeadControl starting on http://localhost:%s", *port)
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		log.Fatalf("server: %v", err)
	}
}
