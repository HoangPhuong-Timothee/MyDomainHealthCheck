package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	DefaultTimeout     = 8 * time.Second
	DefaultConcurrency = 25
	UserAgent          = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"
)

// CheckEngine orchestrates the concurrent health checking
type CheckEngine struct {
	Timeout     time.Duration
	Concurrency int
}

// NewCheckEngine creates a new checker with default or custom settings
func NewCheckEngine(timeout time.Duration, concurrency int) *CheckEngine {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	if concurrency <= 0 {
		concurrency = DefaultConcurrency
	}
	return &CheckEngine{
		Timeout:     timeout,
		Concurrency: concurrency,
	}
}

// CheckSingle performs a comprehensive health check on a single Record
func (ce *CheckEngine) CheckSingle(rec Record) CheckResult {
	start := time.Now()
	res := CheckResult{
		Record:    rec,
		CheckedAt: start,
		Status:    StatusDead,
	}

	if rec.TargetType == "EMPTY" {
		res.Status = StatusDead
		res.StatusText = "Không có domain"
		res.Details = "Hàng trống, không có thông tin tên miền hoặc website"
		res.ResponseTime = 0
		res.ResponseMs = 0
		return res
	}

	if rec.TargetType == "INVALID" {
		res.Status = StatusDead
		res.StatusText = "Tên miền sai"
		res.Details = "Tên miền không hợp lệ (thiếu phần đuôi mở rộng .vn, .com...)"
		res.ResponseTime = 0
		res.ResponseMs = 0
		return res
	}

	target := rec.CleanTarget

	// Determine host for DNS lookup
	var host string
	var initialURL string

	if strings.HasPrefix(strings.ToLower(target), "http://") || strings.HasPrefix(strings.ToLower(target), "https://") {
		initialURL = target
		u, err := url.Parse(target)
		if err == nil {
			host = u.Hostname()
		} else {
			host = target
		}
	} else {
		// Bare domain
		host = target
		if strings.Contains(host, "/") {
			parts := strings.Split(host, "/")
			host = parts[0]
		}
	}

	// 1. DNS Pre-check (if not a raw IP address)
	if net.ParseIP(host) == nil {
		ips, err := net.LookupIP(host)
		if err != nil {
			res.Status = StatusDead
			res.StatusText = "Lỗi DNS (Chưa ĐK / Hết hạn)"
			res.Details = fmt.Sprintf("Không phân giải được tên miền: %v", err)
			res.ResponseTime = time.Since(start)
			res.ResponseMs = res.ResponseTime.Milliseconds()
			return res
		}
		if len(ips) > 0 {
			res.IP = ips[0].String()
		}
	} else {
		res.IP = host
	}

	// 2. HTTP/HTTPS probe
	if initialURL != "" {
		// Specific URL given
		ce.probeURL(initialURL, &res, start)
	} else {
		// Domain given: Try HTTPS first, then fallback to HTTP
		httpsURL := "https://" + host
		ce.probeURL(httpsURL, &res, start)

		// If HTTPS failed (dead or error), try HTTP as fallback
		if res.Status == StatusDead {
			httpURL := "http://" + host
			httpRes := CheckResult{Record: rec, CheckedAt: start, IP: res.IP}
			ce.probeURL(httpURL, &httpRes, start)

			switch httpRes.Status {
			case StatusOK:
				res = httpRes
				res.Details = "Hoạt động qua HTTP (Không hỗ trợ HTTPS)"
			case StatusWarning:
				res = httpRes
			}
		}
	}

	res.ResponseTime = time.Since(start)
	res.ResponseMs = res.ResponseTime.Milliseconds()
	return res
}

func (ce *CheckEngine) probeURL(rawURL string, res *CheckResult, _ time.Time) {
	// First attempt with standard TLS
	client := &http.Client{
		Timeout: ce.Timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
			DialContext: (&net.Dialer{
				Timeout:   ce.Timeout,
				KeepAlive: 5 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   ce.Timeout,
			ResponseHeaderTimeout: ce.Timeout,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("quá nhiều lần chuyển hướng (redirect loop)")
			}
			return nil
		},
	}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		res.Status = StatusDead
		res.StatusText = "URL không hợp lệ"
		res.Details = err.Error()
		return
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		errStr := err.Error()

		// If it's an SSL/TLS error, probe with InsecureSkipVerify using a fresh Request
		if strings.Contains(errStr, "certificate") || strings.Contains(errStr, "x509") || strings.Contains(errStr, "tls") {
			insecureClient := &http.Client{
				Timeout: ce.Timeout,
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
					DialContext: (&net.Dialer{
						Timeout:   ce.Timeout,
						KeepAlive: 5 * time.Second,
					}).DialContext,
					TLSHandshakeTimeout:   ce.Timeout,
					ResponseHeaderTimeout: ce.Timeout,
				},
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					if len(via) >= 10 {
						return fmt.Errorf("quá nhiều lần chuyển hướng (redirect loop)")
					}
					return nil
				},
			}

			insecureReq, reqErr := http.NewRequest("GET", rawURL, nil)
			if reqErr == nil {
				insecureReq.Header.Set("User-Agent", UserAgent)
				insecureReq.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

				insecureResp, inErr := insecureClient.Do(insecureReq)
				if inErr == nil {
					defer insecureResp.Body.Close()
					res.Status = StatusWarning
					res.StatusCode = insecureResp.StatusCode
					res.StatusText = "Cảnh báo SSL"
					res.SSLError = errStr
					res.SSLValid = false
					res.FinalURL = insecureResp.Request.URL.String()
					res.Details = fmt.Sprintf("Web hoạt động (%d) nhưng SSL bị lỗi: %s", insecureResp.StatusCode, simplifySSLError(errStr))
					return
				}
			}
		}

		// Connection failure
		res.Status = StatusDead
		if strings.Contains(errStr, "context deadline exceeded") || strings.Contains(errStr, "timeout") {
			res.StatusText = "Timeout (>8s)"
			res.Details = "Server phản hồi quá chậm hoặc bị chặn bởi tường lửa"
		} else if strings.Contains(errStr, "connection refused") {
			res.StatusText = "Từ chối kết nối"
			res.Details = "Cổng dịch vụ 80/443 không mở hoặc web server đã tắt"
		} else {
			res.StatusText = "Không thể kết nối"
			res.Details = errStr
		}
		return
	}
	defer resp.Body.Close()

	res.StatusCode = resp.StatusCode
	res.FinalURL = resp.Request.URL.String()
	res.SSLValid = strings.HasPrefix(res.FinalURL, "https://")

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		res.Status = StatusOK
		res.StatusText = fmt.Sprintf("Hoạt động tốt (%d)", resp.StatusCode)
		res.Details = "Website phản hồi nhanh và bình thường"
	} else if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		res.Status = StatusOK
		res.StatusText = fmt.Sprintf("Chuyển hướng (%d)", resp.StatusCode)
		res.Details = fmt.Sprintf("Chuyển hướng đến: %s", res.FinalURL)
	} else if resp.StatusCode == 401 || resp.StatusCode == 403 {
		res.Status = StatusWarning
		res.StatusText = fmt.Sprintf("Bị chặn/Khóa (%d)", resp.StatusCode)
		res.Details = "Máy chủ còn hoạt động nhưng chặn truy cập (WAF/Cloudflare hoặc phân quyền)"
	} else if resp.StatusCode == 404 {
		res.Status = StatusDead
		res.StatusText = "404 Not Found"
		res.Details = "Trang web không tồn tại hoặc đã bị xóa"
	} else if resp.StatusCode >= 500 {
		res.Status = StatusDead
		res.StatusText = fmt.Sprintf("Lỗi máy chủ (%d)", resp.StatusCode)
		res.Details = "Máy chủ nội bộ gặp sự cố (500/502/503/504 Bad Gateway / Service Unavailable)"
	} else {
		res.Status = StatusWarning
		res.StatusText = fmt.Sprintf("Mã phản hồi %d", resp.StatusCode)
		res.Details = fmt.Sprintf("Máy chủ phản hồi mã %d", resp.StatusCode)
	}
}

func simplifySSLError(errStr string) string {
	if strings.Contains(errStr, "certificate has expired") {
		return "Chứng chỉ SSL đã hết hạn"
	}
	if strings.Contains(errStr, "certificate is not trusted") || strings.Contains(errStr, "unknown authority") {
		return "Chứng chỉ tự ký hoặc không đáng tin cậy"
	}
	if strings.Contains(errStr, "certificate is valid for") {
		return "Tên miền không khớp với chứng chỉ SSL"
	}
	return "Lỗi bảo mật SSL/TLS"
}

// ProgressCallback is called whenever a check finishes
type ProgressCallback func(completed int, total int, current CheckResult)

// Run checks all records concurrently with a worker pool
func (ce *CheckEngine) Run(records []Record, progress ProgressCallback) []CheckResult {
	total := len(records)
	results := make([]CheckResult, total)

	type job struct {
		index  int
		record Record
	}

	jobs := make(chan job, total)
	resChan := make(chan struct {
		index int
		res   CheckResult
	}, total)

	var wg sync.WaitGroup

	// Launch worker pool
	workerCount := ce.Concurrency
	if workerCount > total {
		workerCount = total
	}
	if workerCount <= 0 {
		workerCount = 1
	}

	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				r := ce.CheckSingle(j.record)
				resChan <- struct {
					index int
					res   CheckResult
				}{index: j.index, res: r}
			}
		}()
	}

	// Feed jobs
	for i, r := range records {
		jobs <- job{index: i, record: r}
	}
	close(jobs)

	// Wait in background and close resChan
	go func() {
		wg.Wait()
		close(resChan)
	}()

	completed := 0
	for item := range resChan {
		results[item.index] = item.res
		completed++
		if progress != nil {
			progress(completed, total, item.res)
		}
	}

	return results
}

// ComputeStats calculates summary statistics from results
func ComputeStats(results []CheckResult, duration time.Duration) SummaryStats {
	stats := SummaryStats{
		Total:       len(results),
		Duration:    duration,
		CheckedTime: time.Now(),
	}

	var totalMs int64
	for _, r := range results {
		switch r.Status {
		case StatusOK:
			stats.OKCount++
		case StatusDead:
			stats.DeadCount++
		case StatusWarning:
			stats.WarnCount++
		}
		totalMs += r.ResponseMs
	}

	if stats.Total > 0 {
		stats.AverageMs = totalMs / int64(stats.Total)
	}

	return stats
}
