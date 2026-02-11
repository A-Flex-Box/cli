package doctor

import (
	"fmt"
	"strconv"

	"github.com/A-Flex-Box/cli/internal/doctor"
	"github.com/A-Flex-Box/cli/internal/doctor/instrument"
	"github.com/A-Flex-Box/cli/internal/logger"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func newPortCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "port [port]",
		Short:   "Deep analysis of a local port (Process + Protocol)",
		Long:    `Inspect who is using the port, connect to it, grab banner and detect protocol.`,
		Example: "cli doctor port 8080",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			port, err := strconv.Atoi(args[0])
			if err != nil || port <= 0 || port > 65535 {
				return fmt.Errorf("invalid port: %s (must be 1-65535)", args[0])
			}
			return runPort(port)
		},
	}
}

func runPort(port int) error {
	logger.Info("doctor port started",
		zap.String("component", "cmd.doctor.port"),
		zap.Int("port", port))

	target := fmt.Sprintf("127.0.0.1:%d", port)

	// 1. Port occupancy
	info, err := doctor.GetPortOccupancy(port)
	if err != nil {
		logger.Error("doctor port GetPortOccupancy failed",
			zap.String("component", "cmd.doctor.port"),
			zap.Int("port", port),
			zap.Error(err))
		return err
	}

	// 2. Connect and probe
	banner := doctor.GrabBanner(target)
	protocol := instrument.ProtocolTCP
	if len(banner) > 0 {
		protocol = instrument.DetectProtocol([]byte(banner))
	}
	logger.Debug("doctor port probe result",
		zap.String("component", "cmd.doctor.port"),
		zap.Int("port", port),
		zap.String("protocol", protocol),
		zap.String("banner", instrument.TruncateForDisplay([]byte(banner), 80)))

	// 3. Output
	lines := []string{}
	lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true).Render("Port Inspector"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  Port:     %d", port))

	if info != nil {
		if info.ProcessName != "" {
			lines = append(lines, fmt.Sprintf("  Process:  %s (PID: %d)", info.ProcessName, info.PID))
			if info.CommandLine != "" {
				cmdShort := info.CommandLine
				if len(cmdShort) > 60 {
					cmdShort = cmdShort[:57] + "..."
				}
				lines = append(lines, fmt.Sprintf("  Command:  %s", cmdShort))
			}
			if info.Permission != "" {
				lines = append(lines, fmt.Sprintf("  Note:     %s", info.Permission))
			}
		} else {
			lines = append(lines, "  Process:  (unknown)")
		}
	} else {
		lines = append(lines, "  Process:  (port not in use or not detected)")
	}

	lines = append(lines, fmt.Sprintf("  Protocol: %s", protocol))
	if banner != "" {
		bannerShort := instrument.TruncateForDisplay([]byte(banner), 80)
		lines = append(lines, fmt.Sprintf("  Banner:   %s", bannerShort))
	} else {
		lines = append(lines, "  Banner:   (none)")
	}

	box := lipgloss.NewStyle().
		Margin(1, 2).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4"))

	fmt.Println(box.Render(lipgloss.JoinVertical(lipgloss.Left, lines...)))

	logger.Info("doctor port completed",
		zap.String("component", "cmd.doctor.port"),
		zap.Int("port", port))
	return nil
}
