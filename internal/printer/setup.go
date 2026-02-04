package printer

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

// AutoSetup 自动设置打印机
func AutoSetup(log *zap.Logger) error {
	log.Info("开始打印机自动发现和配置")

	// 自动发现打印机
	printers, err := AutoDiscoverAndAdd(log)
	if err != nil {
		return fmt.Errorf("自动发现失败: %w", err)
	}

	if len(printers) == 0 {
		log.Warn("未发现任何打印机")
		return nil
	}

	// 显示已添加的打印机列表
	cupsPrinters, err := getCUPSPrinters(log)
	if err == nil {
		log.Info("已配置的打印机列表", zap.Int("count", len(cupsPrinters)))
		for i, name := range cupsPrinters {
			log.Info("打印机", zap.Int("index", i+1), zap.String("name", name))
		}
	} else {
		log.Warn("无法获取CUPS打印机列表", zap.Error(err))
	}

	return nil
}

// getCUPSPrinters 获取CUPS中已配置的打印机列表
func getCUPSPrinters(log *zap.Logger) ([]string, error) {
	cmd := exec.Command("lpstat", "-p")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var printers []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "printer ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				printers = append(printers, parts[1])
			}
		}
	}

	return printers, nil
}

// InteractiveSelectPrinter 交互式选择打印机
func InteractiveSelectPrinter(log *zap.Logger) (string, error) {
	printers, err := getCUPSPrinters(log)
	if err != nil {
		return "", fmt.Errorf("获取打印机列表失败: %w", err)
	}

	if len(printers) == 0 {
		return "", fmt.Errorf("未找到已配置的打印机")
	}

	log.Info("请选择要使用的打印机", zap.Int("count", len(printers)))
	for i, name := range printers {
		fmt.Printf("  %d. %s\n", i+1, name)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("\n请输入序号 (1-%d): ", len(printers))
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	index, err := strconv.Atoi(input)
	if err != nil || index < 1 || index > len(printers) {
		return "", fmt.Errorf("无效的选择")
	}

	selected := printers[index-1]
	log.Info("已选择打印机", zap.String("printer", selected))
	return selected, nil
}
