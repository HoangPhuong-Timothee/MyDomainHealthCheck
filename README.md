# HƯỚNG DẪN SỬ DỤNG TOOL HEALTH CHECK DOMAIN & WEBSITE

Tool kiểm tra trạng thái hoạt động tên miền và website hàng loạt, **tự động phân loại chính xác tình trạng sức khỏe website** (Hoạt động tốt, Không có SSL / Lỗi SSL, Không thể truy cập, Hết hạn domain / Trỏ sai DNS) và xuất báo cáo ra file **Excel (.xlsx)** đa sheet chuyên nghiệp và file **Text (.txt)** phân nhóm chi tiết.

Được viết bằng **Go (Golang)** và đã đóng gói thành file thực thi độc lập **`domain-health-check.exe`** (không cần cài đặt Python, runtime hay cấu hình môi trường phức tạp).

---

## 1. Hệ thống Phân loại Trạng thái Website Mới (Chuẩn xác 100%)

Phiên bản mới đã khắc phục triệt để tình trạng một số website truy cập bình thường nhưng bị báo hỏng nhờ cơ chế **Dual-protocol Fallback (HTTP & HTTPS)**, **TLS Renegotiation**, **Browser-grade Headers** và **Hệ thống phân loại 4 nhóm chính**:

| Nhóm Phân Loại | Mã Trạng Thái | Ý Nghĩa & Nhận Diện | Hành Động Đề Xuất |
| :--- | :---: | :--- | :--- |
| **Hoạt động tốt** | `OK` | Website phản hồi tốt (HTTP 200/3xx) với kết nối bảo mật HTTPS và chứng chỉ SSL hợp lệ. | Bình thường, không cần xử lý. |
| **Không có SSL / Lỗi SSL** | `NO_SSL` | **Website vẫn vào xem được bình thường**, nhưng:<br>• Chỉ chạy HTTP (port 80), chưa có HTTPS/SSL.<br>• Hoặc có HTTPS nhưng SSL bị hết hạn, sai tên miền hoặc tự ký. | Cài đặt chứng chỉ SSL hoặc gia hạn chứng chỉ (Let's Encrypt / Cloudflare). |
| **Không thể truy cập** | `UNREACHABLE` | Domain có phân giải IP nhưng không thể kết nối tới máy chủ:<br>• **Timeout**: Phản hồi quá lâu (>8s) / Too long to respond.<br>• **Refused**: Cổng 80/443 đóng, Web server nginx/apache đã tắt.<br>• **Reset / Network**: Mất kết nối giữa chừng.<br>• **Lỗi 5xx**: 500, 502 Bad Gateway, 503, 504. | Báo quản trị viên server khởi động lại web server hoặc mở port tường lửa. |
| **Hết hạn domain / Lỗi DNS** | `DNS_ERROR` | Không phân giải được IP:<br>• **NXDOMAIN**: Tên miền chưa đăng ký hoặc đã hết hạn bị thu hồi.<br>• Chưa tạo bản ghi A/CNAME hoặc máy chủ DNS lỗi. | Liên hệ nhà đăng ký tên miền để gia hạn hoặc cấu hình lại DNS. |
| **Bị chặn / Lỗi khác** | `WARNING` | • **403/401**: Bị tường lửa WAF / Cloudflare chặn bot hoặc yêu cầu quyền đăng nhập.<br>• **404 Not Found**: Đường dẫn không tồn tại.<br>• **INVALID / EMPTY**: Dòng trống hoặc tên miền sai cú pháp. | Kiểm tra đường dẫn URL hoặc cấu hình phân quyền web. |

---

## 2. Nâng cấp mới: Quản lý thư mục báo cáo & Giao diện chuyên nghiệp

* **Không báo hỏng nhầm**:
  * Website chỉ có HTTP hoặc có cảnh báo SSL tự ký được xếp riêng vào nhóm **"Không có SSL / Lỗi SSL"**, không còn bị đánh đồng với website chết hay sập.
  * Tự động thử cả 2 giao thức HTTPS và HTTP cùng cơ chế fallback sang `www.` nếu tên miền gốc không có bản ghi A.
* **Giao diện Web Dashboard trực quan**:
  * Bổ sung 6 thẻ thống kê số liệu theo thời gian thực (Tổng số, Hoạt động tốt, Không có SSL, Không thể truy cập, Lỗi DNS, Bị chặn).
  * 6 tab lọc thông minh: Bấm xem riêng từng nhóm chỉ với 1 click.
* **Báo cáo Excel đa sheet chuyên nghiệp**:
  * **Sheet 1: Hết Hạn & Lỗi DNS** (Ưu tiên số 1 - Cần gia hạn tên miền / trỏ DNS).
  * **Sheet 2: Không Thể Truy Cập** (Ưu tiên số 2 - Cần khởi động lại web server hoặc mở firewall).
  * **Sheet 3: Không Có SSL & Cảnh Báo** (Cần cài đặt hoặc gia hạn chứng chỉ SSL).
  * **Sheet 4: Hoạt Động Tốt** (Danh sách các website hoàn hảo).
  * **Sheet 5: Toàn Bộ Kết Quả** (Đầy đủ tất cả website kèm màu sắc trực quan).
  * **Sheet 6: Thống Kê Tổng Quan** (Bảng số liệu tổng hợp và tỉ lệ %).
* **Báo cáo Text (`hong_*.txt`) phân nhóm rõ ràng**:
  * Phân tách rành mạch theo từng nhóm nguyên nhân để kỹ thuật viên dễ dàng xử lý.

---

## 2. Hướng dẫn sử dụng chi tiết

### Cách 1: Chạy trực tiếp (Nhấp đúp chuột vào `domain-health-check.exe`)
1. Nhấp đúp chuột vào file **`domain-health-check.exe`**.
2. Màn hình xuất hiện thông báo:
   ```text
   ========================================================================
          TOOL KIỂM TRA SỨC KHỎE DOMAIN & WEBSITE (HEALTH CHECK TOOL)       
   ========================================================================
     Bạn muốn sử dụng giao diện nào?

     [1] Giao diện Đồ Họa Web UI (Khuyên dùng - Mở trực tiếp trên trình duyệt)
     [2] Giao diện Dòng Lệnh CLI (Chạy trực tiếp tại cửa sổ này)
     [3] Mở thư mục chứa các file báo cáo đã lưu ('reports')
     [4] Thoát
   ========================================================================
   -> Nhập lựa chọn [Mặc định: 1 - Bấm Enter để mở Web UI]: 
   ```
3. Bạn chỉ cần:
   * **Bấm phím Enter** (hoặc gõ `1`): Trình duyệt Chrome/Edge sẽ tự động mở trang Dashboard `http://localhost:8080`.
   * **Gõ 2**: Chuyển sang chế độ CLI để kéo thả file hoặc dán danh sách domain.
   * **Gõ 3**: Bật ngay thư mục `reports` để xem các file báo cáo trước đó.

---

### Cách 2: Sử dụng trên Giao diện Web UI (Dashboard)
Khi mở Web UI trên trình duyệt:
1. **Nạp danh sách**:
   * **Tab 1**: Kéo thả file Excel (`.xlsx`) hoặc `.csv` vào khung tải file. Hoặc bấm **"Dùng File Mẫu Của Bạn"** để thử nghiệm với file mẫu 220 website.
   * **Tab 2**: Dán trực tiếp danh sách domain copy từ Excel vào ô nhập liệu.
2. **Kiểm tra**:
   * Nhấn nút **`► Bắt Đầu Kiểm Tra`**.
   * Thanh tiến trình % sẽ chạy theo thời gian thực cùng bảng số liệu thống kê (Tổng số, Sống, Hỏng, Cảnh báo).
3. **Xem & Lọc kết quả**:
   * Bấm tab **`Chỉ Domain Hỏng`** để lọc ra danh sách sự cố.
   * Có thanh tìm kiếm nhanh theo tên trường học, bệnh viện, doanh nghiệp hoặc domain.
4. **Lấy file báo cáo**:
   * Bấm **`[ 📂 Mở Thư Mục Báo Cáo ]`** để mở ngay folder chứa file trên máy tính.
   * Bấm **`[ 📥 Tải File Excel ]`** để tải bản copy về máy.
   * Bấm **`[ 🔄 Check Lại Domain Hỏng ]`** nếu muốn kiểm tra lại riêng các website vừa bị lỗi.

---

### Cách 3: Sử dụng trên Giao diện Dòng Lệnh (CLI)
Khi chọn `[2] Giao diện Dòng Lệnh CLI`:
* Menu CLI cho phép:
  * **[1]**: Kéo thả file Excel / CSV vào cửa sổ console rồi ấn Enter.
  * **[2]**: Paste trực tiếp danh sách domain từ clipboard vào console, sau đó gõ `CHECK` hoặc ấn Enter 2 lần.
  * **[3]**: Chạy kiểm tra nhanh file mẫu `mau_danh_sach.xlsx`.
* Khi kiểm tra xong:
  * Hiển thị bảng tóm tắt và danh sách từng domain hỏng kèm lý do cụ thể.
  * Tự động lưu 2 file: `reports/baocao_YYYYMMDD_HHMMSS.xlsx` và `reports/hong_YYYYMMDD_HHMMSS.txt`.
  * Hỏi: *`Bạn có muốn MỞ THƯ MỤC BÁO CÁO trên máy tính không? (Y/n)`* $\rightarrow$ Chỉ cần ấn Enter để mở ngay folder.

---

## 3. Cấu trúc thư mục dự án

```text
ping-tool/
├── domain-health-check.exe   # File chạy duy nhất (không cần cài thêm gì)
├── mau_danh_sach.xlsx        # File Excel mẫu (220 website khách hàng)
├── mau_danh_sach.csv         # File CSV mẫu
├── HUONG_DAN_SU_DUNG.md      # Hướng dẫn chi tiết
│
├── reports/                  # THƯ MỤC CHỨA TẤT CẢ FILE BÁO CÁO
│   ├── baocao_20261111_090007.xlsx  # File Excel báo cáo đa sheet (đỏ/vàng/xanh)
│   └── hong_20261111_090007.txt     # File text liệt kê riêng các domain hỏng
│
└── src/ (Mã nguồn Go sạch sẽ, có thể tự biên dịch lại nếu cần)
    ├── main.go
    ├── cli.go
    ├── webui.go
    ├── excel.go
    ├── csv.go
    ├── checker.go
    ├── extractor.go
    └── web/index.html
```

---

## 4. Cách thức phân loại lỗi của Tool
* **`DEAD` (Bị hỏng / Không truy cập được)**:
  * **Lỗi DNS**: Tên miền chưa đăng ký, đã hết hạn hoặc chưa cấu hình bản ghi DNS (Lookup failed).
  * **Timeout**: Quá 8 giây không phản hồi (Server tắt hoặc chặn kết nối).
  * **Mã lỗi máy chủ**: HTTP 500, 502 Bad Gateway, 503 Service Unavailable, 504 Gateway Timeout.
  * **Mã lỗi 404**: Trang web không tồn tại hoặc đã bị xóa.
  * **Tên miền sai / Thiếu đuôi**: Ví dụ `thptcanduoc` (thiếu `.edu.vn` hoặc `.vn`).
* **`WARNING` (Cảnh báo)**:
  * **Lỗi chứng chỉ SSL**: Chứng chỉ hết hạn, tự ký, nhưng máy chủ web vẫn đang hoạt động.
  * **Mã lỗi 403 Forbidden**: Máy chủ hoạt động nhưng chặn bot hoặc cần đăng nhập (Cloudflare/WAF).
* **`OK` (Hoạt động tốt)**:
  * Phản hồi mã HTTP 200..399 nhanh chóng, ổn định qua HTTPS hoặc HTTP.
