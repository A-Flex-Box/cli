package service

import (
	"context"
	"os"
	"os/exec"
	"strings"
)

type ServiceStatus string

const (
	StatusRunning ServiceStatus = "running"
	StatusStopped ServiceStatus = "stopped"
	StatusUnknown ServiceStatus = "unknown"
)

type ServiceInfo struct {
	Status  ServiceStatus
	PID     int
	Port    int
	Uptime  string
	Memory  string
	Version string
}

type Manager struct {
	serviceType string
}

func NewManager(serviceType string) *Manager {
	return &Manager{serviceType: serviceType}
}

func (m *Manager) Start(ctx context.Context) error {
	homeDir, _ := os.UserHomeDir()
	switch m.serviceType {
	case "systemd":
		return exec.CommandContext(ctx, "systemctl", "--user", "start", "openclaw-gateway.service").Run()
	case "launchd":
		plist := homeDir + "/Library/LaunchAgents/ai.openclaw.gateway.plist"
		return exec.CommandContext(ctx, "launchctl", "load", plist).Run()
	case "docker":
		return exec.CommandContext(ctx, "docker", "start", "openclaw").Run()
	default:
		return exec.CommandContext(ctx, "openclaw", "gateway", "--port", "18789").Start()
	}
}

func (m *Manager) Stop(ctx context.Context) error {
	switch m.serviceType {
	case "systemd":
		return exec.CommandContext(ctx, "systemctl", "--user", "stop", "openclaw-gateway.service").Run()
	case "launchd":
		homeDir, _ := os.UserHomeDir()
		plist := homeDir + "/Library/LaunchAgents/ai.openclaw.gateway.plist"
		return exec.CommandContext(ctx, "launchctl", "unload", plist).Run()
	case "docker":
		return exec.CommandContext(ctx, "docker", "stop", "openclaw").Run()
	default:
		return nil
	}
}

func (m *Manager) Restart(ctx context.Context) error {
	if err := m.Stop(ctx); err != nil {
		return err
	}
	return m.Start(ctx)
}

func (m *Manager) Status() ServiceInfo {
	info := ServiceInfo{Status: StatusUnknown}

	switch m.serviceType {
	case "systemd":
		output, err := exec.Command("systemctl", "--user", "is-active", "openclaw-gateway.service").Output()
		if err == nil {
			if strings.Contains(string(output), "active") {
				info.Status = StatusRunning
			} else {
				info.Status = StatusStopped
			}
		}

	case "launchd":
		output, err := exec.Command("launchctl", "list", "ai.openclaw.gateway").Output()
		if err == nil && len(output) > 0 {
			info.Status = StatusRunning
		} else {
			info.Status = StatusStopped
		}

	case "docker":
		output, err := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", "openclaw").Output()
		if err == nil {
			if strings.Contains(string(output), "true") {
				info.Status = StatusRunning
			} else {
				info.Status = StatusStopped
			}
		}

	default:
		if _, err := exec.LookPath("openclaw"); err == nil {
			output, err := exec.Command("pgrep", "-f", "openclaw gateway").Output()
			if err == nil && len(output) > 0 {
				info.Status = StatusRunning
			} else {
				info.Status = StatusStopped
			}
		}
	}

	info.Port = 18789
	return info
}

func (m *Manager) GetLogs(lines int) ([]string, error) {
	var output []byte
	var err error

	switch m.serviceType {
	case "systemd":
		output, err = exec.Command("journalctl", "--user", "-u", "openclaw-gateway.service", "-n", string(rune(lines+'0')), "--no-pager").Output()
	case "docker":
		output, err = exec.Command("docker", "logs", "--tail", string(rune(lines+'0')), "openclaw").Output()
	default:
		homeDir, _ := os.UserHomeDir()
		logPath := homeDir + "/.openclaw/logs/gateway.log"
		output, err = os.ReadFile(logPath)
	}

	if err != nil {
		return nil, err
	}
	return strings.Split(string(output), "\n"), nil
}
