package main

import (
	"regexp"
	"strings"
)

var (
	// Regex matching standard domain pattern: e.g. example.com, thpt-tanan.edu.vn
	domainRegex = regexp.MustCompile(`(?i)\b([a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)*(?:\.[a-z]{2,}))\b`)

	// Regex matching URLs: http:// or https://
	urlRegex = regexp.MustCompile(`(?i)https?://[^\s"'<>]+`)

	// Regex matching User: <domain>
	userRegex = regexp.MustCompile(`(?i)User:\s*([^\s\r\n]+)`)

	// Regex matching URL: <url>
	urlPrefixRegex = regexp.MustCompile(`(?i)URL:\s*([^\s\r\n]+)`)
)

// ExtractTarget extracts the most suitable website domain or URL from raw cell text
func ExtractTarget(raw string) (target string, targetType string, note string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "EMPTY", "Không có thông tin website"
	}

	// Case 1: Multi-line credential / Plesk block
	if strings.Contains(raw, "\n") || strings.Contains(raw, "User:") || strings.Contains(raw, "URL:") || strings.Contains(raw, "FTP") {
		// Look for User: <domain>
		if userMatches := userRegex.FindStringSubmatch(raw); len(userMatches) > 1 {
			userVal := strings.TrimSpace(userMatches[1])
			if strings.Contains(userVal, ".") && !strings.Contains(userVal, "/") {
				return userVal, "DOMAIN", "Trích xuất từ trường User: trong khối cấu hình"
			}
		}

		// Look for standard domain in the block (prioritize .vn, .edu.vn, .com.vn, .com)
		allDomains := domainRegex.FindAllString(raw, -1)
		for _, d := range allDomains {
			lower := strings.ToLower(d)
			if strings.HasSuffix(lower, ".vn") || strings.HasSuffix(lower, ".com") || strings.HasSuffix(lower, ".net") || strings.HasSuffix(lower, ".org") {
				return d, "DOMAIN", "Trích xuất tên miền từ khối văn bản"
			}
		}

		// Look for URL: <url>
		if urlMatches := urlPrefixRegex.FindStringSubmatch(raw); len(urlMatches) > 1 {
			urlVal := strings.TrimSpace(urlMatches[1])
			return strings.TrimRight(urlVal, "/"), "URL", "Trích xuất URL quản trị/web từ khối cấu hình"
		}

		// Look for any http/https URL
		if urlMatch := urlRegex.FindString(raw); urlMatch != "" {
			return strings.TrimRight(urlMatch, "/"), "URL", "Trích xuất link URL"
		}

		// Look for typo domains with underscore (e.g. thcsthuthua_edu_vn)
		lines := strings.Split(raw, "\n")
		for _, l := range lines {
			lTrim := strings.TrimSpace(l)
			if strings.Contains(lTrim, "_edu_vn") || strings.Contains(lTrim, "_com_vn") || strings.Contains(lTrim, "_vn") {
				fixed := strings.ReplaceAll(lTrim, "_edu_vn", ".edu.vn")
				fixed = strings.ReplaceAll(fixed, "_com_vn", ".com.vn")
				fixed = strings.ReplaceAll(fixed, "_vn", ".vn")
				return fixed, "DOMAIN", "Đã tự động sửa dấu gạch dưới thành dấu chấm (từ: " + lTrim + ")"
			}
		}
	}

	// Clean single-line text
	cleaned := strings.TrimSpace(raw)
	// Remove trailing slashes and spaces (e.g. "domain.vn /", "domain.vn/")
	cleaned = strings.TrimRight(cleaned, "/ ")
	cleaned = strings.TrimLeft(cleaned, " ")

	// If there are still newlines, take the first line that looks like a target
	if strings.Contains(cleaned, "\n") {
		for _, line := range strings.Split(cleaned, "\n") {
			l := strings.TrimSpace(line)
			if l != "" && (strings.Contains(l, ".") || strings.Contains(l, "_vn")) {
				cleaned = l
				break
			}
		}
	}

	// Check for underscore replacement typo (e.g. "thcsthuthua_edu_vn")
	if strings.Contains(cleaned, "_edu_vn") || strings.Contains(cleaned, "_com_vn") || strings.Contains(cleaned, "_vn") {
		fixed := strings.ReplaceAll(cleaned, "_edu_vn", ".edu.vn")
		fixed = strings.ReplaceAll(fixed, "_com_vn", ".com.vn")
		fixed = strings.ReplaceAll(fixed, "_vn", ".vn")
		return fixed, "DOMAIN", "Đã tự động sửa dấu gạch dưới thành dấu chấm (từ: " + cleaned + ")"
	}

	// If starts with http:// or https://
	if strings.HasPrefix(strings.ToLower(cleaned), "http://") || strings.HasPrefix(strings.ToLower(cleaned), "https://") {
		return cleaned, "URL", ""
	}

	// Check if it's a domain with at least one dot
	if strings.Contains(cleaned, ".") {
		// Valid domain syntax
		if domainRegex.MatchString(cleaned) {
			return cleaned, "DOMAIN", ""
		}
	}

	// String has no dot (e.g. "thptcanduoc", "maugiaomylecd")
	return cleaned, "INVALID", "Tên miền không hợp lệ (thiếu phần đuôi mở rộng như .vn, .com, .edu.vn)"
}
