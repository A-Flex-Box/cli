package doctor

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"golang.org/x/term"
)

const maxWidth = 100

var (
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	danger    = lipgloss.AdaptiveColor{Light: "#F25D94", Dark: "#F5508D"}
	muted     = lipgloss.AdaptiveColor{Light: "#6B6B6B", Dark: "#9B9B9B"}

	docStyle = lipgloss.NewStyle().
		Margin(1, 2).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(highlight)

	titleStyle = lipgloss.NewStyle().
		Foreground(special).
		Bold(true).
		Padding(0, 1).
		Background(subtle)
)

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

func getTermWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return maxWidth
	}
	if w > maxWidth {
		return maxWidth
	}
	return w - 4
}

// Print writes a formatted report to stdout using Lip Gloss (English only).
func Print(r *Report) {
	termW := getTermWidth()
	docStyle := docStyle.Width(termW)

	var blocks []string

	// Header
	header := lipgloss.NewStyle().Foreground(highlight).Bold(true).Render("CLI Doctor Report")
	sysInfo := lipgloss.NewStyle().Foreground(muted).Render(r.OS + "/" + r.Arch + "  •  " + r.OSDetail)
	blocks = append(blocks, lipgloss.JoinVertical(lipgloss.Left, header, sysInfo))
	blocks = append(blocks, "")

	// Tools section
	if len(r.Tools) > 0 {
		rows := make([][]string, 0, len(r.Tools))
		for _, t := range r.Tools {
			detail := t.Version
			if detail == "" {
				detail = t.Path
			}
			rows = append(rows, []string{t.Name, string(t.Status), trunc(detail, 45)})
		}
		toolsTitle := titleStyle.Render("Development Tools")
		toolsTable := renderTable(
			[]string{"Name", "Status", "Version"},
			rows,
			func(row, col int, cell string) lipgloss.Style {
				s := lipgloss.NewStyle().Padding(0, 1)
				if col == 0 {
					s = s.Foreground(lipgloss.Color("205"))
				}
				if col == 1 {
					if cell == string(InstallStatusInstalled) {
						s = s.Foreground(special)
					} else {
						s = s.Foreground(danger)
					}
				}
				if col == 2 {
					s = s.Foreground(muted)
				}
				return s
			},
		)
		blocks = append(blocks, lipgloss.JoinVertical(lipgloss.Left, toolsTitle, toolsTable))
		blocks = append(blocks, "")
	}

	// Services section
	if len(r.Svc) > 0 {
		rows := make([][]string, 0, len(r.Svc))
		for _, s := range r.Svc {
			detail := s.Version
			if detail == "" {
				detail = s.Path
			}
			portInfo := ""
			if s.Port != "" {
				portInfo = fmt.Sprintf("port %s: %s", s.Port, s.PortStatus)
			}
			rows = append(rows, []string{s.Name, string(s.Status), trunc(detail, 35), portInfo})
		}
		svcTitle := titleStyle.Render("Infrastructure Services")
		svcTable := renderTable(
			[]string{"Name", "Status", "Version", "Port"},
			rows,
			func(row, col int, cell string) lipgloss.Style {
				s := lipgloss.NewStyle().Padding(0, 1)
				if col == 0 {
					s = s.Foreground(lipgloss.Color("205"))
				}
				if col == 1 {
					if cell == string(InstallStatusInstalled) {
						s = s.Foreground(special)
					} else {
						s = s.Foreground(danger)
					}
				}
				if col == 2 {
					s = s.Foreground(muted)
				}
				if col == 3 {
					if strings.Contains(cell, "listening") && !strings.Contains(cell, "not") {
						s = s.Foreground(special)
					} else {
						s = s.Foreground(muted)
					}
				}
				return s
			},
		)
		blocks = append(blocks, lipgloss.JoinVertical(lipgloss.Left, svcTitle, svcTable))
		blocks = append(blocks, "")
	}

	blocks = append(blocks, lipgloss.NewStyle().Foreground(muted).Render("Diagnosis complete."))

	ui := docStyle.Render(lipgloss.JoinVertical(lipgloss.Left, blocks...))
	fmt.Println(ui)
}

func renderTable(headers []string, rows [][]string, styleFunc func(row, col int, cell string) lipgloss.Style) string {
	// Reserve space for doc margin(2*2) + padding(2*2) + border(2) + slack = 20
	availW := getTermWidth() - 20
	if availW < 50 {
		availW = 50
	}
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(highlight)).
		Headers(headers...).
		Rows(rows...).
		Width(availW)

	headerStyle := lipgloss.NewStyle().Foreground(highlight).Bold(true).Padding(0, 1)
	t = t.StyleFunc(func(row, col int) lipgloss.Style {
		if row == table.HeaderRow {
			return headerStyle
		}
		var cell string
		if row < len(rows) && col < len(rows[row]) {
			cell = rows[row][col]
		}
		return styleFunc(row, col, cell)
	})

	return t.String()
}

// Run runs all registered checkers concurrently and prints the report.
func Run() {
	r := DefaultRegistry.Run()
	Print(r)
}
