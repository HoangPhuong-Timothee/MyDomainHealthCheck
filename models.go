package main

import "time"

// HealthStatus represents the status of a domain
type HealthStatus string

const (
	StatusOK          HealthStatus = "OK"          // Website hoạt động tốt (HTTPS/SSL chuẩn)
	StatusNoSSL       HealthStatus = "NO_SSL"      // Website không có SSL / Lỗi SSL (HTTP Only hoặc SSL cảnh báo)
	StatusUnreachable HealthStatus = "UNREACHABLE" // Website không thể truy cập (Timeout, Refused, 5xx)
	StatusDNSError    HealthStatus = "DNS_ERROR"   // Hết hạn domain / Lỗi phân giải DNS
	StatusWarning     HealthStatus = "WARNING"     // Bị chặn (403, 401), 404 hoặc cảnh báo khác
	StatusDead        HealthStatus = "DEAD"        // Không hợp lệ hoặc rỗng
	StatusUnknown     HealthStatus = "UNKNOWN"     // Chưa kiểm tra

	ReportsDir = "reports" // Dedicated folder for all health check reports
)

// Record represents a single row from input (Excel, CSV, or pasted)
type Record struct {
	Index       string `json:"index"`        // STT
	Unit        string `json:"unit"`         // Đơn vị (Thạnh Hóa, Tân An, ...)
	Type        string `json:"type"`         // Loại (Y tế, Giáo dục, Doanh nghiệp, ...)
	Customer    string `json:"customer"`     // Khách hàng
	RawWebsite  string `json:"raw_website"`  // Nguyên bản trong file
	CleanTarget string `json:"clean_target"` // Domain hoặc URL sau khi trích xuất
	TargetType  string `json:"target_type"`  // DOMAIN, URL, IP_PORT, INVALID, EMPTY
}

// CheckResult represents the outcome of a health check on a Record
type CheckResult struct {
	Record
	Status       HealthStatus  `json:"status"`        // OK, NO_SSL, UNREACHABLE, DNS_ERROR, WARNING, DEAD
	Category     string        `json:"category"`      // "Hoạt động tốt", "Không có SSL", "Không thể truy cập", "Hết hạn / Lỗi DNS", "Bị chặn / Khác"
	StatusCode   int           `json:"status_code"`   // HTTP status code (200, 404, 502, ...)
	StatusText   string        `json:"status_text"`   // Tiêu đề trạng thái tóm tắt tiếng Việt
	Details      string        `json:"details"`       // Chi tiết nguyên nhân / giải thích
	IP           string        `json:"ip"`            // IP phân giải được
	FinalURL     string        `json:"final_url"`     // URL cuối cùng sau redirect
	Protocol     string        `json:"protocol"`      // HTTP, HTTPS
	SSLValid     bool          `json:"ssl_valid"`     // Chứng chỉ SSL hợp lệ
	SSLError     string        `json:"ssl_error"`     // Lỗi SSL nếu có
	ResponseTime time.Duration `json:"response_time"` // Thời gian phản hồi
	ResponseMs   int64         `json:"response_ms"`   // Thời gian phản hồi (ms)
	CheckedAt    time.Time     `json:"checked_at"`    // Thời điểm kiểm tra
}

// SummaryStats contains aggregated statistics
type SummaryStats struct {
	Total            int           `json:"total"`
	OKCount          int           `json:"ok_count"`          // Hoạt động tốt
	NoSSLCount       int           `json:"no_ssl_count"`      // Không có SSL / Lỗi SSL
	UnreachableCount int           `json:"unreachable_count"` // Không thể truy cập (Timeout, Refused, 5xx)
	DNSErrorCount    int           `json:"dns_error_count"`   // Hết hạn / Lỗi DNS
	OtherCount       int           `json:"other_count"`       // Bị chặn 403, 404, Không hợp lệ
	DeadCount        int           `json:"dead_count"`        // Tổng số bị hỏng/lỗi kết nối (Unreachable + DNSError)
	WarnCount        int           `json:"warn_count"`        // Tổng cảnh báo (NoSSL + Other)
	Duration         time.Duration `json:"duration"`
	AverageMs        int64         `json:"average_ms"`
	CheckedTime      time.Time     `json:"checked_time"`
}
