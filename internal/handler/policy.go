package handler

import (
	"bytes"
	"io"
	"net/http"
	"os/exec"
)

const ConfigPath = "/etc/headscale/config.yaml"

// GetPolicyHandler trả về nội dung policy hiện tại trong DB
func (h *Handler) GetPolicyHandler(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("headscale", "--config", ConfigPath, "policy", "show")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	_ = cmd.Run()

	currentPolicy := stdout.String()
	if currentPolicy == "" {
		currentPolicy = "{\n  \"acls\": [\n    {\"action\": \"accept\", \"src\": [\"*\"], \"dst\": [\"*:*\"]}\n  ]\n}"
	}

	h.renderPage(w, r, "policy", map[string]interface{}{
		"Title":      "ACL Policy",
		"ActivePage": "policy",
		"Policy":     currentPolicy,
	})
}

// SavePolicyHandler nhận chuỗi JSON từ form và nạp vào stdin của headscale
func (h *Handler) SavePolicyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	policyContent := r.FormValue("policy_content")

	cmd := exec.Command("headscale", "--config", ConfigPath, "policy", "set", "-f", "-")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		w.Write([]byte("<div class='p-4 mb-4 text-sm text-red-800 bg-red-50 rounded-lg'>Lỗi Pipe nội bộ</div>"))
		return
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		w.Write([]byte("<div class='p-4 mb-4 text-sm text-red-800 bg-red-50 rounded-lg'>Lỗi khởi chạy tiến trình</div>"))
		return
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, policyContent)
	}()

	err = cmd.Wait()
	if err != nil {
		w.Write([]byte("<div class='p-4 mb-4 text-sm text-red-800 bg-red-50 rounded-lg whitespace-pre-wrap'>" + stderr.String() + "</div>"))
		return
	}

	h.LogAuditEvent(r, "Update ACL Policy", "Saved new ACL configuration rules")
	w.Write([]byte("<div class='p-4 mb-4 text-sm text-green-800 bg-green-50 rounded-lg'>🛡️ Đã áp dụng chính sách bảo mật mới thành công!</div>"))
}
