# Đặc tả Kiến trúc kỹ thuật - Module ACL Policy

## 1. Luồng dữ liệu và Đóng gói Tiến trình (Process Piping)
Do Headscale vận hành ở chế độ `mode: database`, việc giao tiếp sẽ sử dụng binary `headscale` hệ thống thông qua `os/exec`. Module sử dụng cơ chế Stdin/Stdout Pipes để tránh việc ghi file tạm ra ổ đĩa, tối ưu hóa tốc độ và độ an toàn.

### Luồng đọc (GET /policy)
- Lệnh gọi: `headscale --config /etc/headscale/config.yaml policy show`
- Backend thu kết quả từ Stdout, nếu trống hoặc lỗi thì trả về chuỗi JSON rỗng mặc định `{}`.

### Luồng ghi (POST /policy/save)
- Lệnh gọi: `headscale --config /etc/headscale/config.yaml policy set -f -`
- Dữ liệu text JSON từ form gửi lên được ghi trực tiếp vào `cmd.StdinPipe()`.
- Đầu ra lỗi `Stderr` của tiến trình được bắt lại để trích xuất dòng báo lỗi cú pháp của Headscale (nếu có).

## 2. Bản mẫu cấu trúc Backend (Go Handler Template)
Ứng dụng sử dụng chuẩn `http.HandlerFunc` (hoặc Gin/Chi tùy theo file `routes.go`). Dưới đây là kiến trúc hàm logic lõi cần triển khai trong `handlers/policy.go`:

```go
package handlers

import (
	"bytes"
	"io"
	"net/http"
	"os/exec"
)

const ConfigPath = "/etc/headscale/config.yaml"

// GetPolicyHandler trả về nội dung policy hiện tại trong DB
func GetPolicyHandler(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("headscale", "--config", ConfigPath, "policy", "show")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	_ = cmd.Run() 

	currentPolicy := stdout.String()
	if currentPolicy == "" {
		currentPolicy = "{\n  \"acls\": [\n    {\"action\": \"accept\", \"src\": [\"*\"], \"dst\": [\"*:*\"]}\n  ]\n}"
	}

	// TODO: Nạp data JSON vào struct và render template HTML policy
}

// SavePolicyHandler nhận chuỗi JSON từ form và nạp vào stdin của headscale
func SavePolicyHandler(w http.ResponseWriter, r *http.Request) {
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

	w.Write([]byte("<div class='p-4 mb-4 text-sm text-green-800 bg-green-50 rounded-lg'>🛡️ Đã áp dụng chính sách bảo mật mới thành công!</div>"))
}