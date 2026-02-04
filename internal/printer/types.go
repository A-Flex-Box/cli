package printer

// PrintOptions 打印选项
type PrintOptions struct {
	Copies      int    // 份数
	Sides       string // 单双面: one-sided, two-sided-long-edge, two-sided-short-edge
	ColorMode   string // 颜色模式: auto, color, monochrome
	MediaSource string // 纸张来源: auto, manual, adf, tray-1, tray-2, etc.
	UseCUPS     bool   // 使用CUPS lp命令
}

// ScanOptions 扫描选项
type ScanOptions struct {
	OutputFile  string // 输出文件路径
	Format      string // 输出格式: pdf, jpeg, png
	Resolution  int    // 分辨率 (DPI): 150, 200, 300, 600
	ColorMode   string // 颜色模式: color, grayscale, lineart
	Source      string // 扫描源: flatbed(平板), adf(自动文档进纸器)
	PageSize    string // 页面大小: A4, Letter, Legal
	Duplex      bool   // 双面扫描
	Batch       bool   // 批量扫描模式（多页扫描，适用于ADF）
	BatchFormat string // 批量扫描文件名格式（如 scan_%03d.jpg）
}

// PrinterInfo 打印机信息
type PrinterInfo struct {
	Name            string
	URI             string
	IP              string
	Port            int
	MakeAndModel    string
	Location        string
	State           string
	IsAcceptingJobs bool
}

// ScanDeviceInfo 扫描设备信息
type ScanDeviceInfo struct {
	Name string
	Type string
}
