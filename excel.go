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

	// Use first sheet
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
		// Clean row items
		cleanedRow := make([]string, len(row))
		for i, v := range row {
			cleanedRow[i] = strings.TrimSpace(v)
		}

		// Look for header row
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

		// Row processing
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

// ExportExcelReport creates a professional multi-sheet Excel report
func ExportExcelReport(filePath string, results []CheckResult, stats SummaryStats) error {
	f := excelize.NewFile()
	defer f.Close()

	sheetDead := "Domain Đang Hỏng"
	sheetAll := "Toàn Bộ Kết Quả"
	sheetStats := "Thống Kê Tổng Quan"

	// Create sheets
	f.SetSheetName("Sheet1", sheetDead)
	f.NewSheet(sheetAll)
	f.NewSheet(sheetStats)

	// Define styles
	headerStyleRed, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E53E3E"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})

	headerStyleBlue, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#2B6CB0"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})

	styleDead, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#9B2C2C"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#FED7D7"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	styleWarn, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#975A16"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#FEFCBF"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	styleOK, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#22543D"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#C6F6D5"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	centerStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	headers := []string{
		"STT", "Đơn Vị", "Loại", "Khách Hàng", "Website Gốc Trong File",
		"Target Đã Check", "Trạng Thái", "Mã Phản Hồi", "Chi Tiết Sự Cố / Trạng Thái",
		"IP Máy Chủ", "Thời Gian (ms)",
	}

	// 1. Populate Sheet "Domain Đang Hỏng"
	for c, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(c+1, 1)
		f.SetCellValue(sheetDead, cell, h)
		f.SetCellStyle(sheetDead, cell, cell, headerStyleRed)
	}

	deadRow := 2
	for _, r := range results {
		if r.Status == StatusDead {
			f.SetCellValue(sheetDead, fmt.Sprintf("A%d", deadRow), r.Index)
			f.SetCellValue(sheetDead, fmt.Sprintf("B%d", deadRow), r.Unit)
			f.SetCellValue(sheetDead, fmt.Sprintf("C%d", deadRow), r.Type)
			f.SetCellValue(sheetDead, fmt.Sprintf("D%d", deadRow), r.Customer)
			f.SetCellValue(sheetDead, fmt.Sprintf("E%d", deadRow), r.RawWebsite)
			f.SetCellValue(sheetDead, fmt.Sprintf("F%d", deadRow), r.CleanTarget)
			f.SetCellValue(sheetDead, fmt.Sprintf("G%d", deadRow), string(r.Status))
			f.SetCellStyle(sheetDead, fmt.Sprintf("G%d", deadRow), fmt.Sprintf("G%d", deadRow), styleDead)

			statusVal := ""
			if r.StatusCode > 0 {
				statusVal = strconv.Itoa(r.StatusCode)
			}
			f.SetCellValue(sheetDead, fmt.Sprintf("H%d", deadRow), statusVal)
			f.SetCellValue(sheetDead, fmt.Sprintf("I%d", deadRow), r.StatusText+" - "+r.Details)
			f.SetCellValue(sheetDead, fmt.Sprintf("J%d", deadRow), r.IP)
			f.SetCellValue(sheetDead, fmt.Sprintf("K%d", deadRow), r.ResponseMs)

			f.SetCellStyle(sheetDead, fmt.Sprintf("A%d", deadRow), fmt.Sprintf("A%d", deadRow), centerStyle)
			f.SetCellStyle(sheetDead, fmt.Sprintf("H%d", deadRow), fmt.Sprintf("H%d", deadRow), centerStyle)
			f.SetCellStyle(sheetDead, fmt.Sprintf("K%d", deadRow), fmt.Sprintf("K%d", deadRow), centerStyle)
			deadRow++
		}
	}

	// 2. Populate Sheet "Toàn Bộ Kết Quả"
	for c, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(c+1, 1)
		f.SetCellValue(sheetAll, cell, h)
		f.SetCellStyle(sheetAll, cell, cell, headerStyleBlue)
	}

	for i, r := range results {
		row := i + 2
		f.SetCellValue(sheetAll, fmt.Sprintf("A%d", row), r.Index)
		f.SetCellValue(sheetAll, fmt.Sprintf("B%d", row), r.Unit)
		f.SetCellValue(sheetAll, fmt.Sprintf("C%d", row), r.Type)
		f.SetCellValue(sheetAll, fmt.Sprintf("D%d", row), r.Customer)
		f.SetCellValue(sheetAll, fmt.Sprintf("E%d", row), r.RawWebsite)
		f.SetCellValue(sheetAll, fmt.Sprintf("F%d", row), r.CleanTarget)
		f.SetCellValue(sheetAll, fmt.Sprintf("G%d", row), string(r.Status))

		switch r.Status {
		case StatusDead:
			f.SetCellStyle(sheetAll, fmt.Sprintf("G%d", row), fmt.Sprintf("G%d", row), styleDead)
		case StatusWarning:
			f.SetCellStyle(sheetAll, fmt.Sprintf("G%d", row), fmt.Sprintf("G%d", row), styleWarn)
		default:
			f.SetCellStyle(sheetAll, fmt.Sprintf("G%d", row), fmt.Sprintf("G%d", row), styleOK)
		}

		statusVal := ""
		if r.StatusCode > 0 {
			statusVal = strconv.Itoa(r.StatusCode)
		}
		f.SetCellValue(sheetAll, fmt.Sprintf("H%d", row), statusVal)
		f.SetCellValue(sheetAll, fmt.Sprintf("I%d", row), r.StatusText+" - "+r.Details)
		f.SetCellValue(sheetAll, fmt.Sprintf("J%d", row), r.IP)
		f.SetCellValue(sheetAll, fmt.Sprintf("K%d", row), r.ResponseMs)

		f.SetCellStyle(sheetAll, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), centerStyle)
		f.SetCellStyle(sheetAll, fmt.Sprintf("H%d", row), fmt.Sprintf("H%d", row), centerStyle)
		f.SetCellStyle(sheetAll, fmt.Sprintf("K%d", row), fmt.Sprintf("K%d", row), centerStyle)
	}

	// 3. Populate Sheet "Thống Kê Tổng Quan"
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14, Color: "#1A365D"},
	})
	f.SetCellValue(sheetStats, "A1", "BÁO CÁO SỨC KHỎE HỆ THỐNG WEBSITE / TÊN MIỀN")
	f.SetCellStyle(sheetStats, "A1", "A1", titleStyle)

	metrics := [][]interface{}{
		{"Chỉ số", "Giá trị", "Ghi chú"},
		{"Tổng số website kiểm tra", stats.Total, "Tổng số bản ghi trong file"},
		{"Hoạt động tốt (OK)", stats.OKCount, "Website truy cập bình thường"},
		{"Bị hỏng / Lỗi (DEAD)", stats.DeadCount, "CẦN KHẮC PHỤC NGAY (DNS lỗi, timeout, 5xx, 404)"},
		{"Cảnh báo (WARNING)", stats.WarnCount, "Lỗi SSL hết hạn hoặc bị chặn 403"},
		{"Tỉ lệ sống (%)", fmt.Sprintf("%.1f%%", float64(stats.OKCount)/float64(stats.Total)*100), ""},
		{"Thời gian kiểm tra hoàn tất", stats.Duration.Round(100).String(), ""},
		{"Thời gian thực hiện", stats.CheckedTime.Format("2006-01-02 15:04:05"), ""},
	}

	for r, rowData := range metrics {
		for c, val := range rowData {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+3)
			f.SetCellValue(sheetStats, cell, val)
			if r == 0 {
				f.SetCellStyle(sheetStats, cell, cell, headerStyleBlue)
			}
		}
	}

	// Auto-fit column widths
	for _, sheet := range []string{sheetDead, sheetAll, sheetStats} {
		_ = f.SetColWidth(sheet, "A", "A", 8)
		_ = f.SetColWidth(sheet, "B", "C", 16)
		_ = f.SetColWidth(sheet, "D", "D", 32)
		_ = f.SetColWidth(sheet, "E", "F", 30)
		_ = f.SetColWidth(sheet, "G", "G", 14)
		_ = f.SetColWidth(sheet, "H", "H", 12)
		_ = f.SetColWidth(sheet, "I", "I", 50)
		_ = f.SetColWidth(sheet, "J", "J", 18)
		_ = f.SetColWidth(sheet, "K", "K", 14)
	}

	return f.SaveAs(filePath)
}

// ConvertCSVToExcel creates an Excel template file from CSV
func ConvertCSVToExcel(csvPath string, excelPath string) error {
	records, err := ReadCSVFile(csvPath)
	if err != nil {
		return err
	}

	f := excelize.NewFile()
	defer f.Close()

	sheet := "Danh Sách Domain"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"STT", "Đơn vị", "Loại", "Khách hàng", "Website"}
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#3182CE"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	for c, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(c+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	for i, r := range records {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), r.Index)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), r.Unit)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), r.Type)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), r.Customer)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), r.RawWebsite)
	}

	_ = f.SetColWidth(sheet, "A", "A", 8)
	_ = f.SetColWidth(sheet, "B", "C", 16)
	_ = f.SetColWidth(sheet, "D", "D", 35)
	_ = f.SetColWidth(sheet, "E", "E", 40)

	return f.SaveAs(excelPath)
}
