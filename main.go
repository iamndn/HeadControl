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

	http.HandleFunc("/setup", h.SetupPage)
	http.HandleFunc("/api/test-connection", h.TestConnection)
	http.HandleFunc("/api/save-settings", h.SaveSettings)

	http.HandleFunc("/", h.RequireSetup(h.DashboardPage))
	http.HandleFunc("/users", h.RequireSetup(h.UsersPage))
	http.HandleFunc("/nodes", h.RequireSetup(h.NodesPage))
	http.HandleFunc("/settings", h.RequireSetup(h.SettingsPage))
	http.HandleFunc("/policy", h.RequireSetup(h.GetPolicyHandler))
	http.HandleFunc("/policy/save", h.RequireSetup(h.SavePolicyHandler))
	http.HandleFunc("/routes", h.RequireSetup(h.RoutesPage))

	http.HandleFunc("/dashboard/summary", h.RequireSetup(h.DashboardSummary))
	http.HandleFunc("/users/table", h.RequireSetup(h.UsersTable))
	http.HandleFunc("/nodes/table", h.RequireSetup(h.NodesTable))
	http.HandleFunc("/nodes/detail", h.RequireSetup(h.NodeDetail))
	http.HandleFunc("/routes/table", h.RequireSetup(h.RoutesTable))
	http.HandleFunc("/nodes/{id}/edit", h.RequireSetup(h.EditNodeNameForm))
	http.HandleFunc("/nodes/{id}/rename", h.RequireSetup(h.RenameNodeInline))
	http.HandleFunc("/keys", h.RequireSetup(h.PreAuthKeysPage))
	http.HandleFunc("/keys/table", h.RequireSetup(h.PreAuthKeysTable))
	http.HandleFunc("/logs", h.RequireSetup(h.LogsPage))
	http.HandleFunc("/logs/raw", h.RequireSetup(h.LogsRaw))

	http.HandleFunc("/api/users/create", h.RequireSetup(h.CreateUser))
	http.HandleFunc("/api/users/rename", h.RequireSetup(h.RenameUser))
	http.HandleFunc("/api/users/delete", h.RequireSetup(h.DeleteUser))

	http.HandleFunc("/api/nodes/rename", h.RequireSetup(h.RenameNode))
	http.HandleFunc("/api/nodes/expire", h.RequireSetup(h.ExpireNode))
	http.HandleFunc("/api/nodes/delete", h.RequireSetup(h.DeleteNode))
	http.HandleFunc("/api/nodes/tags", h.RequireSetup(h.SetNodeTags))
	http.HandleFunc("/api/nodes/routes", h.RequireSetup(h.SetNodeRoutes))

	http.HandleFunc("/api/update-settings", h.RequireSetup(h.UpdateSettings))
	http.HandleFunc("/api/routes/approve", h.RequireSetup(h.ApproveRoute))
	http.HandleFunc("/api/routes/reject", h.RequireSetup(h.RejectRoute))
	http.HandleFunc("/api/keys/create", h.RequireSetup(h.CreatePreAuthKey))
	http.HandleFunc("/api/keys/expire", h.RequireSetup(h.ExpirePreAuthKey))

	log.Printf("HeadControl starting on http://localhost:%s", *port)
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		log.Fatalf("server: %v", err)
	}
}
