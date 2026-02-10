package doctor

import (
	"fmt"
	"strings"
)

const (
	minW   = 4
	maxDet = 60 // max detail/version column width to avoid overly wide tables
	sep    = 2
)

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func cellLen(s string) int {
	return len(strings.TrimSpace(s))
}

// collectToolsColWidths returns (nameW, statusW, detailW) and total content width.
func collectToolsColWidths(r *Report) (int, int, int, int) {
	nw, sw, dw := minW, minW, minW
	for _, t := range r.Tools {
		nw = max(nw, cellLen(t.Name))
		sw = max(sw, cellLen(string(t.Status)))
		var detail string
		if t.Version != "" {
			detail = t.Version
		} else {
			detail = t.Path
		}
		dw = max(dw, min(cellLen(detail), maxDet))
	}
	total := nw + sep + sw + sep + dw
	return nw, sw, dw, total
}

// collectSvcColWidths returns (nameW, statusW, detailW, portW) and total content width.
func collectSvcColWidths(r *Report) (int, int, int, int, int) {
	nw, sw, dw, pw := minW, minW, minW, minW
	for _, s := range r.Svc {
		nw = max(nw, cellLen(s.Name))
		sw = max(sw, cellLen(string(s.Status)))
		var detail string
		if s.Version != "" {
			detail = s.Version
		} else {
			detail = s.Path
		}
		dw = max(dw, min(cellLen(detail), maxDet))
		portInfo := ""
		if s.Port != "" {
			portInfo = "port " + s.Port + ": " + string(s.PortStatus)
		}
		pw = max(pw, min(cellLen(portInfo), maxDet))
	}
	total := nw + sep + sw + sep + dw + sep + pw
	return nw, sw, dw, pw, total
}

// Print writes a formatted report to stdout (English only).
func Print(r *Report) {
	// Header: dynamic width, at least 64
	headerContent := "CLI Doctor Report  " + r.OS + "/" + r.Arch
	osContent := "OS: " + r.OSDetail
	boxW := max(64, max(cellLen(headerContent), cellLen(osContent)))
	line := strings.Repeat("─", boxW)

	fmt.Println()
	fmt.Printf("  ╭%s╮\n", line)
	fmt.Printf("  │  %-*s│\n", boxW, trunc(headerContent, boxW))
	fmt.Printf("  │  %-*s│\n", boxW, trunc(osContent, boxW))
	fmt.Printf("  ╰%s╯\n", line)
	fmt.Println()

	// Tools: dynamic column widths; pad last column so row width = boxW
	nw, sw, dw, total := collectToolsColWidths(r)
	toolsBoxW := max(64, total)
	lastW := dw + max(0, toolsBoxW-total)
	line = strings.Repeat("─", toolsBoxW)
	fmt.Printf("  ┌─ Tools %s┐\n", strings.Repeat("─", max(0, toolsBoxW-8)))
	for _, t := range r.Tools {
		var detail string
		if t.Version != "" {
			detail = t.Version
		} else {
			detail = t.Path
		}
		fmt.Printf("  │  %-*s  %-*s  %-*s│\n", nw, t.Name, sw, t.Status, lastW, trunc(detail, lastW))
	}
	fmt.Printf("  └%s┘\n", line)
	fmt.Println()

	// Services: dynamic column widths; pad last column so row width = boxW
	snw, ssw, sdw, spw, stotal := collectSvcColWidths(r)
	svcBoxW := max(64, stotal)
	lastPortW := spw + max(0, svcBoxW-stotal)
	line = strings.Repeat("─", svcBoxW)
	fmt.Printf("  ┌─ Services (binary + default port) %s┐\n", strings.Repeat("─", max(0, svcBoxW-36)))
	for _, s := range r.Svc {
		var detail string
		if s.Version != "" {
			detail = s.Version
		} else {
			detail = s.Path
		}
		portInfo := ""
		if s.Port != "" {
			portInfo = "port " + s.Port + ": " + string(s.PortStatus)
		}
		fmt.Printf("  │  %-*s  %-*s  %-*s  %-*s│\n", snw, s.Name, ssw, s.Status, sdw, trunc(detail, sdw), lastPortW, trunc(portInfo, lastPortW))
	}
	fmt.Printf("  └%s┘\n", line)
	fmt.Println()
	fmt.Println("  Diagnosis complete.")
	fmt.Println()
}

// Run runs all registered checkers concurrently and prints the report.
func Run() {
	r := DefaultRegistry.Run()
	Print(r)
}

func trunc(s string, w int) string {
	s = strings.TrimSpace(s)
	if w <= 0 || len(s) <= w {
		return s
	}
	if w <= 3 {
		return s[:w]
	}
	return s[:w-3] + "..."
}
