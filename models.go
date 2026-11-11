package main

import "time"

// HealthStatus represents the status of a domain
type HealthStatus string

const (
	StatusOK      HealthStatus = "OK"      // Website is working normally
	StatusWarning HealthStatus = "WARNING" // Website works but has SSL/403/Redirect issues
	StatusDead    HealthStatus = "DEAD"    // Website is broken, DNS fail, timeout, 5xx, 404
	StatusUnknown HealthStatus = "UNKNOWN" // Unchecked or empty

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
	Status       HealthStatus  `json:"status"`        // OK, WARNING, DEAD
	StatusCode   int           `json:"status_code"`   // HTTP status code (200, 404, 502, ...)
	StatusText   string        `json:"status_text"`   // Vietnamese status summary
	Details      string        `json:"details"`       // Detailed error or success explanation
	IP           string        `json:"ip"`            // Resolved IP address
	FinalURL     string        `json:"final_url"`     // Final URL after redirects
	Protocol     string        `json:"protocol"`      // HTTP, HTTPS
	SSLValid     bool          `json:"ssl_valid"`     // SSL certificate valid
	SSLError     string        `json:"ssl_error"`     // SSL certificate error if any
	ResponseTime time.Duration `json:"response_time"` // Response duration
	ResponseMs   int64         `json:"response_ms"`   // Duration in milliseconds
	CheckedAt    time.Time     `json:"checked_at"`    // Timestamp
}

// SummaryStats contains aggregated statistics
type SummaryStats struct {
	Total       int           `json:"total"`
	OKCount     int           `json:"ok_count"`
	DeadCount   int           `json:"dead_count"`
	WarnCount   int           `json:"warn_count"`
	Duration    time.Duration `json:"duration"`
	AverageMs   int64         `json:"average_ms"`
	CheckedTime time.Time     `json:"checked_time"`
}
