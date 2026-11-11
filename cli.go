package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RunInteractiveCLI handles the terminal menu loop
func RunInteractiveCLI(timeout time.Duration, concurrency int, scanner *bufio.Scanner) {
	if scanner == nil {
		scanner = bufio.NewScanner(os.Stdin)
	}

	for {
		printBanner()
		fmt.Println("  [1] Kiểm tra từ file Excel (.xlsx) hoặc CSV (.csv)")
		fmt.Println("  [2] Dán danh sách domain trực tiếp (Copy từ cột Excel)")
		fmt.Println("  [3] Chạy kiểm tra nhanh với file mẫu 'mau_danh_sach.xlsx'")
		fmt.Println("  [4] Mở thư mục chứa các file báo cáo đã lưu ('reports')")
		fmt.Println("  [5] Mở giao diện Web UI trực quan trên trình duyệt (Dashboard)")
		fmt.Println("  [6] Thoát chương trình")
		fmt.Println("========================================================================")
		fmt.Print("-> Chọn chức năng (1-6): ")

		if !scanner.Scan() {
			break
		}
		choice := strings.TrimSpace(scanner.Text())

		switch choice {
		case "1":
			handleFileCheck(scanner, timeout, concurrency)
		case "2":
			handlePasteCheck(scanner, timeout, concurrency)
		case "3":
			handleSampleCheck(timeout, concurrency)
		case "4":
			_ = OpenFolder(ReportsDir)
			absDir, _ := filepath.Abs(ReportsDir)
			fmt.Printf("\n[+] Đã mở thư mục báo cáo: %s\n", absDir)
			pause(scanner)
		case "5":
			fmt.Println("\n[*] Đang khởi động Web UI trên cổng 8080...")
			err := StartWebUI(8080, true)
			if err != nil {
				fmt.Printf("[!] Lỗi Web UI: %v\n", err)
			}
		case "6", "q", "exit":
			fmt.Println("\nTạm biệt!")
			return
		default:
			fmt.Println("\n[!] Lựa chọn không hợp lệ, vui lòng chọn từ 1 đến 6.")
			pause(scanner)
		}
	}
}

func printBanner() {
	fmt.Println()
	fmt.Println("========================================================================")
	fmt.Println("       TOOL KIỂM TRA SỨC KHỎE DOMAIN & WEBSITE (HEALTH CHECK TOOL)       ")
	fmt.Println("                 Tự động phát hiện & xuất domain hỏng                   ")
	fmt.Println("========================================================================")
}

func handleFileCheck(scanner *bufio.Scanner, timeout time.Duration, concurrency int) {
	fmt.Print("\n[+] Kéo thả file Excel (.xlsx) hoặc CSV vào đây (hoặc nhập đường dẫn): ")
	if !scanner.Scan() {
		return
	}
	rawPath := strings.TrimSpace(scanner.Text())
	filePath := strings.Trim(rawPath, "\"") // remove surrounding quotes from drag-and-drop

	if filePath == "" {
		fmt.Println("[!] Bạn chưa nhập đường dẫn file.")
		pause(scanner)
		return
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("[!] Không tìm thấy file: %s\n", filePath)
		pause(scanner)
		return
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	var records []Record
	var err error

	fmt.Printf("[*] Đang đọc file '%s'...\n", filepath.Base(filePath))
	if ext == ".xlsx" || ext == ".xlsm" {
		records, err = ReadExcelFile(filePath)
	} else {
		records, err = ReadCSVFile(filePath)
	}

	if err != nil {
		fmt.Printf("[!] Lỗi khi đọc file: %v\n", err)
		pause(scanner)
		return
	}

	fmt.Printf("[+] Đã nạp thành công %d bản ghi từ file.\n", len(records))
	executeHealthCheck(records, timeout, concurrency, scanner)
}

func handlePasteCheck(scanner *bufio.Scanner, timeout time.Duration, concurrency int) {
	fmt.Println("\n[+] Dán danh sách domain vào đây (copy từ Excel).")
	fmt.Println("    Sau khi dán xong, gõ chữ 'CHECK' hoặc nhấn phím Enter 2 lần để bắt đầu:")
	fmt.Println("------------------------------------------------------------------------")

	var lines []string
	emptyLineCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.EqualFold(trimmed, "CHECK") || strings.EqualFold(trimmed, "END") {
			break
		}

		if trimmed == "" {
			emptyLineCount++
			if emptyLineCount >= 2 && len(lines) > 0 {
				break
			}
			continue
		} else {
			emptyLineCount = 0
		}

		lines = append(lines, line)
	}

	if len(lines) == 0 {
		fmt.Println("[!] Chưa có nội dung nào được dán.")
		pause(scanner)
		return
	}

	records, err := ParsePastedText(strings.Join(lines, "\n"))
	if err != nil {
		fmt.Printf("[!] Lỗi phân tích dữ liệu dán: %v\n", err)
		pause(scanner)
		return
	}

	fmt.Printf("\n[+] Đã nhận diện %d tên miền/website.\n", len(records))
	executeHealthCheck(records, timeout, concurrency, scanner)
}

func handleSampleCheck(timeout time.Duration, concurrency int) {
	sampleFile := "mau_danh_sach.csv"
	if _, err := os.Stat(sampleFile); os.IsNotExist(err) {
		fmt.Printf("[!] Không tìm thấy file mẫu '%s' trong thư mục hiện tại.\n", sampleFile)
		return
	}

	records, err := ReadCSVFile(sampleFile)
	if err != nil {
		fmt.Printf("[!] Lỗi đọc file mẫu: %v\n", err)
		return
	}

	fmt.Printf("[*] Đang chạy kiểm tra với %d website trong file mẫu...\n", len(records))
	executeHealthCheck(records, timeout, concurrency, nil)
}

func executeHealthCheck(records []Record, timeout time.Duration, concurrency int, scanner *bufio.Scanner) {
	fmt.Printf("[*] Bắt đầu kiểm tra với %d luồng song song (Timeout: %v)...\n", concurrency, timeout)
	engine := NewCheckEngine(timeout, concurrency)
	start := time.Now()

	lastPrint := time.Now()
	progress := func(completed int, total int, res CheckResult) {
		if time.Since(lastPrint) > 100*time.Millisecond || completed == total {
			lastPrint = time.Now()
			pct := completed * 100 / total
			barLen := 25
			filled := completed * barLen / total
			bar := strings.Repeat("=", filled) + strings.Repeat(" ", barLen-filled)
			target := res.CleanTarget
			if len(target) > 30 {
				target = target[:27] + "..."
			}
			fmt.Printf("\r[%s] %d%% (%d/%d) - Check: %-30s", bar, pct, completed, total, target)
		}
	}

	results := engine.Run(records, progress)
	duration := time.Since(start)
	stats := ComputeStats(results, duration)

	fmt.Printf("\r%s\r", strings.Repeat(" ", 80)) // Clear progress line

	// Display Summary
	fmt.Println("\n========================================================================")
	fmt.Println("                            KẾT QUẢ KIỂM TRA                            ")
	fmt.Println("========================================================================")
	fmt.Printf(" - Tổng số website kiểm tra : %d\n", stats.Total)
	fmt.Printf(" - Hoạt động bình thường(OK): %d\n", stats.OKCount)
	fmt.Printf(" - Cảnh báo (SSL / Chặn 403): %d\n", stats.WarnCount)
	fmt.Printf(" - BỊ HỎNG / SẬP (DEAD)     : %d  (CẦN XỬ LÝ!)\n", stats.DeadCount)
	fmt.Printf(" - Thời gian thực hiện      : %s\n", duration.Round(time.Millisecond))
	fmt.Println("========================================================================")

	// List Dead Domains
	if stats.DeadCount > 0 {
		fmt.Println("\nDANH SÁCH TÊN MIỀN ĐANG HỎNG / SẬP:")
		fmt.Println("------------------------------------------------------------------------")
		count := 0
		for _, r := range results {
			if r.Status == StatusDead {
				count++
				customer := r.Customer
				if customer == "" {
					customer = "N/A"
				}
				target := r.CleanTarget
				if target == "" {
					target = r.RawWebsite
				}
				fmt.Printf(" %2d. [HỎNG] %-32s | KH: %-25s | Lỗi: %s\n", count, target, truncate(customer, 25), r.StatusText)
			}
		}
		fmt.Println("------------------------------------------------------------------------")
	} else {
		fmt.Println("\n[OK] TUYỆT VỜI! Tất cả các tên miền đều đang hoạt động bình thường.")
	}

	// Export Files into reports folder with clean, short timestamped names
	_ = os.MkdirAll(ReportsDir, 0755)
	timeTag := time.Now().Format("20060102_150405")
	excelFile := fmt.Sprintf("baocao_%s.xlsx", timeTag)
	txtFile := fmt.Sprintf("hong_%s.txt", timeTag)
	excelOut := filepath.Join(ReportsDir, excelFile)
	txtOut := filepath.Join(ReportsDir, txtFile)

	err := ExportExcelReport(excelOut, results, stats)
	if err == nil {
		fmt.Printf("\n[+] Đã lưu báo cáo Excel : %s\n", excelOut)
	} else {
		fmt.Printf("\n[!] Không thể xuất Excel: %v\n", err)
	}

	err = ExportTextSummary(txtOut, results)
	if err == nil {
		fmt.Printf("[+] Đã lưu danh sách hỏng: %s\n", txtOut)
	}

	absDir, _ := filepath.Abs(ReportsDir)
	fmt.Println("------------------------------------------------------------------------")
	fmt.Printf("[i] Tất cả báo cáo được lưu tại thư mục: %s\n", absDir)

	if scanner != nil {
		fmt.Print("\n[?] Bạn có muốn MỞ THƯ MỤC BÁO CÁO trên máy tính không? (Y/n): ")
		if scanner.Scan() {
			ans := strings.ToLower(strings.TrimSpace(scanner.Text()))
			if ans == "" || ans == "y" || ans == "yes" {
				_ = OpenFolder(ReportsDir)
				fmt.Println("[+] Đã mở thư mục báo cáo trong File Explorer!")
			}
		}

		if stats.DeadCount > 0 {
			fmt.Print("\n[?] Bạn có muốn kiểm tra lại (re-check) chỉ các domain đang hỏng không? (y/N): ")
			if scanner.Scan() {
				ans := strings.ToLower(strings.TrimSpace(scanner.Text()))
				if ans == "y" || ans == "yes" {
					var deadRecords []Record
					for _, r := range results {
						if r.Status == StatusDead {
							deadRecords = append(deadRecords, r.Record)
						}
					}
					fmt.Printf("\n[*] Đang kiểm tra lại %d domain bị hỏng...\n", len(deadRecords))
					executeHealthCheck(deadRecords, timeout, concurrency, scanner)
					return
				}
			}
		}
		pause(scanner)
	}
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen-3]) + "..."
	}
	return s
}

func pause(scanner *bufio.Scanner) {
	fmt.Print("\n-> Nhấn phím Enter để tiếp tục...")
	scanner.Scan()
}
