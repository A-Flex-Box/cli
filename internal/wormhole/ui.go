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
	Err error
}

// transferModel holds the Bubble Tea model for a transfer.
type transferModel struct {
	progress progress.Model
	current  int64
	total    int64
	title    string
	ch       <-chan tea.Msg
	err      error
}

func (m transferModel) Init() tea.Cmd {
	return waitForTransferMsg(m.ch)
}

func (m transferModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		done := m.err != nil || (m.total > 0 && m.current >= m.total)
		if done {
			return m, tea.Quit
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - 4
		if m.progress.Width > 80 {
			m.progress.Width = 80
		}
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
		m.current = m.total
		if m.total == 0 {
			m.total = 1
		}
		return m, tea.Quit
	}
	return m, nil
}

func (m transferModel) View() string {
	var b strings.Builder

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(uiHighlight).
		Padding(1, 2).
		Width(60)

	b.WriteString(lipgloss.NewStyle().Foreground(uiSpecial).Render("Secure Connection Established"))
	b.WriteString("\n\n")
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
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(uiMuted).Render("Press any key to exit"))

	return box.Render(b.String())
}

func waitForTransferMsg(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

// RunTransferUI runs a transfer with Bubble Tea + Bubbles progress bar.
// fn receives an onProgress callback; call it with (current, total) during transfer.
func RunTransferUI(title string, total int64, fn func(onProgress func(int64, int64)) error) error {
	ch := make(chan tea.Msg, 64)

	go func() {
		err := fn(func(cur, tot int64) {
			select {
			case ch <- ProgressMsg{cur, tot}:
			default:
			}
		})
		ch <- DoneMsg{Err: err}
	}()

	prog := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(60),
	)

	m := transferModel{
		progress: prog,
		total:    total,
		title:    title,
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
