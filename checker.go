package main

import (
	"context"
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
	UserAgent          = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
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

// resolveHost resolves hostname to IP addresses with retries, public DNS fallback, and www fallback
func resolveHost(host string, timeout time.Duration) ([]string, string, error) {
	h := strings.TrimSpace(host)
	// Strip port if present
	if strings.Contains(h, ":") {
		sh, _, splitErr := net.SplitHostPort(h)
		if splitErr == nil {
			h = sh
		}
	}

	// Raw IP address check
	if ip := net.ParseIP(h); ip != nil {
		return []string{ip.String()}, h, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 1. Try system resolver
	resolvedIPs, err := net.DefaultResolver.LookupIP(ctx, "ip", h)
	if err == nil && len(resolvedIPs) > 0 {
		var res []string
		for _, ip := range resolvedIPs {
			res = append(res, ip.String())
		}
		return res, h, nil
	}

	// 2. Fallback to Google Public DNS (8.8.8.8:53)
	pubResolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 2 * time.Second}
			return d.DialContext(ctx, "udp", "8.8.8.8:53")
		},
	}
	resolvedIPs, pubErr := pubResolver.LookupIP(ctx, "ip", h)
	if pubErr == nil && len(resolvedIPs) > 0 {
		var res []string
		for _, ip := range resolvedIPs {
			res = append(res, ip.String())
		}
		return res, h, nil
	}

	// 3. WWW fallback: If h does not start with "www.", try "www." + h
	lower := strings.ToLower(h)
	if !strings.HasPrefix(lower, "www.") && strings.Count(h, ".") >= 1 {
		wwwHost := "www." + h
		resolvedIPs, wwwErr := pubResolver.LookupIP(ctx, "ip", wwwHost)
		if wwwErr == nil && len(resolvedIPs) > 0 {
			var res []string
			for _, ip := range resolvedIPs {
				res = append(res, ip.String())
			}
			return res, wwwHost, nil
		}
	} else if strings.HasPrefix(lower, "www.") {
		nonWwwHost := strings.TrimPrefix(h, "www.")
		resolvedIPs, nonWwwErr := pubResolver.LookupIP(ctx, "ip", nonWwwHost)
		if nonWwwErr == nil && len(resolvedIPs) > 0 {
			var res []string
			for _, ip := range resolvedIPs {
				res = append(res, ip.String())
			}
			return res, nonWwwHost, nil
		}
	}

	return nil, h, err
}

// CheckSingle performs a comprehensive health check on a single Record
func (ce *CheckEngine) CheckSingle(rec Record) CheckResult {
	start := time.Now()
	res := CheckResult{
		Record:    rec,
		CheckedAt: start,
		Status:    StatusDead,
		Category:  "Không hợp lệ",
	}

	if rec.TargetType == "EMPTY" {
		res.Status = StatusDead
		res.Category = "Không hợp lệ"
		res.StatusText = "Không có domain"
		res.Details = "Hàng trống, không có thông tin tên miền hoặc website"
		res.ResponseTime = 0
		res.ResponseMs = 0
		return res
	}

	if rec.TargetType == "INVALID" {
		res.Status = StatusDead
		res.Category = "Không hợp lệ"
		res.StatusText = "Tên miền sai"
		res.Details = "Tên miền không hợp lệ (thiếu phần đuôi mở rộng .vn, .com...)"
		res.ResponseTime = 0
		res.ResponseMs = 0
		return res
	}

	target := strings.TrimSpace(rec.CleanTarget)
	target = strings.TrimRight(target, "/")

	// Determine host and explicit URL if given
	var host string
	var explicitScheme string
	var port string

	if strings.HasPrefix(strings.ToLower(target), "http://") {
		explicitScheme = "http"
		u, err := url.Parse(target)
		if err == nil {
			host = u.Hostname()
			port = u.Port()
		} else {
			host = strings.TrimPrefix(target, "http://")
		}
	} else if strings.HasPrefix(strings.ToLower(target), "https://") {
		explicitScheme = "https"
		u, err := url.Parse(target)
		if err == nil {
			host = u.Hostname()
			port = u.Port()
		} else {
			host = strings.TrimPrefix(target, "https://")
		}
	} else {
		if strings.Contains(target, "/") {
			parts := strings.Split(target, "/")
			host = parts[0]
		} else {
			host = target
		}
		if strings.Contains(host, ":") {
			h, p, err := net.SplitHostPort(host)
			if err == nil {
				host = h
				port = p
			}
		}
	}

	// 1. DNS Resolution Pre-check
	dnsTimeout := 4 * time.Second
	if ce.Timeout < dnsTimeout {
		dnsTimeout = ce.Timeout
	}
	ips, resolvedHost, err := resolveHost(host, dnsTimeout)
	if err != nil {
		res.Status = StatusDNSError
		res.Category = "Hết hạn / Lỗi DNS"
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "no such host") || strings.Contains(errStr, "nxdomain") {
			res.StatusText = "Hết hạn / Chưa ĐK (NXDOMAIN)"
			res.Details = "Tên miền không tồn tại trên DNS toàn cầu (chưa đăng ký, đã hết hạn hoặc xóa DNS)"
		} else {
			res.StatusText = "Lỗi DNS (Không phân giải)"
			res.Details = fmt.Sprintf("Không phân giải được tên miền: %v", err)
		}
		res.ResponseTime = time.Since(start)
		res.ResponseMs = res.ResponseTime.Milliseconds()
		return res
	}

	if len(ips) > 0 {
		res.IP = ips[0]
	}

	host = resolvedHost
	hostAndPort := host
	if port != "" {
		hostAndPort = net.JoinHostPort(host, port)
	}

	// 2. HTTP/HTTPS probe
	ce.probeSmart(target, hostAndPort, explicitScheme, &res, start)

	res.ResponseTime = time.Since(start)
	res.ResponseMs = res.ResponseTime.Milliseconds()
	return res
}

func (res *CheckResult) applyProbe(probe CheckResult) {
	res.Status = probe.Status
	res.Category = probe.Category
	res.StatusCode = probe.StatusCode
	res.StatusText = probe.StatusText
	res.Details = probe.Details
	res.FinalURL = probe.FinalURL
	res.Protocol = probe.Protocol
	res.SSLValid = probe.SSLValid
	res.SSLError = probe.SSLError
}

func (ce *CheckEngine) probeSmart(target string, hostAndPort string, explicitScheme string, res *CheckResult, _ time.Time) {
	// Case 1: Custom non-standard port specified (e.g. :8443)
	if strings.Contains(hostAndPort, ":") {
		scheme := explicitScheme
		if scheme == "" {
			scheme = "https"
		}
		ce.probeProtocol(scheme+"://"+hostAndPort, scheme == "https", res)
		if res.Status == StatusDead || res.Status == StatusUnreachable {
			altScheme := "http"
			if scheme == "http" {
				altScheme = "https"
			}
			var altRes CheckResult
			ce.probeProtocol(altScheme+"://"+hostAndPort, altScheme == "https", &altRes)
			if altRes.Status == StatusOK || altRes.Status == StatusNoSSL || altRes.Status == StatusWarning {
				res.applyProbe(altRes)
			}
		}
		return
	}

	// Case 2: Explicitly specified HTTP scheme
	if explicitScheme == "http" {
		httpURL := "http://" + hostAndPort
		ce.probeProtocol(httpURL, false, res)

		if res.Status == StatusOK {
			if strings.HasPrefix(strings.ToLower(res.FinalURL), "https://") {
				if res.SSLValid {
					res.Status = StatusOK
					res.Category = "Hoạt động tốt"
					res.StatusText = fmt.Sprintf("Hoạt động tốt (%d)", res.StatusCode)
					res.Details = "Tự động chuyển hướng sang HTTPS bảo mật"
				} else {
					res.Status = StatusNoSSL
					res.Category = "Không có SSL"
					res.StatusText = "Lỗi SSL (Chuyển hướng HTTPS)"
					res.Details = fmt.Sprintf("Chuyển hướng HTTPS nhưng chứng chỉ SSL bị lỗi: %s", res.SSLError)
				}
			} else {
				res.Status = StatusNoSSL
				res.Category = "Không có SSL"
				res.StatusText = fmt.Sprintf("Chưa có SSL (%d)", res.StatusCode)
				res.Details = "Website hoạt động bình thường qua HTTP (Chưa cài đặt chứng chỉ SSL/HTTPS)"
				res.Protocol = "HTTP"
				res.SSLValid = false
			}
			return
		}

		// If HTTP failed, fallback to HTTPS
		if res.Status == StatusUnreachable || res.Status == StatusDead {
			httpsURL := "https://" + hostAndPort
			var httpsRes CheckResult
			ce.probeProtocol(httpsURL, true, &httpsRes)
			if httpsRes.Status == StatusOK || httpsRes.Status == StatusNoSSL {
				res.applyProbe(httpsRes)
				return
			}
		}
		return
	}

	// Case 3: Default (bare domain or https): Probe HTTPS first, then fallback to HTTP
	httpsURL := "https://" + hostAndPort
	ce.probeProtocol(httpsURL, true, res)

	// If HTTPS works perfectly with valid SSL, we are done
	if res.Status == StatusOK {
		return
	}

	// If HTTPS works with SSL issue (expired/self-signed/untrusted), we are done (classified as NO_SSL)
	if res.Status == StatusNoSSL {
		return
	}

	// If HTTPS failed (connection refused, timeout, protocol error), fallback to HTTP
	httpURL := "http://" + hostAndPort
	var httpRes CheckResult
	ce.probeProtocol(httpURL, false, &httpRes)

	if httpRes.Status == StatusOK {
		res.applyProbe(httpRes)
		if strings.HasPrefix(strings.ToLower(httpRes.FinalURL), "https://") {
			if httpRes.SSLValid {
				res.Category = "Hoạt động tốt"
				res.Details = "Tự động chuyển hướng sang HTTPS bảo mật"
			} else {
				res.Status = StatusNoSSL
				res.Category = "Không có SSL"
				res.StatusText = "Lỗi SSL (Chuyển hướng HTTPS)"
				res.Details = fmt.Sprintf("Chuyển hướng HTTPS nhưng SSL bị lỗi: %s", httpRes.SSLError)
			}
		} else {
			res.Status = StatusNoSSL
			res.Category = "Không có SSL"
			res.StatusText = fmt.Sprintf("Chưa có SSL (%d)", httpRes.StatusCode)
			res.Details = "Website hoạt động bình thường qua HTTP (Không hỗ trợ HTTPS)"
			res.Protocol = "HTTP"
			res.SSLValid = false
		}
		return
	}

	if httpRes.Status == StatusWarning {
		res.applyProbe(httpRes)
		return
	}
}

func (ce *CheckEngine) probeProtocol(rawURL string, isHTTPS bool, res *CheckResult) {
	client := &http.Client{
		Timeout: ce.Timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
				MinVersion:         tls.VersionTLS10,
				Renegotiation:      tls.RenegotiateFreelyAsClient,
			},
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
			setStandardHeaders(req)
			return nil
		},
	}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		res.Status = StatusDead
		res.Category = "Không hợp lệ"
		res.StatusText = "URL không hợp lệ"
		res.Details = err.Error()
		return
	}
	setStandardHeaders(req)

	resp, err := client.Do(req)
	if err != nil {
		errStr := err.Error()

		// If SSL error on HTTPS, retry with InsecureSkipVerify to check if web is alive
		if isHTTPS && isSSLError(err) {
			insecureClient := &http.Client{
				Timeout: ce.Timeout,
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
						MinVersion:         tls.VersionTLS10,
						Renegotiation:      tls.RenegotiateFreelyAsClient,
					},
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
					setStandardHeaders(req)
					return nil
				},
			}

			insecureReq, reqErr := http.NewRequest("GET", rawURL, nil)
			if reqErr == nil {
				setStandardHeaders(insecureReq)
				insecureResp, inErr := insecureClient.Do(insecureReq)
				if inErr == nil {
					defer insecureResp.Body.Close()
					res.Status = StatusNoSSL
					res.Category = "Không có SSL"
					res.StatusCode = insecureResp.StatusCode
					res.StatusText = fmt.Sprintf("Lỗi SSL (%d)", insecureResp.StatusCode)
					res.SSLError = simplifySSLError(errStr)
					res.SSLValid = false
					res.FinalURL = insecureResp.Request.URL.String()
					res.Protocol = "HTTPS (Insecure)"
					res.Details = fmt.Sprintf("Website hoạt động bình thường nhưng SSL bị lỗi: %s (Trình duyệt sẽ hiện cảnh báo bảo mật)", simplifySSLError(errStr))
					return
				}
			}
		}

		// Connection Failure diagnosis
		res.Status = StatusUnreachable
		res.Category = "Không thể truy cập"
		lowerErr := strings.ToLower(errStr)
		if strings.Contains(lowerErr, "context deadline exceeded") || strings.Contains(lowerErr, "timeout") {
			res.StatusText = "Quá thời gian phản hồi (Timeout)"
			res.Details = "Máy chủ phản hồi quá lâu (>8s) hoặc bị tường lửa chặn (Too long to respond)"
		} else if strings.Contains(lowerErr, "connection refused") || strings.Contains(lowerErr, "actively refused") {
			res.StatusText = "Từ chối kết nối (Refused)"
			res.Details = "Cổng dịch vụ 80/443 không mở hoặc web server tắt (Refused to connect)"
		} else if strings.Contains(lowerErr, "connection reset") || strings.Contains(lowerErr, "forcibly closed") {
			res.StatusText = "Mất kết nối (Reset)"
			res.Details = "Máy chủ ngắt kết nối giữa chừng (Connection reset by peer)"
		} else if strings.Contains(lowerErr, "no route to host") || strings.Contains(lowerErr, "network is unreachable") {
			res.StatusText = "Không thể tiếp cận mạng"
			res.Details = "Không tìm thấy đường truyền mạng tới máy chủ (Network unreachable)"
		} else if strings.Contains(lowerErr, "redirect") {
			res.Status = StatusWarning
			res.Category = "Bị chặn / Khác"
			res.StatusText = "Lặp chuyển hướng"
			res.Details = "Website bị vòng lặp chuyển hướng quá 10 lần (Redirect loop)"
		} else {
			res.StatusText = "Không thể kết nối"
			res.Details = errStr
		}
		return
	}
	defer resp.Body.Close()

	res.StatusCode = resp.StatusCode
	res.FinalURL = resp.Request.URL.String()
	res.Protocol = strings.ToUpper(resp.Request.URL.Scheme)
	res.SSLValid = strings.HasPrefix(strings.ToLower(res.FinalURL), "https://")

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		res.Status = StatusOK
		res.Category = "Hoạt động tốt"
		res.StatusText = fmt.Sprintf("Hoạt động tốt (%d)", resp.StatusCode)
		res.Details = "Website phản hồi nhanh và bình thường"
	} else if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		res.Status = StatusOK
		res.Category = "Hoạt động tốt"
		res.StatusText = fmt.Sprintf("Chuyển hướng (%d)", resp.StatusCode)
		res.Details = fmt.Sprintf("Chuyển hướng đến: %s", res.FinalURL)
	} else if resp.StatusCode == 401 || resp.StatusCode == 403 {
		res.Status = StatusWarning
		res.Category = "Bị chặn / Khác"
		res.StatusText = fmt.Sprintf("Bị chặn/Khóa (%d)", resp.StatusCode)
		res.Details = "Máy chủ còn hoạt động nhưng chặn truy cập (WAF/Cloudflare hoặc phân quyền)"
	} else if resp.StatusCode == 404 {
		res.Status = StatusWarning
		res.Category = "Bị chặn / Khác"
		res.StatusText = "404 Not Found"
		res.Details = "Trang web không tồn tại hoặc đã bị xóa"
	} else if resp.StatusCode >= 500 {
		res.Status = StatusUnreachable
		res.Category = "Không thể truy cập"
		res.StatusText = fmt.Sprintf("Lỗi máy chủ (%d)", resp.StatusCode)
		res.Details = "Máy chủ nội bộ gặp sự cố (500/502/503/504 Bad Gateway / Service Unavailable)"
	} else {
		res.Status = StatusWarning
		res.Category = "Bị chặn / Khác"
		res.StatusText = fmt.Sprintf("Mã phản hồi %d", resp.StatusCode)
		res.Details = fmt.Sprintf("Máy chủ phản hồi mã %d", resp.StatusCode)
	}
}

func setStandardHeaders(req *http.Request) {
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "vi-VN,vi;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Ch-Ua", `"Chromium";v="124", "Google Chrome";v="124", "Not-A.Brand";v="99"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
}

func isSSLError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "certificate") ||
		strings.Contains(errStr, "x509") ||
		strings.Contains(errStr, "tls") ||
		strings.Contains(errStr, "ssl") ||
		strings.Contains(errStr, "handshake")
}

func simplifySSLError(errStr string) string {
	lower := strings.ToLower(errStr)
	if strings.Contains(lower, "expired") {
		return "Chứng chỉ SSL đã hết hạn"
	}
	if strings.Contains(lower, "untrusted") || strings.Contains(lower, "unknown authority") || strings.Contains(lower, "self signed") {
		return "Chứng chỉ tự ký hoặc tổ chức phát hành không tin cậy"
	}
	if strings.Contains(lower, "valid for") || strings.Contains(lower, "common name") {
		return "Tên miền không khớp với chứng chỉ SSL"
	}
	if strings.Contains(lower, "handshake failure") {
		return "Lỗi bắt tay SSL (Giao thức/Mã hóa TLS cũ)"
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

	for i, r := range records {
		jobs <- job{index: i, record: r}
	}
	close(jobs)

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
		case StatusNoSSL:
			stats.NoSSLCount++
			stats.WarnCount++
		case StatusUnreachable:
			stats.UnreachableCount++
			stats.DeadCount++
		case StatusDNSError:
			stats.DNSErrorCount++
			stats.DeadCount++
		case StatusWarning:
			stats.OtherCount++
			stats.WarnCount++
		case StatusDead:
			stats.OtherCount++
			stats.DeadCount++
		default:
			stats.OtherCount++
		}
		totalMs += r.ResponseMs
	}

	if stats.Total > 0 {
		stats.AverageMs = totalMs / int64(stats.Total)
	}

	return stats
}
