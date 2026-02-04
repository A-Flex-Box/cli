package printer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ScanDocument 扫描文档并保存为PDF
func ScanDocument(deviceName string, options ScanOptions, log *zap.Logger) error {
	log.Info("开始扫描流程", zap.String("device", deviceName), zap.Any("options", options))

	// 1. 检查扫描设备
	log.Info("检查扫描设备", zap.String("device", deviceName))
	cmd := exec.Command("scanimage", "-L")
	output, err := cmd.Output()
	if err != nil {
		log.Error("scanimage命令不可用", zap.Error(err))
		return fmt.Errorf("scanimage不可用: %w", err)
	}

	log.Info("scanimage可用")
	allDevices := parseScanDevices(string(output))
	if len(allDevices) == 0 {
		log.Warn("未找到可用扫描设备")
		return fmt.Errorf("未找到可用扫描设备")
	}

	// 过滤和分类设备：优先打印机扫描设备，排除摄像头
	var printerDevices []ScanDeviceInfo
	var otherDevices []ScanDeviceInfo

	for _, dev := range allDevices {
		if strings.HasPrefix(dev.Name, "v4l:") || strings.Contains(strings.ToLower(dev.Type), "camera") {
			log.Debug("跳过摄像头设备", zap.String("name", dev.Name), zap.String("type", dev.Type))
			continue
		}

		deviceNameLower := strings.ToLower(dev.Name)
		deviceTypeLower := strings.ToLower(dev.Type)
		if strings.HasPrefix(deviceNameLower, "airscan:") ||
			strings.HasPrefix(deviceNameLower, "escl:") ||
			strings.HasPrefix(deviceNameLower, "epson") ||
			strings.HasPrefix(deviceNameLower, "hp:") ||
			strings.HasPrefix(deviceNameLower, "canon:") ||
			strings.Contains(deviceTypeLower, "printer") ||
			strings.Contains(deviceTypeLower, "scanner") ||
			strings.Contains(deviceTypeLower, "multi-function") ||
			strings.Contains(deviceTypeLower, "mfp") ||
			strings.Contains(deviceTypeLower, "wsd") ||
			strings.Contains(deviceTypeLower, "escl") {
			printerDevices = append(printerDevices, dev)
		} else {
			otherDevices = append(otherDevices, dev)
		}
	}

	devices := append(printerDevices, otherDevices...)
	if len(devices) == 0 {
		log.Warn("未找到可用的扫描设备（已排除摄像头）", zap.Int("total_devices", len(allDevices)))
		return fmt.Errorf("未找到可用的扫描设备")
	}

	log.Info("找到可用扫描设备", zap.Int("count", len(devices)), zap.Int("printer_devices", len(printerDevices)))
	for i, dev := range devices {
		isPrinter := i < len(printerDevices)
		log.Info("扫描设备", zap.Int("index", i+1), zap.String("name", dev.Name), zap.String("type", dev.Type), zap.Bool("is_printer", isPrinter))
	}

	// 如果未指定设备，优先选择打印机扫描设备
	if deviceName == "" {
		if len(printerDevices) > 0 {
			deviceName = printerDevices[0].Name
			log.Info("自动选择打印机扫描设备", zap.String("device", deviceName))
		} else if len(devices) > 0 {
			deviceName = devices[0].Name
			log.Warn("自动选择第一个可用设备（可能不是打印机扫描设备）", zap.String("device", deviceName))
		} else {
			return fmt.Errorf("未找到可用扫描设备")
		}
	}

	// 验证设备是否存在
	deviceFound := false
	for _, dev := range devices {
		if dev.Name == deviceName {
			deviceFound = true
			log.Info("设备验证成功", zap.String("device", deviceName))
			break
		}
	}
	if !deviceFound {
		log.Error("设备不存在", zap.String("device", deviceName))
		return fmt.Errorf("设备 '%s' 不存在", deviceName)
	}

	// 2. 准备输出文件
	if options.OutputFile == "" {
		timestamp := time.Now().Format("20060102_150405")
		options.OutputFile = fmt.Sprintf("scan_%s.pdf", timestamp)
		log.Info("生成默认文件名", zap.String("file", options.OutputFile))
	}
	log.Info("输出文件", zap.String("file", options.OutputFile))

	// 3. 构建扫描命令
	// 设置默认值
	if options.Resolution == 0 {
		options.Resolution = 300
	}
	if options.Format == "" {
		options.Format = "pdf"
	}
	if options.ColorMode == "" {
		options.ColorMode = "color"
	}
	if options.Source == "" {
		options.Source = "flatbed"
	}
	if options.PageSize == "" {
		options.PageSize = "A4"
	}

	log.Info("扫描参数",
		zap.Int("resolution", options.Resolution),
		zap.String("format", options.Format),
		zap.String("color_mode", options.ColorMode),
		zap.String("source", options.Source),
		zap.String("page_size", options.PageSize),
		zap.Bool("duplex", options.Duplex),
	)

	// 4. 执行扫描
	log.Info("启动扫描进程", zap.String("device", deviceName))
	args := []string{"-d", deviceName}

	// 查询设备支持的选项
	log.Info("查询设备支持的选项")
	queryCmd := exec.Command("scanimage", "-d", deviceName, "-A")
	queryOutput, queryErr := queryCmd.Output()

	supportsResolution := false
	supportsMode := false
	supportsSource := false
	deviceOptions := ""

	if queryErr == nil && len(queryOutput) > 0 {
		deviceOptions = string(queryOutput)
		log.Info("成功查询设备选项", zap.Int("size", len(deviceOptions)))

		resolutionPatterns := []string{"resolution", "--resolution", "resolution-x", "resolution-y"}
		for _, pattern := range resolutionPatterns {
			if strings.Contains(deviceOptions, pattern) {
				supportsResolution = true
				break
			}
		}

		modeOptionPatterns := []string{"mode", "color-mode", "color mode", "scan-mode", "scan mode"}
		for _, pattern := range modeOptionPatterns {
			if strings.Contains(deviceOptions, pattern) {
				lines := strings.Split(deviceOptions, "\n")
				for _, line := range lines {
					lineLower := strings.ToLower(line)
					if strings.Contains(lineLower, strings.ToLower(pattern)) && !strings.Contains(lineLower, "adf-mode") {
						supportsMode = true
						if strings.Contains(deviceOptions, "color") || strings.Contains(deviceOptions, "Color") || strings.Contains(deviceOptions, "RGB") {
							log.Info("检测到设备支持彩色扫描")
						}
						break
					}
				}
				if supportsMode {
					break
				}
			}
		}

		if strings.Contains(deviceOptions, "source") {
			supportsSource = true
		}

		log.Info("设备选项支持情况",
			zap.Bool("resolution", supportsResolution),
			zap.Bool("mode", supportsMode),
			zap.Bool("source", supportsSource),
		)
	} else {
		log.Warn("无法查询设备选项", zap.Error(queryErr))
		supportsResolution = false
		supportsMode = false
		supportsSource = false
	}

	// 设置分辨率
	resolutionSet := false
	if deviceOptions != "" {
		resolutionOptionNames := []string{"resolution", "resolution-x", "resolution-y", "x-resolution", "y-resolution"}
		for _, optName := range resolutionOptionNames {
			if strings.Contains(deviceOptions, optName) {
				args = append(args, fmt.Sprintf("--%s=%d", optName, options.Resolution))
				log.Info("设置分辨率参数", zap.String("option", optName), zap.Int("resolution", options.Resolution))
				resolutionSet = true
				break
			}
		}
		if !resolutionSet && supportsResolution {
			args = append(args, "--resolution", fmt.Sprintf("%d", options.Resolution))
			log.Info("设置分辨率", zap.Int("resolution", options.Resolution))
			resolutionSet = true
		}
	}
	if !resolutionSet {
		log.Warn("设备不支持分辨率选项，将使用设备默认值")
	}

	// 设置颜色模式
	colorModeSet := false
	if supportsMode {
		colorModeMap := map[string]string{
			"color":     "Color",
			"grayscale": "Gray",
			"lineart":   "Lineart",
		}
		if options.ColorMode == "color" {
			args = append(args, "--mode", "Color")
			log.Info("设置颜色模式", zap.String("mode", "Color"))
			colorModeSet = true
		} else if mode, ok := colorModeMap[options.ColorMode]; ok {
			args = append(args, "--mode", mode)
			log.Info("设置颜色模式", zap.String("mode", mode))
			colorModeSet = true
		}
	} else {
		log.Warn("设备可能不支持标准--mode选项")
		colorModeOptions := []string{"color-mode", "scan-mode", "mode"}
		for _, optName := range colorModeOptions {
			if strings.Contains(deviceOptions, optName) {
				if options.ColorMode == "color" {
					args = append(args, fmt.Sprintf("--%s=Color", optName))
					log.Info("尝试设置颜色模式", zap.String("option", optName))
					colorModeSet = true
					break
				}
			}
		}
		if !colorModeSet && options.ColorMode == "color" {
			args = append(args, "--mode", "Color")
			log.Info("尝试直接使用--mode Color")
			colorModeSet = true
		}
		if !colorModeSet {
			log.Warn("无法设置颜色模式，将使用设备默认值")
		}
	}

	if options.ColorMode == "color" && colorModeSet {
		log.Info("已设置为彩色扫描模式")
	} else if options.ColorMode == "color" && !colorModeSet {
		log.Warn("彩色模式可能未正确设置，扫描结果可能是黑白的")
	}

	// 设置扫描源
	if supportsSource {
		if options.Source == "adf" {
			args = append(args, "--source", "ADF")
			log.Info("设置扫描源", zap.String("source", "ADF"))
			if options.Duplex {
				args = append(args, "--adf-mode", "Duplex")
				log.Info("设置ADF模式", zap.String("mode", "Duplex"))
			} else {
				args = append(args, "--adf-mode", "Simplex")
				log.Info("设置ADF模式", zap.String("mode", "Simplex"))
			}
		} else {
			args = append(args, "--source", "Flatbed")
			log.Info("设置扫描源", zap.String("source", "Flatbed"))
		}
	} else {
		log.Warn("设备可能不支持--source选项，将使用默认扫描源")
	}

	// 设置输出格式和文件
	isBatchMode := options.Batch || (options.Source == "adf" && !strings.HasSuffix(strings.ToLower(options.OutputFile), ".pdf"))

	if isBatchMode {
		log.Info("批量扫描模式（适用于ADF多页扫描）")
		batchFormat := options.BatchFormat
		if batchFormat == "" {
			if options.Format == "jpeg" || options.Format == "jpg" {
				batchFormat = strings.TrimSuffix(options.OutputFile, ".jpg")
				batchFormat = strings.TrimSuffix(batchFormat, ".jpeg")
				if batchFormat == options.OutputFile {
					batchFormat = "scan_%03d.jpg"
				} else {
					batchFormat = batchFormat + "_%03d.jpg"
				}
			} else if options.Format == "png" {
				batchFormat = strings.TrimSuffix(options.OutputFile, ".png")
				if batchFormat == options.OutputFile {
					batchFormat = "scan_%03d.png"
				} else {
					batchFormat = batchFormat + "_%03d.png"
				}
			} else {
				batchFormat = "scan_%03d.jpg"
			}
		}

		args = append(args, "--batch", batchFormat)
		log.Info("设置批量扫描", zap.String("format", batchFormat))

		formatMap := map[string]string{"jpeg": "jpeg", "jpg": "jpeg", "png": "png"}
		if format, ok := formatMap[options.Format]; ok {
			args = append(args, "--format", format)
			log.Info("设置输出格式", zap.String("format", format))
		} else {
			args = append(args, "--format", "jpeg")
			log.Info("设置输出格式", zap.String("format", "jpeg"))
		}
	} else if options.Format == "pdf" {
		tempFile := strings.TrimSuffix(options.OutputFile, ".pdf") + ".tiff"
		args = append(args, "--format", "tiff")
		args = append(args, "-o", tempFile)
		log.Info("设置PDF输出", zap.String("temp_file", tempFile))
	} else {
		formatMap := map[string]string{"jpeg": "jpeg", "png": "png"}
		if format, ok := formatMap[options.Format]; ok {
			args = append(args, "--format", format)
			log.Info("设置输出格式", zap.String("format", format))
		} else {
			args = append(args, "--format", "pnm")
			log.Info("设置输出格式", zap.String("format", "pnm"))
		}
		args = append(args, "-o", options.OutputFile)
		log.Info("设置输出文件", zap.String("file", options.OutputFile))
	}

	log.Info("执行scanimage命令", zap.Strings("args", args))
	cmd = exec.Command("scanimage", args...)
	output, err = cmd.CombinedOutput()

	if err != nil {
		log.Error("扫描失败", zap.Error(err), zap.String("output", string(output)))

		errorStr := string(output)
		if strings.Contains(errorStr, "unrecognized option") || strings.Contains(errorStr, "Unknown option") {
			log.Info("检测到不支持的选项，尝试使用最小参数集")

			minimalArgs := []string{"-d", deviceName}
			if options.ColorMode == "color" {
				minimalArgs = append(minimalArgs, "--mode", "Color")
				log.Info("保留颜色模式设置")
			}

			if options.Format == "pdf" {
				tempFile := strings.TrimSuffix(options.OutputFile, ".pdf") + ".tiff"
				minimalArgs = append(minimalArgs, "--format", "tiff", "-o", tempFile)
			} else {
				formatMap := map[string]string{"jpeg": "jpeg", "png": "png"}
				if format, ok := formatMap[options.Format]; ok {
					minimalArgs = append(minimalArgs, "--format", format)
				}
				minimalArgs = append(minimalArgs, "-o", options.OutputFile)
			}

			log.Info("使用最小参数集重新扫描", zap.Strings("args", minimalArgs))
			cmd = exec.Command("scanimage", minimalArgs...)
			output, err = cmd.CombinedOutput()

			if err != nil {
				log.Error("简化命令也失败", zap.Error(err), zap.String("output", string(output)))
			} else {
				log.Info("使用最小参数集扫描成功")
			}
		}

		if err != nil {
			return fmt.Errorf("扫描失败: %w", err)
		}
	}

	log.Info("扫描完成")

	// 5. 处理批量扫描结果或PDF转换
	var finalOutputFile = options.OutputFile
	var batchFiles []string

	isBatchMode = options.Batch || (options.Source == "adf" && !strings.HasSuffix(strings.ToLower(options.OutputFile), ".pdf"))

	if isBatchMode {
		log.Info("处理批量扫描结果")
		batchFormat := options.BatchFormat
		if batchFormat == "" {
			if options.Format == "jpeg" || options.Format == "jpg" {
				batchFormat = "scan_%03d.jpg"
			} else if options.Format == "png" {
				batchFormat = "scan_%03d.png"
			} else {
				batchFormat = "scan_%03d.jpg"
			}
		}

		dir := "."
		basePattern := batchFormat
		if strings.Contains(batchFormat, "/") {
			parts := strings.Split(batchFormat, "/")
			dir = strings.Join(parts[:len(parts)-1], "/")
			basePattern = parts[len(parts)-1]
		}

		pattern := strings.ReplaceAll(basePattern, "%03d", "*")
		pattern = strings.ReplaceAll(pattern, "%d", "*")

		entries, err := os.ReadDir(dir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					name := entry.Name()
					if matched, _ := filepath.Match(pattern, name); matched {
						fullPath := filepath.Join(dir, name)
						batchFiles = append(batchFiles, fullPath)
					}
				}
			}
		}

		sort.Strings(batchFiles)

		if len(batchFiles) > 0 {
			log.Info("找到批量扫描文件", zap.Int("count", len(batchFiles)))
			for i, file := range batchFiles {
				if info, err := os.Stat(file); err == nil {
					log.Info("扫描文件", zap.Int("index", i+1), zap.String("file", file), zap.Int64("size", info.Size()))
				}
			}

			if options.Format == "pdf" || strings.HasSuffix(strings.ToLower(options.OutputFile), ".pdf") {
				log.Warn("批量扫描PDF合并功能待实现", zap.Int("file_count", len(batchFiles)))
			}
		} else {
			log.Warn("未找到批量扫描文件")
		}
	} else if options.Format == "pdf" {
		tempFile := strings.TrimSuffix(options.OutputFile, ".pdf") + ".tiff"
		if _, err := os.Stat(tempFile); err == nil {
			log.Info("转换为PDF格式", zap.String("input", tempFile), zap.String("output", options.OutputFile))
			if err := convertToPDF(tempFile, options.OutputFile, log); err != nil {
				log.Error("PDF转换失败", zap.Error(err))
				finalOutputFile = tempFile
			} else {
				log.Info("PDF转换成功")
				os.Remove(tempFile)
				log.Info("临时文件已删除", zap.String("file", tempFile))
				finalOutputFile = options.OutputFile
			}
		} else {
			finalOutputFile = options.OutputFile
		}
	}

	// 6. 验证最终输出文件
	log.Info("验证扫描结果")
	fileInfo, err := os.Stat(finalOutputFile)
	if err != nil {
		log.Error("输出文件不存在", zap.Error(err))
		return fmt.Errorf("输出文件不存在: %w", err)
	}

	log.Info("扫描流程完成",
		zap.String("file", finalOutputFile),
		zap.Int64("size", fileInfo.Size()),
	)

	return nil
}

// convertToPDF 将图像文件转换为PDF（使用ImageMagick或类似工具）
func convertToPDF(imageFile, pdfFile string, log *zap.Logger) error {
	log.Info("尝试使用ImageMagick convert命令（保持彩色）")
	cmd := exec.Command("convert",
		"-colorspace", "RGB",
		"-quality", "95",
		imageFile,
		pdfFile)
	output, err := cmd.CombinedOutput()
	if err == nil {
		log.Info("ImageMagick转换成功（已保持彩色模式）")
		return nil
	}
	log.Warn("ImageMagick失败", zap.Error(err), zap.String("output", string(output)))

	log.Info("尝试使用简化命令")
	cmd = exec.Command("convert", imageFile, pdfFile)
	output, err = cmd.CombinedOutput()
	if err == nil {
		log.Info("ImageMagick转换成功（使用简化命令）")
		return nil
	}
	log.Warn("ImageMagick简化命令也失败", zap.Error(err), zap.String("output", string(output)))

	log.Info("尝试使用img2pdf命令（保持原始颜色）")
	cmd = exec.Command("img2pdf", imageFile, "-o", pdfFile)
	output, err = cmd.CombinedOutput()
	if err == nil {
		log.Info("img2pdf转换成功（已保持原始颜色）")
		return nil
	}
	log.Error("img2pdf失败", zap.Error(err), zap.String("output", string(output)))

	return fmt.Errorf("无法转换为PDF，请安装ImageMagick (convert) 或 img2pdf")
}

// parseScanDevices 解析scanimage -L的输出
func parseScanDevices(output string) []ScanDeviceInfo {
	var devices []ScanDeviceInfo
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.Contains(line, "device `") && strings.Contains(line, "' is a ") {
			parts := strings.Split(line, "device `")
			if len(parts) < 2 {
				continue
			}

			devicePart := strings.Split(parts[1], "' is a ")
			if len(devicePart) < 2 {
				continue
			}

			deviceName := devicePart[0]
			deviceType := strings.TrimSpace(devicePart[1])

			devices = append(devices, ScanDeviceInfo{
				Name: deviceName,
				Type: deviceType,
			})
		}
	}

	return devices
}

// ListScanDevices 列出所有可用的扫描设备
func ListScanDevices(log *zap.Logger) ([]ScanDeviceInfo, error) {
	log.Info("正在查找扫描设备")

	cmd := exec.Command("scanimage", "-L")
	output, err := cmd.Output()
	if err != nil {
		log.Error("scanimage命令不可用", zap.Error(err))
		return nil, fmt.Errorf("scanimage不可用: %w", err)
	}

	devices := parseScanDevices(string(output))
	log.Info("找到扫描设备", zap.Int("count", len(devices)))
	return devices, nil
}

// PrintScanDevices 打印扫描设备列表
func PrintScanDevices(log *zap.Logger, devices []ScanDeviceInfo) {
	if len(devices) == 0 {
		log.Warn("未找到可用扫描设备")
		return
	}

	// 分类设备
	var printerDevices []ScanDeviceInfo
	var cameraDevices []ScanDeviceInfo
	var otherDevices []ScanDeviceInfo

	for _, dev := range devices {
		if strings.HasPrefix(dev.Name, "v4l:") || strings.Contains(strings.ToLower(dev.Type), "camera") {
			cameraDevices = append(cameraDevices, dev)
		} else if strings.HasPrefix(strings.ToLower(dev.Name), "airscan:") ||
			strings.HasPrefix(strings.ToLower(dev.Name), "escl:") ||
			strings.Contains(strings.ToLower(dev.Type), "printer") ||
			strings.Contains(strings.ToLower(dev.Type), "scanner") {
			printerDevices = append(printerDevices, dev)
		} else {
			otherDevices = append(otherDevices, dev)
		}
	}

	log.Info("打印机扫描设备（推荐）", zap.Int("count", len(printerDevices)))
	for i, dev := range printerDevices {
		fmt.Printf("  %d. %s\n", i+1, dev.Name)
		fmt.Printf("     类型: %s\n", dev.Type)
		fmt.Println()
	}

	if len(otherDevices) > 0 {
		log.Info("其他扫描设备", zap.Int("count", len(otherDevices)))
		for i, dev := range otherDevices {
			fmt.Printf("  %d. %s\n", i+1, dev.Name)
			fmt.Printf("     类型: %s\n", dev.Type)
			fmt.Println()
		}
	}

	if len(cameraDevices) > 0 {
		log.Info("摄像头设备（已排除）", zap.Int("count", len(cameraDevices)))
	}
}
