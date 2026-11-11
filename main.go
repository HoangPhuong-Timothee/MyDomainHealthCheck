package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	filePath := flag.String("file", "", "Đường dẫn file Excel (.xlsx) hoặc CSV để kiểm tra trực tiếp")
	uiMode := flag.Bool("ui", false, "Chạy chế độ Web UI trực quan trên trình duyệt")
	port := flag.Int("port", 8080, "Cổng máy chủ Web UI (mặc định 8080)")
	timeoutSec := flag.Int("timeout", 18, "Thời gian chờ tối đa cho mỗi domain (giây)")
	concurrency := flag.Int("concurrency", 30, "Số luồng kiểm tra song song (mặc định 30)")
	convertSample := flag.Bool("convert-sample", false, "Chuyển file mau_danh_sach.csv thành mau_danh_sach.xlsx")

	flag.Parse()

	timeout := time.Duration(*timeoutSec) * time.Second

	// Optional helper to create sample excel file
	if *convertSample {
		err := ConvertCSVToExcel("mau_danh_sach.csv", "mau_danh_sach.xlsx")
		if err != nil {
			fmt.Printf("Lỗi chuyển đổi: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Đã tạo file mau_danh_sach.xlsx thành công!")
		return
	}

	// Mode 1: Web UI flag
	if *uiMode {
		fmt.Printf("[*] Đang khởi động Domain Health Check Web UI trên cổng %d...\n", *port)
		if err := StartWebUI(*port, true); err != nil {
			fmt.Printf("[!] Lỗi khi chạy Web UI: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Mode 2: Direct file check via CLI flag
	if *filePath != "" {
		ext := strings.ToLower(filepath.Ext(*filePath))
		var records []Record
		var err error

		if ext == ".xlsx" || ext == ".xlsm" {
			records, err = ReadExcelFile(*filePath)
		} else {
			records, err = ReadCSVFile(*filePath)
		}

		if err != nil {
			fmt.Printf("[!] Lỗi đọc file '%s': %v\n", *filePath, err)
			os.Exit(1)
		}

		fmt.Printf("[+] Đã nạp %d website từ file '%s'.\n", len(records), filepath.Base(*filePath))
		executeHealthCheck(records, timeout, *concurrency, nil)
		return
	}

	// Mode 3: Interactive Startup Prompt (When double clicked or run without arguments)
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println()
	fmt.Println("========================================================================")
	fmt.Println("       TOOL KIỂM TRA SỨC KHỎE DOMAIN & WEBSITE (HEALTH CHECK TOOL)       ")
	fmt.Println("========================================================================")
	fmt.Println("  Bạn muốn sử dụng giao diện nào?")
	fmt.Println()
	fmt.Println("  [1] Giao diện Đồ Họa Web UI (Khuyên dùng - Mở trực tiếp trên trình duyệt)")
	fmt.Println("  [2] Giao diện Dòng Lệnh CLI (Chạy trực tiếp tại cửa sổ này)")
	fmt.Println("  [3] Mở thư mục chứa các file báo cáo đã lưu ('reports')")
	fmt.Println("  [4] Thoát")
	fmt.Println("========================================================================")
	fmt.Print("-> Nhập lựa chọn [Mặc định: 1 - Bấm Enter để mở Web UI]: ")

	choice := "1"
	if scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text != "" {
			choice = text
		}
	}

	switch choice {
	case "1":
		fmt.Printf("\n[*] Đang khởi chạy Giao diện Web UI trên cổng %d và mở trình duyệt...\n", *port)
		if err := StartWebUI(*port, true); err != nil {
			fmt.Printf("[!] Lỗi khi chạy Web UI: %v\n", err)
			pause(scanner)
		}
	case "2":
		RunInteractiveCLI(timeout, *concurrency, scanner)
	case "3":
		_ = OpenFolder(ReportsDir)
		absDir, _ := filepath.Abs(ReportsDir)
		fmt.Printf("\n[+] Đã mở thư mục báo cáo: %s\n", absDir)
		pause(scanner)
	case "4", "q", "exit":
		fmt.Println("\nTạm biệt!")
		return
	default:
		fmt.Printf("\n[*] Đang khởi chạy Giao diện Web UI trên cổng %d...\n", *port)
		_ = StartWebUI(*port, true)
	}
}
