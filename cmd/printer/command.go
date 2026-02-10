package printer

import (
	"github.com/A-Flex-Box/cli/internal/logger"
	printerpkg "github.com/A-Flex-Box/cli/internal/printer"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// NewCmd returns the printer command.
func NewCmd() *cobra.Command {
	var (
		printerSetupMode   bool
		printerName        string
		pdfFile            string
		pdfURL             string
		autoSelect         bool
		copies             int
		sides              string
		colorMode          string
		mediaSource        string
		useCUPS            bool
		scanMode           bool
		scanDevice         string
		scanOutput         string
		scanResolution     int
		scanColor          string
		scanSource         string
		scanFormat         string
		scanBatch          bool
		scanBatchFormat    string
		listScanDevices    bool
	)

	cmd := &cobra.Command{
		Use:   "printer",
		Short: "打印机和扫描仪管理工具",
		Long: `打印机和扫描仪管理工具，支持：
- 自动发现和配置网络打印机
- 打印PDF文件（支持本地文件和远程URL）
- 文档扫描（支持平板和ADF扫描）
- 打印选项控制（份数、单双面、颜色、纸张来源等）`,
		Run: func(cmd *cobra.Command, args []string) {
			log := logger.NewLogger()
			defer log.Sync()

			if printerSetupMode {
				if err := printerpkg.AutoSetup(log); err != nil {
					log.Fatal("自动配置失败", zap.Error(err))
				}
				return
			}

			if listScanDevices {
				devices, err := printerpkg.ListScanDevices(log)
				if err != nil {
					log.Fatal("查找扫描设备失败", zap.Error(err))
				}
				if len(devices) == 0 {
					log.Warn("未找到可用扫描设备")
					return
				}
				printerpkg.PrintScanDevices(log, devices)
				return
			}

			if scanMode {
				scanOptions := printerpkg.ScanOptions{
					OutputFile:  scanOutput,
					Format:      scanFormat,
					Resolution:  scanResolution,
					ColorMode:   scanColor,
					Source:      scanSource,
					PageSize:    "A4",
					Batch:       scanBatch,
					BatchFormat: scanBatchFormat,
				}

				if scanOptions.Source == "adf" && !scanOptions.Batch && scanOptions.Format != "pdf" {
					scanOptions.Batch = true
					log.Info("检测到ADF扫描，自动启用批量扫描模式")
				}

				if err := printerpkg.ScanDocument(scanDevice, scanOptions, log); err != nil {
					log.Fatal("扫描失败", zap.Error(err))
				}
				return
			}

			if pdfFile == "" && pdfURL == "" {
				cmd.Help()
				return
			}

			var selectedPrinter string
			var err error
			if printerName != "" {
				selectedPrinter = printerName
			} else if autoSelect {
				selectedPrinter, err = printerpkg.AutoSelectPrinter(log)
				if err != nil {
					log.Fatal("自动选择打印机失败", zap.Error(err))
				}
			} else {
				selectedPrinter, err = printerpkg.InteractiveSelectPrinter(log)
				if err != nil {
					log.Fatal("选择打印机失败", zap.Error(err))
				}
			}

			printOptions := printerpkg.PrintOptions{
				Copies:      copies,
				Sides:       sides,
				ColorMode:   colorMode,
				MediaSource: mediaSource,
				UseCUPS:     useCUPS,
			}

			if err := printerpkg.ValidatePrintOptions(printOptions, log); err != nil {
				log.Fatal("打印选项验证失败", zap.Error(err))
			}

			if pdfURL != "" {
				if err := printerpkg.PrintFromURL(pdfURL, selectedPrinter, printOptions, log); err != nil {
					log.Fatal("远程打印失败", zap.Error(err))
				}
			} else {
				if err := printerpkg.PrintFile(pdfFile, selectedPrinter, printOptions, log); err != nil {
					log.Fatal("打印失败", zap.Error(err))
				}
			}
		},
	}

	cmd.Flags().BoolVar(&printerSetupMode, "setup", false, "自动发现并配置打印机")
	cmd.Flags().StringVarP(&pdfFile, "file", "f", "", "要打印的PDF文件路径")
	cmd.Flags().StringVarP(&pdfURL, "url", "u", "", "要打印的PDF文件URL（会下载到临时目录）")
	cmd.Flags().StringVarP(&printerName, "printer", "p", "", "指定打印机名称（如果不指定，将自动选择）")
	cmd.Flags().BoolVarP(&autoSelect, "auto", "a", false, "自动选择第一台可用打印机")
	cmd.Flags().IntVar(&copies, "copies", 1, "打印份数 (1-999)")
	cmd.Flags().StringVar(&sides, "sides", "one-sided", "单双面设置: one-sided(单面), two-sided-long-edge(双面长边), two-sided-short-edge(双面短边)")
	cmd.Flags().StringVar(&colorMode, "color", "auto", "颜色模式: auto(自动), color(彩色), monochrome(黑白)")
	cmd.Flags().StringVar(&mediaSource, "source", "auto", "纸张来源: auto(自动), manual(手动进纸), adf(自动文档进纸器), tray-1(纸盒1), tray-2(纸盒2), top(顶部), bottom(底部)")
	cmd.Flags().BoolVar(&useCUPS, "cups", false, "使用CUPS lp命令打印（支持所有选项，包括颜色设置）")
	cmd.Flags().BoolVar(&scanMode, "scan", false, "扫描模式：扫描文档并保存为PDF")
	cmd.Flags().StringVar(&scanDevice, "scan-device", "", "扫描设备名称（如果不指定，将自动选择）")
	cmd.Flags().StringVar(&scanOutput, "scan-output", "", "扫描输出文件路径（默认: scan_YYYYMMDD_HHMMSS.pdf）")
	cmd.Flags().IntVar(&scanResolution, "scan-resolution", 300, "扫描分辨率DPI: 150, 200, 300, 600 (默认: 300)")
	cmd.Flags().StringVar(&scanColor, "scan-color", "color", "扫描颜色模式: color(彩色,默认), grayscale(灰度), lineart(线条图)")
	cmd.Flags().StringVar(&scanSource, "scan-source", "flatbed", "扫描源: flatbed(平板扫描,默认), adf(自动文档进纸器/ADF扫描)")
	cmd.Flags().StringVar(&scanFormat, "scan-format", "pdf", "输出格式: pdf, jpeg, png (默认: pdf)")
	cmd.Flags().BoolVar(&scanBatch, "scan-batch", false, "批量扫描模式（适用于ADF多页扫描，自动扫描所有页面）")
	cmd.Flags().StringVar(&scanBatchFormat, "scan-batch-format", "", "批量扫描文件名格式（如 scan_%03d.jpg，默认自动生成）")
	cmd.Flags().BoolVar(&listScanDevices, "list-scan-devices", false, "列出所有可用的扫描设备")

	return cmd
}
