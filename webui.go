package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

//go:embed web/index.html
var indexHTML []byte

// StartWebUI starts the embedded HTTP server and opens browser
func StartWebUI(port int, autoOpen bool) error {
	mux := http.NewServeMux()

	// 1. Root HTML
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexHTML)
	})

	// 2. Load Sample
	mux.HandleFunc("/api/load-sample", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		sampleFile := "mau_danh_sach.csv"
		if _, err := os.Stat(sampleFile); os.IsNotExist(err) {
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Không tìm thấy file mẫu mau_danh_sach.csv"})
			return
		}

		records, err := ReadCSVFile(sampleFile)
		if err != nil {
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		var lines []string
		for _, rec := range records {
			lines = append(lines, fmt.Sprintf("%s\t%s\t%s\t%s\t%s", rec.Index, rec.Unit, rec.Type, rec.Customer, rec.RawWebsite))
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"content": strings.Join(lines, "\n"),
			"count":   len(records),
		})
	})

	// 3. Paste Check (SSE Stream)
	mux.HandleFunc("/api/paste-check", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var reqData struct {
			Text        string `json:"text"`
			Timeout     int    `json:"timeout"`
			Concurrency int    `json:"concurrency"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		records, err := ParsePastedText(reqData.Text)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		streamHealthCheck(w, records, reqData.Timeout, reqData.Concurrency)
	})

	// 4. Upload Check (SSE Stream)
	mux.HandleFunc("/api/upload-check", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		err := r.ParseMultipartForm(32 << 20) // 32MB max
		if err != nil {
			http.Error(w, "Lỗi đọc form: "+err.Error(), http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Vui lòng chọn file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		timeout, _ := strconv.Atoi(r.FormValue("timeout"))
		concurrency, _ := strconv.Atoi(r.FormValue("concurrency"))

		// Save temp file to parse
		ext := strings.ToLower(filepath.Ext(header.Filename))
		tempFile, err := os.CreateTemp("", "upload-*"+ext)
		if err != nil {
			http.Error(w, "Không tạo được file tạm", http.StatusInternalServerError)
			return
		}
		defer func() {
			tempFile.Close()
			_ = os.Remove(tempFile.Name())
		}()

		if _, err := io.Copy(tempFile, file); err != nil {
			http.Error(w, "Lỗi ghi file tạm", http.StatusInternalServerError)
			return
		}

		var records []Record
		if ext == ".xlsx" || ext == ".xlsm" {
			records, err = ReadExcelFile(tempFile.Name())
		} else {
			records, err = ReadCSVFile(tempFile.Name())
		}

		if err != nil {
			http.Error(w, "Lỗi đọc file: "+err.Error(), http.StatusBadRequest)
			return
		}

		streamHealthCheck(w, records, timeout, concurrency)
	})

	// 5. Export Excel Download
	mux.HandleFunc("/api/export", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var results []CheckResult
		if err := json.NewDecoder(r.Body).Decode(&results); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		stats := ComputeStats(results, 0)
		timeTag := time.Now().Format("20060102_150405")
		downloadName := fmt.Sprintf("baocao_%s.xlsx", timeTag)

		tempFile, err := os.CreateTemp("", "export-*.xlsx")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tempPath := tempFile.Name()
		tempFile.Close()
		defer os.Remove(tempPath)

		if err := ExportExcelReport(tempPath, results, stats); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", downloadName))
		http.ServeFile(w, r, tempPath)
	})

	// 6. Open Reports Folder in File Explorer
	mux.HandleFunc("/api/open-folder", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		absPath, _ := filepath.Abs(ReportsDir)
		err := OpenFolder(ReportsDir)
		if err != nil {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "path": absPath})
	})

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	localURL := fmt.Sprintf("http://localhost:%d", port)

	fmt.Printf("\n[UI] Giao diện Web đang chạy tại: %s\n", localURL)
	if autoOpen {
		go func() {
			time.Sleep(500 * time.Millisecond)
			OpenBrowser(localURL)
		}()
	}

	return http.ListenAndServe(addr, mux)
}

func streamHealthCheck(w http.ResponseWriter, records []Record, timeoutSec int, concurrency int) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	timeout := time.Duration(timeoutSec) * time.Second
	if timeoutSec <= 0 {
		timeout = DefaultTimeout
	}
	if concurrency <= 0 {
		concurrency = DefaultConcurrency
	}

	engine := NewCheckEngine(timeout, concurrency)
	startTime := time.Now()

	progress := func(completed int, total int, res CheckResult) {
		msg := map[string]interface{}{
			"type":      "item",
			"completed": completed,
			"total":     total,
			"data":      res,
		}
		bytesData, _ := json.Marshal(msg)
		fmt.Fprintf(w, "data: %s\n\n", bytesData)
		flusher.Flush()
	}

	results := engine.Run(records, progress)
	stats := ComputeStats(results, time.Since(startTime))

	// Auto save reports into reports folder
	_ = os.MkdirAll(ReportsDir, 0755)
	timeTag := time.Now().Format("20060102_150405")
	excelFile := fmt.Sprintf("baocao_%s.xlsx", timeTag)
	txtFile := fmt.Sprintf("hong_%s.txt", timeTag)
	excelPath := filepath.Join(ReportsDir, excelFile)
	txtPath := filepath.Join(ReportsDir, txtFile)

	_ = ExportExcelReport(excelPath, results, stats)
	_ = ExportTextSummary(txtPath, results)

	absDir, _ := filepath.Abs(ReportsDir)

	// Send done event
	doneMsg := map[string]interface{}{
		"type":        "done",
		"stats":       stats,
		"excel_file":  excelFile,
		"txt_file":    txtFile,
		"reports_dir": absDir,
	}
	bytesDone, _ := json.Marshal(doneMsg)
	fmt.Fprintf(w, "data: %s\n\n", bytesDone)
	flusher.Flush()
}

// OpenFolder opens a folder in the OS file manager (Windows File Explorer, macOS Finder)
func OpenFolder(dirPath string) error {
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		absPath = dirPath
	}
	_ = os.MkdirAll(absPath, 0755)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer.exe", absPath)
	case "darwin":
		cmd = exec.Command("open", absPath)
	default:
		cmd = exec.Command("xdg-open", absPath)
	}
	return cmd.Start()
}

// OpenBrowser opens the URL in the system default browser
func OpenBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
