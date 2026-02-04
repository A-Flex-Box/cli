package printer

import "fmt"

// getSidesDescription 获取单双面设置的描述
func getSidesDescription(sides string) string {
	descriptions := map[string]string{
		"one-sided":              "单面",
		"two-sided-long-edge":    "双面长边翻转",
		"two-sided-short-edge":   "双面短边翻转",
	}
	if desc, ok := descriptions[sides]; ok {
		return fmt.Sprintf("%s (%s)", desc, sides)
	}
	return sides
}

// getColorModeDescription 获取颜色模式的描述
func getColorModeDescription(colorMode string) string {
	descriptions := map[string]string{
		"auto":       "自动",
		"color":      "彩色",
		"monochrome": "黑白",
	}
	if desc, ok := descriptions[colorMode]; ok {
		return fmt.Sprintf("%s (%s)", desc, colorMode)
	}
	return colorMode
}

// getMediaSourceDescription 获取纸张来源的描述
func getMediaSourceDescription(source string) string {
	descriptions := map[string]string{
		"auto":   "自动选择",
		"manual": "手动进纸",
		"adf":    "自动文档进纸器 (ADF)",
		"tray-1": "纸盒1",
		"tray-2": "纸盒2",
		"tray-3": "纸盒3",
		"tray-4": "纸盒4",
		"top":    "顶部进纸器",
		"middle": "中部进纸器",
		"bottom": "底部进纸器",
	}
	if desc, ok := descriptions[source]; ok {
		return fmt.Sprintf("%s (%s)", desc, source)
	}
	return source
}

// getSourceDescription 获取扫描源的描述
func getSourceDescription(source string) string {
	descriptions := map[string]string{
		"flatbed": "平板扫描 (Flatbed)",
		"adf":     "自动文档进纸器 (ADF - Automatic Document Feeder)",
	}
	if desc, ok := descriptions[source]; ok {
		return desc
	}
	return source
}

// getColorModeDescriptionForScan 获取扫描颜色模式的描述
func getColorModeDescriptionForScan(colorMode string) string {
	descriptions := map[string]string{
		"color":     "彩色",
		"grayscale": "灰度",
		"lineart":   "线条图",
	}
	if desc, ok := descriptions[colorMode]; ok {
		return fmt.Sprintf("%s (%s)", desc, colorMode)
	}
	return colorMode
}
