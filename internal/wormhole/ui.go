package wormhole

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	uiHighlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	uiSpecial   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	uiMuted     = lipgloss.AdaptiveColor{Light: "#6B6B6B", Dark: "#9B9B9B"}
)

// ProgressMsg is sent when transfer progress updates.
type ProgressMsg struct {
	Current, Total int64
}

// DoneMsg is sent when transfer completes.
type DoneMsg struct {
	Err    error
	Result *ReceiveResult // optional, for receive: show in UI
}

// transferModel holds the Bubble Tea model for a transfer.
type transferModel struct {
	progress   progress.Model
	current    int64
	total      int64
	title      string
	code       string // pairing code to display while waiting
	ch         <-chan tea.Msg
	err        error
	doneResult *ReceiveResult // set when DoneMsg has Result, shown in View
}

func (m transferModel) Init() tea.Cmd {
	return waitForTransferMsg(m.ch)
}

func (m transferModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// q, Esc, Ctrl+C: always allow exit (fixes "Press any key" being ignored during wait)
		switch msg.String() {
		case "q", "Q", "esc", "ctrl+c":
			return m, tea.Quit
		}
		done := m.err != nil || (m.total > 0 && m.current >= m.total)
		if done {
			return m, tea.Quit
		}
		return m, nil

	case tea.WindowSizeMsg:
		// Keep progress bar single-line: bar+percent must fit in box (52 visible chars)
		w := msg.Width - 6
		if w > 46 {
			w = 46
		}
		if w < 20 {
			w = 20
		}
		m.progress.Width = w
		return m, nil

	case ProgressMsg:
		m.current = msg.Current
		m.total = msg.Total
		return m, waitForTransferMsg(m.ch)

	case progress.FrameMsg:
		prog, cmd := m.progress.Update(msg)
		m.progress = prog.(progress.Model)
		return m, cmd

	case DoneMsg:
		m.err = msg.Err
		m.doneResult = msg.Result
		m.current = m.total
		if m.total == 0 {
			m.total = 1
		}
		// Don't quit: show result in View, wait for q/Esc
		return m, nil
	}
	return m, nil
}

func (m transferModel) View() string {
	var b strings.Builder

	// Width 64 ensures progress bar + percent stay on single line
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(uiHighlight).
		Padding(1, 2).
		Width(64)

	b.WriteString(lipgloss.NewStyle().Foreground(uiSpecial).Render("Secure Connection Established"))
	b.WriteString("\n\n")
	if m.code != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(uiMuted).Render("Code: "))
		b.WriteString(lipgloss.NewStyle().Foreground(uiHighlight).Bold(true).Render(m.code))
		b.WriteString(lipgloss.NewStyle().Foreground(uiMuted).Render(" (share with peer)"))
		b.WriteString("\n\n")
	}
	b.WriteString(lipgloss.NewStyle().Foreground(uiHighlight).Bold(true).Render(m.title))
	b.WriteString("\n\n")
	if m.total > 0 {
		pct := float64(m.current) / float64(m.total)
		b.WriteString(m.progress.ViewAs(pct))
	} else {
		b.WriteString(m.progress.ViewAs(0))
	}
	if m.total > 0 {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(uiMuted).Render(fmt.Sprintf("%d / %d bytes", m.current, m.total)))
	}
	if m.err != nil {
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#F25D94")).Render("Error: "+m.err.Error()))
	}
	if m.doneResult != nil && m.err == nil {
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(uiSpecial).Render("Transfer Complete"))
		if m.doneResult.FilePath != "" {
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(uiMuted).Render("Received 1 file:"))
			b.WriteString("\n  ")
			b.WriteString(lipgloss.NewStyle().Foreground(uiHighlight).Render("• "+m.doneResult.FilePath))
		}
		if m.doneResult.Text != "" {
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(uiMuted).Render("Received text:"))
			b.WriteString("\n")
			text := m.doneResult.Text
			if len(text) > 180 {
				text = text[:177] + "..."
			}
			text = strings.ReplaceAll(text, "\n", " ")
			b.WriteString(lipgloss.NewStyle().Foreground(uiHighlight).Render("  " + text))
		}
	}
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(uiMuted).Render("Press q or Esc to exit"))

	return box.Render(b.String())
}

func waitForTransferMsg(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

// RunTransferUI runs a transfer with Bubble Tea + Bubbles progress bar.
// code is displayed in the UI while waiting (empty = hide). fn receives an onProgress callback.
// result: if non-nil, fn should fill it (e.g. Receive) and it will be shown in the UI box when done.
func RunTransferUI(title string, total int64, code string, result *ReceiveResult, fn func(onProgress func(int64, int64)) error) error {
	ch := make(chan tea.Msg, 64)

	go func() {
		err := fn(func(cur, tot int64) {
			select {
			case ch <- ProgressMsg{cur, tot}:
			default:
			}
		})
		ch <- DoneMsg{Err: err, Result: result}
	}()

	prog := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(46), // fits in box without wrap (bar ~41 + " 100%" ~5)
	)

	m := transferModel{
		progress: prog,
		total:    total,
		title:    title,
		code:     code,
		ch:       ch,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return err
	}
	if fm, ok := final.(transferModel); ok && fm.err != nil {
		return fm.err
	}
	return nil
}

// RenderSecureBox renders a "Secure Connection Established" box (for non-TUI use).
func RenderSecureBox() string {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(uiHighlight).
		Padding(1, 2).
		Width(60)
	return box.Render(lipgloss.NewStyle().Foreground(uiSpecial).Render("Secure Connection Established"))
}
