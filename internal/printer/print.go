package printer

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/phin1x/go-ipp"
	"go.uber.org/zap"
)

// PrintFile 打印本地PDF文件
func PrintFile(filePath string, printerName string, options PrintOptions, log *zap.Logger) error {
	log.Info("开始打印流程", zap.String("file", filePath), zap.String("printer", printerName))

	// 读取文件
	filePayload, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	log.Info("文件读取成功", zap.Int("size", len(filePayload)))

	if options.UseCUPS {
		return printPDFViaCUPS(filePath, printerName, options, log)
	}
	return printPDF(filePath, filePayload, printerName, options, log)
}

// PrintFromURL 从远程URL下载并打印PDF
func PrintFromURL(url string, printerName string, options PrintOptions, log *zap.Logger) error {
	log.Info("开始远程打印流程", zap.String("url", url), zap.String("printer", printerName))

	// 创建临时目录
	tmpDir := filepath.Join(os.TempDir(), "printer_downloads")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer func() {
		// 清理临时目录
		if err := os.RemoveAll(tmpDir); err != nil {
			log.Warn("清理临时目录失败", zap.Error(err))
		} else {
			log.Info("临时目录已清理", zap.String("dir", tmpDir))
		}
	}()

	// 下载文件
	log.Info("开始下载文件", zap.String("url", url))
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("下载文件失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，HTTP状态码: %d", resp.StatusCode)
	}

	// 生成临时文件名
	timestamp := time.Now().Format("20060102_150405")
	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("download_%s.pdf", timestamp))

	// 保存到临时文件
	file, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer file.Close()

	written, err := io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("保存文件失败: %w", err)
	}

	log.Info("文件下载成功", zap.String("file", tmpFile), zap.Int64("size", written))

	// 读取文件内容
	filePayload, err := os.ReadFile(tmpFile)
	if err != nil {
		return fmt.Errorf("读取临时文件失败: %w", err)
	}

	// 打印
	if options.UseCUPS {
		return printPDFViaCUPS(tmpFile, printerName, options, log)
	}
	return printPDF(tmpFile, filePayload, printerName, options, log)
}

// printPDF 使用IPP协议打印PDF
func printPDF(fileName string, filePayload []byte, printerName string, options PrintOptions, log *zap.Logger) error {
	log.Info("使用IPP协议打印", zap.String("file", fileName), zap.String("printer", printerName))

	// 初始化IPP客户端
	client := ipp.NewIPPClient("localhost", 631, "arch-user", "", false)
	log.Info("IPP客户端创建成功")

	// 创建Document结构体
	doc := ipp.Document{
		Document: bytes.NewReader(filePayload),
		Size:     len(filePayload),
		Name:     fileName,
		MimeType: "application/pdf",
	}

	// 构建打印属性
	jobAttributes := map[string]interface{}{
		ipp.AttributeJobName: fileName,
		ipp.AttributeCopies:  options.Copies,
		ipp.AttributeSides:   options.Sides,
	}

	log.Info("发送打印任务", zap.Any("options", options))

	// 发送打印任务
	jobID, err := client.PrintJob(doc, printerName, jobAttributes)
	if err != nil {
		log.Error("打印任务提交失败", zap.Error(err))
		return fmt.Errorf("打印失败: %w", err)
	}

	log.Info("打印任务提交成功", zap.Int("job_id", jobID))

	// 查询任务状态
	jobAttrs, err := client.GetJobAttributes(jobID, nil)
	if err != nil {
		log.Warn("无法获取任务属性", zap.Error(err))
	} else {
		if jobState, ok := jobAttrs[ipp.AttributeJobState]; ok {
			log.Info("任务状态", zap.Any("state", jobState[0].Value))
		}
	}

	return nil
}

// printPDFViaCUPS 使用CUPS lp命令打印
func printPDFViaCUPS(fileName string, printerName string, options PrintOptions, log *zap.Logger) error {
	log.Info("使用CUPS lp命令打印", zap.String("file", fileName), zap.String("printer", printerName))

	// 构建lp命令
	args := []string{"-d", printerName}

	// 设置份数
	if options.Copies > 1 {
		args = append(args, "-n", fmt.Sprintf("%d", options.Copies))
	}

	// 设置单双面
	if options.Sides != "one-sided" {
		args = append(args, "-o", fmt.Sprintf("sides=%s", options.Sides))
	}

	// 设置颜色模式
	if options.ColorMode != "auto" {
		args = append(args, "-o", fmt.Sprintf("print-color-mode=%s", options.ColorMode))
	}

	// 设置纸张来源
	if options.MediaSource != "auto" {
		args = append(args, "-o", fmt.Sprintf("media-source=%s", options.MediaSource))
	}

	// 添加文件名
	args = append(args, fileName)

	log.Info("执行lp命令", zap.Strings("args", args))

	cmd := exec.Command("lp", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Error("打印失败", zap.Error(err), zap.String("output", string(output)))
		return fmt.Errorf("打印失败: %w", err)
	}

	outputStr := strings.TrimSpace(string(output))
	log.Info("打印成功", zap.String("output", outputStr))

	return nil
}

// ValidatePrintOptions 验证打印选项
func ValidatePrintOptions(options PrintOptions, log *zap.Logger) error {
	if options.Copies < 1 || options.Copies > 999 {
		return fmt.Errorf("打印份数必须在 1-999 之间，当前值: %d", options.Copies)
	}

	validSides := map[string]bool{
		"one-sided":              true,
		"two-sided-long-edge":    true,
		"two-sided-short-edge":   true,
	}
	if !validSides[options.Sides] {
		return fmt.Errorf("无效的单双面设置: %s", options.Sides)
	}

	validColorModes := map[string]bool{
		"auto":       true,
		"color":      true,
		"monochrome": true,
	}
	if !validColorModes[options.ColorMode] {
		return fmt.Errorf("无效的颜色模式: %s", options.ColorMode)
	}

	validMediaSources := map[string]bool{
		"auto": true, "manual": true, "adf": true,
		"tray-1": true, "tray-2": true, "tray-3": true, "tray-4": true,
		"top": true, "bottom": true, "middle": true,
	}
	if !validMediaSources[options.MediaSource] {
		return fmt.Errorf("无效的纸张来源: %s", options.MediaSource)
	}

	return nil
}

// AutoSelectPrinter 自动选择第一台可用打印机
func AutoSelectPrinter(log *zap.Logger) (string, error) {
	printers, err := getCUPSPrinters(log)
	if err != nil || len(printers) == 0 {
		return "", fmt.Errorf("无法获取打印机列表，请先运行 --setup 配置打印机")
	}
	log.Info("自动选择打印机", zap.String("printer", printers[0]))
	return printers[0], nil
}
