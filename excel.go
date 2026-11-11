package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ReadExcelFile reads an Excel workbook (.xlsx) and parses domain records
func ReadExcelFile(filePath string) ([]Record, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("không thể mở file Excel: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("file Excel không có sheet nào")
	}

	sheetName := sheets[0]
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("lỗi đọc sheet '%s': %w", sheetName, err)
	}

	var records []Record
	headerMap := make(map[string]int)
	headerFound := false
	rowIdx := 0

	for _, row := range rows {
		cleanedRow := make([]string, len(row))
		for i, v := range row {
			cleanedRow[i] = strings.TrimSpace(v)
		}

		if !headerFound {
			websiteCol := -1
			for colIdx, colVal := range cleanedRow {
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
				for colIdx, colVal := range cleanedRow {
					cleanCol := strings.ToLower(colVal)
					if strings.Contains(cleanCol, "website") || strings.Contains(cleanCol, "domain") ||
						strings.Contains(cleanCol, "web") || strings.Contains(cleanCol, "tên miền") || strings.Contains(cleanCol, "url") {
						headerMap["website"] = colIdx
					} else if strings.Contains(cleanCol, "khách hàng") || strings.Contains(cleanCol, "khach hang") {
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

		rowIdx++
		rec := Record{
			Index: strconv.Itoa(rowIdx),
		}

		if headerFound {
			if idx, ok := headerMap["website"]; ok && idx < len(cleanedRow) {
				rec.RawWebsite = cleanedRow[idx]
			}
			if idx, ok := headerMap["customer"]; ok && idx < len(cleanedRow) {
				rec.Customer = cleanedRow[idx]
			}
			if idx, ok := headerMap["unit"]; ok && idx < len(cleanedRow) {
				rec.Unit = cleanedRow[idx]
			}
			if idx, ok := headerMap["type"]; ok && idx < len(cleanedRow) {
				rec.Type = cleanedRow[idx]
			}
			if idx, ok := headerMap["index"]; ok && idx < len(cleanedRow) && cleanedRow[idx] != "" {
				rec.Index = cleanedRow[idx]
			}
		} else {
			if len(cleanedRow) >= 5 {
				rec.Index = cleanedRow[0]
				rec.Unit = cleanedRow[1]
				rec.Type = cleanedRow[2]
				rec.Customer = cleanedRow[3]
				rec.RawWebsite = cleanedRow[4]
			} else if len(cleanedRow) == 1 {
				rec.RawWebsite = cleanedRow[0]
			} else if len(cleanedRow) > 0 {
				rec.RawWebsite = cleanedRow[len(cleanedRow)-1]
			}
		}

		target, tType, _ := ExtractTarget(rec.RawWebsite)
		rec.CleanTarget = target
		rec.TargetType = tType
		records = append(records, rec)
	}

	return records, nil
}

// ExportExcelReport creates a professional multi-sheet Excel report with specialized categories
func ExportExcelReport(filePath string, results []CheckResult, stats SummaryStats) error {
	f := excelize.NewFile()
	defer f.Close()

	sheetDNS := "Hết Hạn & Lỗi DNS"
	sheetUnreachable := "Không Thể Truy Cập"
	sheetNoSSL := "Không Có SSL & Cảnh Báo"
	sheetOK := "Hoạt Động Tốt"
	sheetAll := "Toàn Bộ Kết Quả"
	sheetStats := "Thống Kê Tổng Quan"

	// Create sheets
	f.SetSheetName("Sheet1", sheetDNS)
	f.NewSheet(sheetUnreachable)
	f.NewSheet(sheetNoSSL)
	f.NewSheet(sheetOK)
	f.NewSheet(sheetAll)
	f.NewSheet(sheetStats)

	// Styles
	headerRed, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#C53030"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})
	headerOrange, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#DD6B20"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})
	headerYellow, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#D69E2E"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})
	headerGreen, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#38A169"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})
	headerBlue, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#2B6CB0"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})

	styleDNS, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#9B2C2C"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#FED7D7"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	styleUnreachable, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#C05621"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#FEEBC8"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	styleNoSSL, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#975A16"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#FEFCBF"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	styleOK, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#22543D"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#C6F6D5"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	styleWarn, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#553C9A"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E9D8FD"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	centerStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	headers := []string{
		"STT", "Đơn Vị", "Loại", "Khách Hàng", "Website Gốc Trong File",
		"Target Đã Check", "Phân Loại", "Trạng Thái", "Mã HTTP", "Chi Tiết Sự Cố / Trạng Thái",
		"IP Máy Chủ", "Thời Gian (ms)",
	}

	populateSheet := func(sheet string, headerStyle int, filter func(r CheckResult) bool) {
		for c, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(c+1, 1)
			f.SetCellValue(sheet, cell, h)
			f.SetCellStyle(sheet, cell, cell, headerStyle)
		}

		row := 2
		for _, r := range results {
			if filter != nil && !filter(r) {
				continue
			}

			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), r.Index)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), r.Unit)
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), r.Type)
			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), r.Customer)
			f.SetCellValue(sheet, fmt.Sprintf("E%d", row), r.RawWebsite)
			f.SetCellValue(sheet, fmt.Sprintf("F%d", row), r.CleanTarget)
			f.SetCellValue(sheet, fmt.Sprintf("G%d", row), r.Category)
			f.SetCellValue(sheet, fmt.Sprintf("H%d", row), string(r.Status))

			badgeStyle := styleOK
			switch r.Status {
			case StatusDNSError:
				badgeStyle = styleDNS
			case StatusUnreachable:
				badgeStyle = styleUnreachable
			case StatusNoSSL:
				badgeStyle = styleNoSSL
			case StatusWarning:
				badgeStyle = styleWarn
			case StatusDead:
				badgeStyle = styleDNS
			}

			f.SetCellStyle(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("G%d", row), badgeStyle)
			f.SetCellStyle(sheet, fmt.Sprintf("H%d", row), fmt.Sprintf("H%d", row), badgeStyle)

			statusVal := ""
			if r.StatusCode > 0 {
				statusVal = strconv.Itoa(r.StatusCode)
			}
			f.SetCellValue(sheet, fmt.Sprintf("I%d", row), statusVal)
			f.SetCellValue(sheet, fmt.Sprintf("J%d", row), r.StatusText+" - "+r.Details)
			f.SetCellValue(sheet, fmt.Sprintf("K%d", row), r.IP)
			f.SetCellValue(sheet, fmt.Sprintf("L%d", row), r.ResponseMs)

			f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), centerStyle)
			f.SetCellStyle(sheet, fmt.Sprintf("I%d", row), fmt.Sprintf("I%d", row), centerStyle)
			f.SetCellStyle(sheet, fmt.Sprintf("L%d", row), fmt.Sprintf("L%d", row), centerStyle)
			row++
		}

		// Set column widths
		f.SetColWidth(sheet, "A", "A", 6)
		f.SetColWidth(sheet, "B", "B", 14)
		f.SetColWidth(sheet, "C", "C", 14)
		f.SetColWidth(sheet, "D", "D", 26)
		f.SetColWidth(sheet, "E", "E", 26)
		f.SetColWidth(sheet, "F", "F", 26)
		f.SetColWidth(sheet, "G", "G", 18)
		f.SetColWidth(sheet, "H", "H", 16)
		f.SetColWidth(sheet, "I", "I", 10)
		f.SetColWidth(sheet, "J", "J", 45)
		f.SetColWidth(sheet, "K", "K", 16)
		f.SetColWidth(sheet, "L", "L", 14)
	}

	// 1. Populate Sheet "Hết Hạn & Lỗi DNS"
	populateSheet(sheetDNS, headerRed, func(r CheckResult) bool {
		return r.Status == StatusDNSError
	})

	// 2. Populate Sheet "Không Thể Truy Cập"
	populateSheet(sheetUnreachable, headerOrange, func(r CheckResult) bool {
		return r.Status == StatusUnreachable
	})

	// 3. Populate Sheet "Không Có SSL & Cảnh Báo"
	populateSheet(sheetNoSSL, headerYellow, func(r CheckResult) bool {
		return r.Status == StatusNoSSL || r.Status == StatusWarning
	})

	// 4. Populate Sheet "Hoạt Động Tốt"
	populateSheet(sheetOK, headerGreen, func(r CheckResult) bool {
		return r.Status == StatusOK
	})

	// 5. Populate Sheet "Toàn Bộ Kết Quả"
	populateSheet(sheetAll, headerBlue, nil)

	// 6. Populate Sheet "Thống Kê Tổng Quan"
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14, Color: "#1A365D"},
	})
	f.SetCellValue(sheetStats, "A1", "BÁO CÁO PHÂN LOẠI SỨC KHỎE HỆ THỐNG WEBSITE / TÊN MIỀN")
	f.SetCellStyle(sheetStats, "A1", "A1", titleStyle)

	calcPct := func(count int) string {
		if stats.Total == 0 {
			return "0%"
		}
		return fmt.Sprintf("%.1f%%", float64(count)/float64(stats.Total)*100)
	}

	metrics := [][]interface{}{
		{"Nhóm Phân Loại", "Số Lượng", "Tỉ Lệ (%)", "Ý Nghĩa / Hướng Xử Lý"},
		{"Hoạt động tốt (OK)", stats.OKCount, calcPct(stats.OKCount), "Website hoạt động bình thường, HTTPS và SSL chuẩn"},
		{"Không có SSL / Lỗi SSL", stats.NoSSLCount, calcPct(stats.NoSSLCount), "Website vào được bình thường (chạy HTTP hoặc SSL lỗi/hết hạn)"},
		{"Không thể truy cập", stats.UnreachableCount, calcPct(stats.UnreachableCount), "Máy chủ sập, Timeout (>8s), Từ chối kết nối, Lỗi 5xx (Cần kiểm tra Server)"},
		{"Hết hạn domain / Lỗi DNS", stats.DNSErrorCount, calcPct(stats.DNSErrorCount), "Không phân giải được IP (Tên miền hết hạn, chưa ĐK, chưa trỏ DNS)"},
		{"Bị chặn / Lỗi khác", stats.OtherCount, calcPct(stats.OtherCount), "Bị chặn truy cập (403 WAF/Cloudflare), 404 Not Found, hoặc dòng trống"},
		{"TỔNG CỘNG KIỂM TRA", stats.Total, "100%", "Tổng số website trong danh sách nạp vào"},
		{"Thời gian kiểm tra hoàn tất", stats.Duration.Round(100).String(), "", ""},
		{"Thời điểm xuất báo cáo", stats.CheckedTime.Format("2006-01-02 15:04:05"), "", ""},
	}

	for r, rowData := range metrics {
		for c, val := range rowData {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+3)
			f.SetCellValue(sheetStats, cell, val)
			if r == 0 {
				f.SetCellStyle(sheetStats, cell, cell, headerBlue)
			}
		}
	}

	f.SetColWidth(sheetStats, "A", "A", 32)
	f.SetColWidth(sheetStats, "B", "B", 14)
	f.SetColWidth(sheetStats, "C", "C", 14)
	f.SetColWidth(sheetStats, "D", "D", 65)

	return f.SaveAs(filePath)
}

// ConvertCSVToExcel reads a CSV and writes an Excel workbook (.xlsx)
func ConvertCSVToExcel(csvPath, excelPath string) error {
	records, err := ReadCSVFile(csvPath)
	if err != nil {
		return err
	}
	stats := ComputeStats(nil, 0)
	stats.Total = len(records)
	var results []CheckResult
	for _, r := range records {
		results = append(results, CheckResult{
			Record:   r,
			Status:   StatusUnknown,
			Category: "Chưa kiểm tra",
		})
	}
	return ExportExcelReport(excelPath, results, stats)
}
