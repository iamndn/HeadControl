# HeadControl

Minimal Headscale admin console and dashboard, built with Go, HTMX, and Ace Editor.

This project provides a lightweight, highly responsive, and beautiful dashboard to manage your self-hosted **Headscale** server. It relies on Go for the backend, HTMX for zero-reload reactive frontend components, and SQLite for configuration state. 

---

## 📊 Core Features

- **Dashboard Statistics**: Instant overview of registered users, total nodes, active/online nodes, and nodes nearing expiration.
- **User Management**: Easily create, rename, and delete Headscale users.
- **Node Management**: Check node details, rename, force key expiration, delete, configure ACL tags, and approve/enable routes.
- **🛡️ Integrated ACL Policy Control Panel**:
  - View and edit security policy JSONs natively using an integrated **Ace Editor** instance.
  - Interactive syntax validation and error tracking: passes output checks to catch JSON and rule errors instantly.
  - Premium **Dynamic Theme Syncing**: The Ace Editor automatically shifts its theme (`chrome` / `tomorrow_night`) to match the user's selected dashboard theme in real time.
  - Safe backend execution: Runs command piping directly via `StdinPipe()` (no temp files created, completely memory-buffered, and protected against shell-command injection).
- **16 Premium Color Themes**: Includes Light, Dark, Cyberpunk, Dracula, Nord, Synthwave, Terminal, and more.
- **Responsive Layout**: Designed to look premium across Desktop, Tablet, and Mobile devices.

---

## ⚙️ Architectural Highlights

To support Headscale servers running natively using `mode: database`, HeadControl interfaces with the system's local `headscale` daemon via secure `os/exec` wrappers:
- **Zero Disk Overhead**: Temporary policy files are never written to disk during saves. Instead, bytes are piped directly into the CLI's standard input stream (`StdinPipe`).
- **Strict Input Sanitization**: Commands are called with explicit argument slice definitions (`exec.Command("headscale", "--config", "/etc/headscale/config.yaml", ...)`) preventing any command injection vectors.
- **Stderr Capturing**: Headscale's validation checks and error messages are captured directly from Stderr and processed as HTMX-swapped warnings on the UI.

---

## 🛠️ Stack

- **Backend**: Go (Go 1.23+)
- **Frontend Interactivity**: [HTMX](https://htmx.org/) (Zero heavy JS framework overhead)
- **Code Editor**: [Ace Editor](https://ace.c9.io/) (Syntax highlighting, soft tabs, and dynamic theme switching)
- **Icons**: Lucide Icons (loaded via CDN)
- **Database**: SQLite3 (Local storage for HeadControl console configuration settings)

---

## 🚀 Requirements

- **Go 1.23** or later
- **GCC** (required by `go-sqlite3` driver via CGO)
- A running **Headscale** server with API key enabled (or local access for the CLI policies)

### GCC Installation on Windows
The `go-sqlite3` driver requires CGO compilation. On Windows, install [MSYS2](https://www.msys2.org/) or [TDM-GCC](https://jmeubank.github.io/tdm-gcc/), then verify that `gcc` is in your system's `PATH`.

---

## 💻 Setup & Run

1. **Clone the repository**:
   ```bash
   git clone https://github.com/iamndn/HeadControl.git
   cd HeadControl
   ```

2. **Install dependencies**:
   ```bash
   go mod tidy
   ```

3. **Build the binary**:
   ```bash
   go build -o headcontrol
   ```

4. **Run the server**:
   ```bash
   ./headcontrol
   ```
   By default, the server runs on `http://localhost:8080`.

### Command-line Parameters

| Flag | Default | Description |
|------|---------|-------------|
| `-port` | `8080` | Server listening port |
| `-db` | `headcontrol.db` | Path to HeadControl's SQLite configuration database |

Example custom startup:
```bash
./headcontrol -port 9000 -db /var/lib/headcontrol/headcontrol.db
```

---

## 🔑 Initial Configuration

1. Visit `http://localhost:8080` in your web browser.
2. You will be automatically redirected to the setup wizard.
3. Enter your Headscale server base URL (e.g. `https://headscale.yourdomain.com`).
4. Input your API key (generate one on your headscale machine with `headscale apikeys create`).
5. Click **Test Connection** to verify settings, then save to enter the console dashboard.

---

## 🧑‍💻 Development Guide

For automated builds and hot-reload upon saving files, install [Air](https://github.com/air-verse/air):
```bash
go install github.com/air-verse/air@latest
```
Run the development environment:
```bash
air
```
Hot-reload logic is configured inside `.air.toml`.

### Project Structure

```
HeadControl/
  ├── main.go                      # Entry point, HTTP route definition
  ├── internal/
  │     ├── handler/
  │     │     ├── handler.go       # Core Handler setup, template renderer, middleware
  │     │     ├── helpers.go       # Helper utilities, time formatters
  │     │     ├── setup.go         # Initialization page handlers
  │     │     ├── dashboard.go     # Dashboard stats handlers
  │     │     ├── users.go         # User administration handlers
  │     │     ├── nodes.go         # Node configuration & action handlers
  │     │     ├── settings.go      # Global settings page handlers
  │     │     └── policy.go        # ACL policy (CLI show/set) handlers
  │     ├── headscale/
  │     │     └── client.go        # Headscale REST API client functions
  │     ├── model/
  │     │     └── models.go        # Global structure structs
  │     └── store/
  │           └── store.go         # SQLite database persistence layer
  ├── templates/
  │     ├── layout/
  │     │     └── layout.html      # Outer wrapper page layout (Sidebar, themes)
  │     ├── pages/
  │     │     ├── dashboard.html   # Dashboard components
  │     │     ├── nodes.html       # Node listing tables & modal targets
  │     │     ├── settings.html    # Configuration form layout
  │     │     ├── setup.html       # Onboarding page
  │     │     ├── users.html       # User listing tables & creations
  │     │     └── policy.html      # Ace Editor & HTMX save policy form
  │     └── partials/              # HTMX components (tables, results, modals)
  └── static/
        ├── css/
        │    ├── app.css           # Global core layout styles
        │    └── theme/            # Theme color variables
        └── js/
             └── app.js            # Frontend logic (theme toggle, sidebar)
```

---

## 📄 License

This project is licensed under the MIT License. Feel free to use, modify, and distribute it.
