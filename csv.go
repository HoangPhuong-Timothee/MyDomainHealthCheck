package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// ParsePastedText parses raw text pasted from clipboard (single column or multiple columns)
func ParsePastedText(content string) ([]Record, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("nội dung trống")
	}

	lines := strings.Split(content, "\n")
	var records []Record
	stt := 1

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Check if it's tab-separated (direct copy from Excel rows)
		if strings.Contains(line, "\t") {
			parts := strings.Split(line, "\t")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}

			// If last part looks like website
			rec := Record{
				Index: strconv.Itoa(stt),
			}
			stt++

			if len(parts) >= 5 {
				rec.Index = parts[0]
				rec.Unit = parts[1]
				rec.Type = parts[2]
				rec.Customer = parts[3]
				rec.RawWebsite = parts[4]
			} else if len(parts) == 4 {
				rec.Unit = parts[0]
				rec.Type = parts[1]
				rec.Customer = parts[2]
				rec.RawWebsite = parts[3]
			} else if len(parts) == 2 {
				rec.Customer = parts[0]
				rec.RawWebsite = parts[1]
			} else {
				rec.RawWebsite = parts[len(parts)-1]
			}

			target, tType, _ := ExtractTarget(rec.RawWebsite)
			rec.CleanTarget = target
			rec.TargetType = tType
			records = append(records, rec)
			continue
		}

		// Single column paste (just domain or url)
		target, tType, _ := ExtractTarget(trimmed)
		records = append(records, Record{
			Index:       strconv.Itoa(stt),
			RawWebsite:  trimmed,
			CleanTarget: target,
			TargetType:  tType,
		})
		stt++
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("không tìm thấy dòng nào hợp lệ")
	}

	return records, nil
}

// ReadCSVFile reads and parses a CSV file with automatic header detection and BOM stripping
func ReadCSVFile(filePath string) ([]Record, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("không thể đọc file CSV: %w", err)
	}

	// Strip UTF-8 BOM if present
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))

	// Detect delimiter (comma or semicolon)
	delimiter := rune(',')
	firstLine := ""
	if idx := bytes.IndexByte(data, '\n'); idx != -1 {
		firstLine = string(data[:idx])
	} else {
		firstLine = string(data)
	}
	if strings.Count(firstLine, ";") > strings.Count(firstLine, ",") {
		delimiter = ';'
	}

	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = delimiter
	r.LazyQuotes = true
	r.FieldsPerRecord = -1 // Allow variable number of fields

	var records []Record
	headerMap := make(map[string]int)
	headerFound := false
	rowIdx := 0

	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Skip malformed row or continue
			continue
		}

		// Clean row items
		for i := range row {
			row[i] = strings.TrimSpace(row[i])
		}

		// Identify header
		if !headerFound {
			websiteCol := -1
			for colIdx, colVal := range row {
				cleanCol := strings.ToLower(colVal)
				if strings.Contains(cleanCol, "website") || strings.Contains(cleanCol, "domain") ||
					strings.Contains(cleanCol, "web") || strings.Contains(cleanCol, "tên miền") ||
					strings.Contains(cleanCol, "url") {
					websiteCol = colIdx
					break
				}
			}

			if websiteCol != -1 {
				headerFound = true
				for colIdx, colVal := range row {
					cleanCol := strings.ToLower(colVal)
					if strings.Contains(cleanCol, "website") || strings.Contains(cleanCol, "domain") ||
						strings.Contains(cleanCol, "web") || strings.Contains(cleanCol, "tên miền") || strings.Contains(cleanCol, "url") {
						headerMap["website"] = colIdx
					} else if strings.Contains(cleanCol, "khách hàng") || strings.Contains(cleanCol, "ten khach") {
						headerMap["customer"] = colIdx
					} else if strings.Contains(cleanCol, "đơn vị") || strings.Contains(cleanCol, "don vi") {
						headerMap["unit"] = colIdx
					} else if strings.Contains(cleanCol, "loại") || strings.Contains(cleanCol, "loai") {
						headerMap["type"] = colIdx
					} else if strings.Contains(cleanCol, "stt") || colIdx == 0 {
						headerMap["index"] = colIdx
					}
				}
				continue
			}
		}

		// Row processing
		rowIdx++
		rec := Record{
			Index: strconv.Itoa(rowIdx),
		}

		if headerFound {
			if idx, ok := headerMap["website"]; ok && idx < len(row) {
				rec.RawWebsite = row[idx]
			}
			if idx, ok := headerMap["customer"]; ok && idx < len(row) {
				rec.Customer = row[idx]
			}
			if idx, ok := headerMap["unit"]; ok && idx < len(row) {
				rec.Unit = row[idx]
			}
			if idx, ok := headerMap["type"]; ok && idx < len(row) {
				rec.Type = row[idx]
			}
			if idx, ok := headerMap["index"]; ok && idx < len(row) && row[idx] != "" {
				rec.Index = row[idx]
			}
		} else {
			// No header found, use positional heuristic
			if len(row) >= 5 {
				rec.Index = row[0]
				rec.Unit = row[1]
				rec.Type = row[2]
				rec.Customer = row[3]
				rec.RawWebsite = row[4]
			} else if len(row) == 1 {
				rec.RawWebsite = row[0]
			} else {
				rec.RawWebsite = row[len(row)-1]
			}
		}

		target, tType, _ := ExtractTarget(rec.RawWebsite)
		rec.CleanTarget = target
		rec.TargetType = tType
		records = append(records, rec)
	}

	return records, nil
}

// ExportTextSummary writes a plain text file containing only dead domains
func ExportTextSummary(filePath string, results []CheckResult) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# DANH SÁCH DOMAIN BỊ HỎNG / SẬP (Thời gian kiểm tra: %s)\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString("# -----------------------------------------------------------------------\n\n")

	deadCount := 0
	for _, r := range results {
		if r.Status == StatusDead {
			deadCount++
			customer := r.Customer
			if customer == "" {
				customer = "N/A"
			}
			target := r.CleanTarget
			if target == "" {
				target = r.RawWebsite
			}
			sb.WriteString(fmt.Sprintf("%s\t(KH: %s | Lỗi: %s - %s)\n", target, customer, r.StatusText, r.Details))
		}
	}

	sb.WriteString(fmt.Sprintf("\n# Tổng số domain hỏng: %d\n", deadCount))

	return os.WriteFile(filePath, []byte(sb.String()), 0644)
}
