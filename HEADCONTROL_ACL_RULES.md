# Quy tắc Phát triển cho AI Agent (DEVELOPMENT_RULES.md)

## 1. Nguyên tắc An toàn Hệ thống (Security Rules)
- **Cấm Command Injection**: Khi gọi lệnh shell qua `os/exec`, TUYỆT ĐỐI KHÔNG sử dụng cơ chế cộng chuỗi thủ công. Bắt buộc khai báo tách biệt dưới dạng mảng tham số: `exec.Command("headscale", "--config", "/etc/headscale/config.yaml", "policy", "set", "-f", "-")`.
- **Cấm ghi file tạm**: Không tạo file cấu hình JSON tạm thời trên ổ đĩa VPS rồi gọi CLI. Việc này gây thắt nút cổ chai I/O. Bắt buộc dùng luồng dữ liệu nhớ đệm Memory Pipe (`cmd.StdinPipe()`).

## 2. Tiêu chuẩn giao diện HTMX (Frontend Rules)
- **Không Tải Lại Toàn Trang (No Full Page Reloads)**: Mọi tương tác lưu chính sách phải trả về các thẻ HTML thành phần (`<div>`, `<p>`). Sử dụng cơ chế hoán đổi cục bộ của HTMX (`hx-swap="innerHTML"`) để hiển thị thông báo.
- **Xử lý Textarea của Trình soạn thảo**: Do Ace Editor hoạt động trên một thẻ `div` giả lập, bạn phải luôn gọi hàm `updateTextarea()` dính liền với sự kiện `onsubmit` của thẻ `<form>` để đồng bộ giá trị sang ô text ẩn trước khi gửi HTTP Request đi.

## 3. Quy ước về Database
- Không dùng code Go mở trực tiếp file SQLite `/var/lib/headscale/db.sqlite` để thao tác bảng chính sách. Bắt buộc phải thông qua lớp CLI wrapper của `headscale` (như đã định nghĩa ở phần Architecture) để đảm bảo đồng bộ bộ nhớ đệm an toàn.