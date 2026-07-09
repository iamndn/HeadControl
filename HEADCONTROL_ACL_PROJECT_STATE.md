
### 3. Tệp: `HEADCONTROL_ACL_PROJECT_STATE.md`
*Tệp tiến độ nguyên tử, AI sẽ tick chọn [x] sau khi code xong mỗi bước.*
```markdown
# Bảng Quản lý Tiến độ Dự án (PROJECT_STATE.md)

## 📊 Trạng thái Tổng quan
- **Tính năng**: Tích hợp ACL Policy Control Panel.
- **Tiến độ**: 100% Hoàn thành (Hoàn tất toàn bộ các pha).

## 📝 Nhật ký công việc (Checklist)

### PHA 1: Khảo sát Tuyến đường & Đăng ký Router
- [x] Quét thư mục gốc, xác định tệp chứa danh sách định tuyến chính (`routes.go`, `main.go` hoặc tương đương).
- [x] Khai báo 2 endpoints mới vào router chính của dự án:
  - `GET /policy` trỏ tới `handlers.GetPolicyHandler`
  - `POST /policy/save` trỏ tới `handlers.SavePolicyHandler`

### PHA 2: Xây dựng Bộ xử lý Backend (Golang logic)
- [x] Tạo mới/bổ sung tệp `handlers/policy.go`.
- [x] Triển khai hàm `GetPolicyHandler` đọc dữ liệu từ `os/exec` bốc lệnh `policy show`.
- [x] Triển khai hàm `SavePolicyHandler` liên kết luồng `StdinPipe` nạp dữ liệu `-f -`.
- [x] Viết hàm kiểm tra lỗi `Stderr` đầu ra để đóng gói thông báo lỗi cú pháp JSON cho HTMX.

### PHA 3: Thiết kế Giao diện UI (HTMX & Ace Editor)
- [x] Tìm tệp thanh Sidebar (`sidebar.html` hoặc `layout.html`), chèn thêm thẻ liên kết đến `/policy` sử dụng thuộc tính HTMX chuẩn của dự án.
- [x] Tạo mới tệp giao diện thành phần `templates/policy.html` dựa theo đặc tả thiết kế.
- [x] Đảm bảo script Ace Editor và hàm `updateTextarea()` đồng bộ chuẩn giá trị trước khi HTMX gọi lệnh POST.

### PHA 4: Kiểm thử & Phê duyệt (Testing & Verification)
- [x] Rà soát Code & Sanity Check các thuộc tính thẻ `<form>` (`hx-post`, `hx-target`, `onsubmit="updateTextarea()"`).
- [x] Rà soát cơ chế bắt lỗi `cmd.Stderr` ở Backend, đảm bảo đóng gói thẻ `<div>` cảnh báo màu đỏ/xanh chuẩn TailwindCSS/HTMX.
- [x] Phát hành kịch bản kiểm thử thủ công (Manual Test Script) gồm 3 bước.

