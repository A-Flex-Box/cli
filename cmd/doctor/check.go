package doctor

import (
	"fmt"
	"net"
	"strings"

	"github.com/A-Flex-Box/cli/internal/config"
	"github.com/A-Flex-Box/cli/internal/doctor"
	"github.com/A-Flex-Box/cli/internal/logger"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func newCheckCmd(cfg *config.Root) *cobra.Command {
	return &cobra.Command{
		Use:     "check",
		Short:   "Quick connectivity and environment health check",
		Long:    `Check internet (DNS), Relay Server latency, and development tools/services.`,
		Example: "cli doctor check",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheck(cfg)
		},
	}
}

func runCheck(cfg *config.Root) error {
	logger.Info("doctor check started",
		zap.String("component", "cmd.doctor.check"))

	// 1. Connectivity checks
	items := []checkItem{}

	// DNS
	dnsOk := doctor.CheckDNSResolve("google.com") == nil
	logger.Debug("doctor check DNS result",
		zap.String("component", "cmd.doctor.check"),
		zap.Bool("ok", dnsOk))
	items = append(items, checkItem{
		Name:   "Internet (DNS)",
		Status: dnsOk,
		Detail: "google.com",
	})

	// Relay
	relayAddr := "tcp://relay.flex-box.dev:9000"
	if cfg != nil && cfg.Wormhole.Relays != nil {
		if a := cfg.Wormhole.GetActiveRelayAddr(); a != "" {
			relayAddr = a
		}
	}
	host, port := parseRelayAddr(relayAddr)
	relayTarget := net.JoinHostPort(host, port)
	latency := doctor.MeasureLatency(relayTarget)
	relayOk := latency > 0
	logger.Debug("doctor check Relay result",
		zap.String("component", "cmd.doctor.check"),
		zap.String("target", relayTarget),
		zap.Bool("ok", relayOk),
		zap.Duration("latency", latency))
	detail := "unreachable"
	if relayOk {
		detail = latency.String()
	}
	items = append(items, checkItem{
		Name:   "Relay Server",
		Status: relayOk,
		Detail: detail,
	})

	// 2. Tools & Services (existing registry)
	r := doctor.DefaultRegistry.Run()
	logger.Info("doctor check tools/services completed",
		zap.String("component", "cmd.doctor.check"),
		zap.Int("tools", len(r.Tools)),
		zap.Int("services", len(r.Svc)))

	// 3. Output with Lip Gloss
	printCheckOutput(items, r)
	return nil
}

type checkItem struct {
	Name   string
	Status bool
	Detail string
}

func parseRelayAddr(addr string) (host, port string) {
	addr = strings.TrimSpace(addr)
	addr = strings.TrimPrefix(addr, "tcp://")
	addr = strings.TrimPrefix(addr, "tls://")
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		if strings.Contains(addr, ":") {
			parts := strings.SplitN(addr, ":", 2)
			return parts[0], parts[1]
		}
		return addr, "9000"
	}
	if port == "" {
		port = "9000"
	}
	return host, port
}

var (
	checkOkStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#43BF6D")).Bold(true)
	checkFailStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F25D94")).Bold(true)
	checkDetail    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B6B6B"))
)

func printCheckOutput(items []checkItem, r *doctor.Report) {
	lines := []string{}
	lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true).Render("Connectivity"))
	for _, it := range items {
		st := checkFailStyle.Render("✗")
		if it.Status {
			st = checkOkStyle.Render("✓")
		}
		lines = append(lines, fmt.Sprintf("  %s %s  %s", st, it.Name, checkDetail.Render(it.Detail)))
	}
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true).Render("Tools & Services"))
	lines = append(lines, doctor.RenderCheckReport(r))
	fmt.Println(lipgloss.JoinVertical(lipgloss.Left, lines...))
}
