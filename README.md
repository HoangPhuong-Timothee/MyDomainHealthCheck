# HƯỚNG DẪN SỬ DỤNG TOOL HEALTH CHECK DOMAIN & WEBSITE

Tool kiểm tra trạng thái hoạt động tên miền và website hàng loạt, tự động lọc và xuất danh sách các domain bị hỏng / chết / lỗi DNS / sập server ra file **Excel (.xlsx)** và **Text (.txt)**.

Được viết bằng **Go (Golang)** và đã đóng gói thành file thực thi độc lập **`domain-health-check.exe`** (không cần cài đặt Python, runtime hay cấu hình môi trường phức tạp).

---

## 1. Nâng cấp mới: Tự động hỏi chế độ & Quản lý thư mục báo cáo

* **Không cần gõ lệnh phức tạp**: Chỉ cần nhấp đúp chuột vào file `domain-health-check.exe`, hệ thống sẽ hỏi bạn muốn dùng giao diện nào ngay từ đầu:
  * **Chọn 1 (hoặc ấn Enter)**: Tự động bật giao diện Web UI đồ họa và mở trình duyệt web.
  * **Chọn 2**: Chạy trực tiếp tại cửa sổ console (CLI).
  * **Chọn 3**: Mở nhanh thư mục chứa các file báo cáo đã lưu.
* **Lưu gọn gàng vào thư mục `reports/`**:
  * Toàn bộ kết quả sau khi chạy không còn nằm rải rác ngoài thư mục gốc, mà được tự động gom vào folder con **`reports/`**.
* **Nguyên tắc đặt tên file ngắn gọn theo Ngày - Giờ**:
  * File Excel tổng hợp & domain hỏng: `baocao_YYYYMMDD_HHMMSS.xlsx` (Ví dụ: `baocao_20261111_090007.xlsx`)
  * File danh sách text: `hong_YYYYMMDD_HHMMSS.txt` (Ví dụ: `hong_20261111_090007.txt`)
  * Dễ nhìn, phân loại theo thời gian thực hiện, không bị trùng lặp hay ghi đè.
* **Nút bấm 1-click mở thư mục báo cáo**:
  * Trên **Web UI**: Có nút **`[ 📂 Mở Thư Mục Báo Cáo ]`** để tự động bật Windows File Explorer trỏ thẳng vào folder `reports/`.
  * Trên **CLI**: Sau khi check xong, chỉ cần ấn `Enter` hoặc gõ `y` là Windows File Explorer tự động mở thư mục ra.

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
