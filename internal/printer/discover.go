package printer

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/phin1x/go-ipp"
	"go.uber.org/zap"
)

// DiscoverPrinters 扫描网络上的IPP打印机
func DiscoverPrinters(log *zap.Logger) ([]PrinterInfo, error) {
	var printers []PrinterInfo

	log.Info("开始扫描网络打印机")
	log.Info("获取本地网络接口信息")
	localIPs := getLocalIPs(log)
	log.Info("检测到本地IP地址", zap.Strings("ips", localIPs))
	log.Info("扫描每个网段的IPP端口(631)")

	// 扫描每个本地IP的网段
	for i, localIP := range localIPs {
		log.Info("处理本地IP", zap.Int("index", i+1), zap.String("ip", localIP))
		ip := net.ParseIP(localIP)
		if ip == nil {
			log.Warn("跳过无效的IP地址格式", zap.String("ip", localIP))
			continue
		}

		log.Info("计算网段范围")
		network := getNetwork(ip)
		if network == nil {
			log.Warn("无法计算网段", zap.String("ip", localIP))
			continue
		}
		log.Info("网段范围", zap.String("network", network.String()), zap.Int("max_scan", 50))

		// 扫描该网段的所有IP的631端口
		discovered := scanNetworkForIPP(network, log)
		log.Info("网段扫描完成", zap.String("network", network.String()), zap.Int("found", len(discovered)))
		printers = append(printers, discovered...)
	}

	// 方法2: 使用CUPS命令发现打印机（如果可用）
	log.Info("通过CUPS命令发现打印机")
	log.Info("运行 lpinfo -v 命令")
	cupsPrinters, err := discoverViaCUPS(log)
	if err == nil {
		log.Info("CUPS发现打印机", zap.Int("count", len(cupsPrinters)))
		printers = append(printers, cupsPrinters...)
	} else {
		log.Warn("CUPS发现失败", zap.Error(err))
	}

	// 去重
	log.Info("去重打印机列表", zap.Int("before", len(printers)))
	printers = deduplicatePrinters(printers)
	log.Info("去重完成", zap.Int("after", len(printers)))

	return printers, nil
}

// getLocalIPs 获取所有本地IP地址
func getLocalIPs(log *zap.Logger) []string {
	var ips []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Warn("获取网络接口地址失败", zap.Error(err))
		return ips
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}
	return ips
}

// getNetwork 获取IP所在的网段
func getNetwork(ip net.IP) *net.IPNet {
	// 假设是/24网段
	mask := net.CIDRMask(24, 32)
	return &net.IPNet{
		IP:   ip.Mask(mask),
		Mask: mask,
	}
}

// scanNetworkForIPP 扫描网段中的IPP打印机
func scanNetworkForIPP(network *net.IPNet, log *zap.Logger) []PrinterInfo {
	var printers []PrinterInfo

	maxScan := 50
	scanned := 0
	found := 0

	for ip := network.IP.Mask(network.Mask); network.Contains(ip) && scanned < maxScan; incrementIP(ip) {
		scanned++
		targetIP := ip.String()

		// 每10个IP显示一次进度
		if scanned%10 == 0 {
			log.Info("扫描进度", zap.Int("scanned", scanned), zap.Int("max", maxScan), zap.Int("found", found))
		}

		log.Debug("检查IPP端口", zap.String("ip", targetIP), zap.Int("port", 631))
		if isIPPPrinter(targetIP, 631) {
			log.Info("发现IPP端口开放", zap.String("ip", targetIP))
			printer, err := getPrinterInfo(targetIP, 631, log)
			if err == nil {
				log.Info("发现打印机", zap.String("name", printer.Name), zap.String("ip", targetIP))
				found++
				printers = append(printers, printer)
			} else {
				log.Warn("获取打印机信息失败", zap.String("ip", targetIP), zap.Error(err))
			}
		}
	}

	log.Info("扫描完成", zap.Int("scanned", scanned), zap.Int("found", found))
	return printers
}

// isIPPPrinter 检查指定IP和端口是否是IPP打印机
func isIPPPrinter(ip string, port int) bool {
	address := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// getPrinterInfo 获取打印机详细信息
func getPrinterInfo(ip string, port int, log *zap.Logger) (PrinterInfo, error) {
	printer := PrinterInfo{
		IP:   ip,
		Port: port,
		URI:  fmt.Sprintf("ipp://%s:%d/ipp/print", ip, port),
	}

	log.Info("创建IPP客户端", zap.String("ip", ip), zap.Int("port", port))
	client := ipp.NewIPPClient(ip, port, "", "", false)

	log.Info("构造 Get-Printer-Attributes IPP请求")
	req := ipp.NewRequest(ipp.OperationGetPrinterAttributes, 1)
	req.OperationAttributes[ipp.AttributePrinterURI] = printer.URI
	req.OperationAttributes[ipp.AttributeRequestedAttributes] = []string{
		ipp.AttributePrinterName,
		ipp.AttributePrinterMakeAndModel,
		ipp.AttributePrinterLocation,
		ipp.AttributePrinterState,
		ipp.AttributePrinterIsAcceptingJobs,
	}

	url := fmt.Sprintf("http://%s:%d/ipp/print", ip, port)
	log.Info("发送IPP请求", zap.String("url", url))
	resp, err := client.SendRequest(url, req, nil)
	if err != nil {
		log.Error("IPP请求失败", zap.Error(err))
		return printer, err
	}

	log.Info("收到IPP响应", zap.Int16("status_code", resp.StatusCode))
	if resp.StatusCode != ipp.StatusOk {
		log.Warn("非成功状态码", zap.Int16("status_code", resp.StatusCode))
	}

	log.Info("解析IPP响应属性")
	if len(resp.PrinterAttributes) > 0 {
		attrs := resp.PrinterAttributes[0]

		if val, ok := attrs[ipp.AttributePrinterName]; ok && len(val) > 0 {
			printer.Name = fmt.Sprintf("%v", val[0].Value)
			log.Info("获取打印机名称", zap.String("name", printer.Name))
		}
		if val, ok := attrs[ipp.AttributePrinterMakeAndModel]; ok && len(val) > 0 {
			printer.MakeAndModel = fmt.Sprintf("%v", val[0].Value)
			log.Info("获取打印机型号", zap.String("model", printer.MakeAndModel))
		}
		if val, ok := attrs[ipp.AttributePrinterLocation]; ok && len(val) > 0 {
			printer.Location = fmt.Sprintf("%v", val[0].Value)
			log.Info("获取打印机位置", zap.String("location", printer.Location))
		}
		if val, ok := attrs[ipp.AttributePrinterState]; ok && len(val) > 0 {
			printer.State = fmt.Sprintf("%v", val[0].Value)
			log.Info("获取打印机状态", zap.String("state", printer.State))
		}
		if val, ok := attrs[ipp.AttributePrinterIsAcceptingJobs]; ok && len(val) > 0 {
			if b, ok := val[0].Value.(bool); ok {
				printer.IsAcceptingJobs = b
				log.Info("获取接受任务状态", zap.Bool("accepting", printer.IsAcceptingJobs))
			}
		}
	} else {
		log.Warn("响应中未包含打印机属性")
	}

	// 如果没有名称，使用IP作为名称
	if printer.Name == "" {
		printer.Name = fmt.Sprintf("Printer_%s", ip)
		log.Info("使用默认名称", zap.String("name", printer.Name))
	}

	log.Info("成功获取打印机信息", zap.String("name", printer.Name))
	return printer, nil
}

// discoverViaCUPS 通过CUPS命令发现打印机
func discoverViaCUPS(log *zap.Logger) ([]PrinterInfo, error) {
	var printers []PrinterInfo

	log.Info("运行 lpinfo -v 命令")
	cmd := exec.Command("lpinfo", "-v")
	output, err := cmd.Output()
	if err != nil {
		log.Error("命令执行失败", zap.Error(err))
		return printers, err
	}

	log.Info("命令执行成功", zap.Int("output_size", len(output)))
	log.Info("解析命令输出，查找IPP打印机URI")

	lines := strings.Split(string(output), "\n")
	ippURIs := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "network ipp://") {
			ippURIs++
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				uri := parts[1]
				log.Info("发现IPP URI", zap.String("uri", uri))
				if ip, port, err := parseIPPURI(uri); err == nil {
					log.Info("解析URI", zap.String("ip", ip), zap.Int("port", port))
					printer, err := getPrinterInfo(ip, port, log)
					if err == nil {
						printers = append(printers, printer)
					} else {
						log.Warn("获取打印机信息失败", zap.Error(err))
					}
				} else {
					log.Warn("URI解析失败", zap.Error(err))
				}
			}
		}
	}
	log.Info("CUPS发现完成", zap.Int("uris", ippURIs), zap.Int("printers", len(printers)))
	return printers, nil
}

// parseIPPURI 解析IPP URI，提取IP和端口
func parseIPPURI(uri string) (string, int, error) {
	uri = strings.TrimPrefix(uri, "ipp://")
	parts := strings.Split(uri, ":")
	if len(parts) < 2 {
		return "", 0, fmt.Errorf("invalid URI format")
	}

	ip := parts[0]
	portPart := strings.Split(parts[1], "/")[0]

	var port int
	fmt.Sscanf(portPart, "%d", &port)
	if port == 0 {
		port = 631 // 默认端口
	}

	return ip, port, nil
}

// deduplicatePrinters 去重打印机列表
func deduplicatePrinters(printers []PrinterInfo) []PrinterInfo {
	seen := make(map[string]bool)
	var unique []PrinterInfo

	for _, p := range printers {
		key := fmt.Sprintf("%s:%d", p.IP, p.Port)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, p)
		}
	}

	return unique
}

// incrementIP 递增IP地址
func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// AddPrinterToCUPS 自动添加打印机到CUPS
func AddPrinterToCUPS(printer PrinterInfo, log *zap.Logger) error {
	log.Info("添加打印机到CUPS", zap.String("name", printer.Name))

	safeName := sanitizePrinterName(printer.Name)
	if safeName != printer.Name {
		log.Info("名称转换", zap.String("from", printer.Name), zap.String("to", safeName))
	}

	deviceURI := fmt.Sprintf("ipp://%s:%d/ipp/print", printer.IP, printer.Port)
	log.Info("构建设备URI", zap.String("uri", deviceURI))

	// 尝试使用everywhere驱动
	log.Info("使用 everywhere 驱动添加打印机")
	cmd := exec.Command("lpadmin",
		"-p", safeName,
		"-E",
		"-v", deviceURI,
		"-m", "everywhere",
		"-L", printer.Location,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Warn("everywhere驱动失败", zap.Error(err), zap.String("output", string(output)))
		// 尝试raw驱动
		log.Info("尝试使用 raw 驱动")
		cmd = exec.Command("lpadmin",
			"-p", safeName,
			"-E",
			"-v", deviceURI,
			"-m", "raw",
		)
		output, err = cmd.CombinedOutput()
		if err != nil {
			log.Error("raw驱动也失败", zap.Error(err), zap.String("output", string(output)))
			return fmt.Errorf("添加打印机失败: %w, 输出: %s", err, string(output))
		}
		log.Info("raw驱动添加成功")
	} else {
		log.Info("everywhere驱动添加成功")
	}

	// 验证打印机是否已添加
	log.Info("验证打印机是否已添加")
	verifyCmd := exec.Command("lpstat", "-p", safeName)
	verifyOutput, verifyErr := verifyCmd.Output()
	if verifyErr == nil {
		log.Info("验证成功", zap.String("printer", safeName))
		if len(verifyOutput) > 0 {
			log.Info("打印机状态", zap.String("status", strings.TrimSpace(string(verifyOutput))))
		}
	} else {
		log.Warn("验证警告", zap.Error(verifyErr))
	}

	log.Info("打印机添加完成", zap.String("name", printer.Name), zap.String("cups_name", safeName))
	return nil
}

// sanitizePrinterName 清理打印机名称，使其符合CUPS命名规范
func sanitizePrinterName(name string) string {
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, ":", "_")
	name = strings.ReplaceAll(name, "*", "_")
	name = strings.ReplaceAll(name, "?", "_")
	name = strings.ReplaceAll(name, "\"", "_")
	name = strings.ReplaceAll(name, "<", "_")
	name = strings.ReplaceAll(name, ">", "_")
	name = strings.ReplaceAll(name, "|", "_")

	if len(name) > 127 {
		name = name[:127]
	}

	return name
}

// AutoDiscoverAndAdd 自动发现并添加所有打印机
func AutoDiscoverAndAdd(log *zap.Logger) ([]PrinterInfo, error) {
	log.Info("开始自动发现和添加打印机流程")

	// 1. 发现打印机
	log.Info("发现网络打印机")
	printers, err := DiscoverPrinters(log)
	if err != nil {
		log.Error("发现打印机失败", zap.Error(err))
		return nil, fmt.Errorf("发现打印机失败: %w", err)
	}

	if len(printers) == 0 {
		log.Warn("未发现任何打印机")
		return printers, nil
	}

	log.Info("发现打印机列表", zap.Int("count", len(printers)))
	for i, p := range printers {
		log.Info("打印机信息",
			zap.Int("index", i+1),
			zap.String("name", p.Name),
			zap.String("ip", p.IP),
			zap.Int("port", p.Port),
			zap.String("uri", p.URI),
			zap.String("model", p.MakeAndModel),
			zap.String("location", p.Location),
			zap.String("state", p.State),
			zap.Bool("accepting_jobs", p.IsAcceptingJobs),
		)
	}

	// 2. 自动添加到CUPS
	log.Info("添加打印机到CUPS", zap.Int("count", len(printers)))
	var added []PrinterInfo

	for i, printer := range printers {
		log.Info("处理打印机", zap.Int("index", i+1), zap.Int("total", len(printers)), zap.String("name", printer.Name))
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		done := make(chan error, 1)
		go func(p PrinterInfo) {
			done <- AddPrinterToCUPS(p, log)
		}(printer)

		select {
		case err := <-done:
			cancel()
			if err != nil {
				log.Error("添加失败", zap.String("name", printer.Name), zap.Error(err))
			} else {
				added = append(added, printer)
			}
		case <-ctx.Done():
			cancel()
			log.Warn("添加超时", zap.String("name", printer.Name))
		}
	}

	log.Info("添加结果汇总", zap.Int("total", len(printers)), zap.Int("success", len(added)), zap.Int("failed", len(printers)-len(added)))
	return added, nil
}
