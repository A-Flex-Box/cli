package platform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Linux struct{}

func (l *Linux) Name() string {
	if distro := l.getDistro(); distro != "" {
		return "Linux (" + distro + ")"
	}
	return "Linux"
}

func (l *Linux) ServiceManager() string {
	if l.hasSystemd() {
		return "systemd"
	}
	if l.hasOpenRC() {
		return "openrc"
	}
	return "none"
}

func (l *Linux) InstallService(name, executable string) error {
	switch l.ServiceManager() {
	case "systemd":
		return l.installSystemdService(name, executable)
	case "openrc":
		return l.installOpenRCService(name, executable)
	default:
		return fmt.Errorf("no supported service manager found")
	}
}

func (l *Linux) UninstallService(name string) error {
	switch l.ServiceManager() {
	case "systemd":
		return l.uninstallSystemdService(name)
	case "openrc":
		return l.uninstallOpenRCService(name)
	default:
		return fmt.Errorf("no supported service manager found")
	}
}

func (l *Linux) StartService(ctx context.Context, name string) error {
	switch l.ServiceManager() {
	case "systemd":
		return exec.CommandContext(ctx, "systemctl", "--user", "start", name+".service").Run()
	case "openrc":
		return exec.CommandContext(ctx, "sudo", "rc-service", name, "start").Run()
	default:
		return fmt.Errorf("no supported service manager found")
	}
}

func (l *Linux) StopService(ctx context.Context, name string) error {
	switch l.ServiceManager() {
	case "systemd":
		return exec.CommandContext(ctx, "systemctl", "--user", "stop", name+".service").Run()
	case "openrc":
		return exec.CommandContext(ctx, "sudo", "rc-service", name, "stop").Run()
	default:
		return fmt.Errorf("no supported service manager found")
	}
}

func (l *Linux) ServiceStatus(name string) (string, error) {
	switch l.ServiceManager() {
	case "systemd":
		output, err := exec.Command("systemctl", "--user", "is-active", name+".service").Output()
		if err != nil {
			return "stopped", nil
		}
		status := strings.TrimSpace(string(output))
		if status == "active" {
			return "running", nil
		}
		return "stopped", nil
	case "openrc":
		output, err := exec.Command("sudo", "rc-service", name, "status").Output()
		if err != nil {
			return "stopped", nil
		}
		if strings.Contains(string(output), "started") {
			return "running", nil
		}
		return "stopped", nil
	default:
		return "unknown", nil
	}
}

func (l *Linux) OpenURL(url string) error {
	return exec.Command("xdg-open", url).Start()
}

func (l *Linux) hasSystemd() bool {
	_, err := exec.LookPath("systemctl")
	return err == nil
}

func (l *Linux) hasOpenRC() bool {
	_, err := exec.LookPath("rc-service")
	return err == nil
}

func (l *Linux) getDistro() string {
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
			}
		}
	}
	return ""
}

func (l *Linux) installSystemdService(name, executable string) error {
	serviceContent := fmt.Sprintf(`[Unit]
Description=%s
After=network.target

[Service]
Type=simple
ExecStart=%s
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
`, name, executable)

	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	serviceDir := filepath.Join(configDir, "systemd", "user")
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		return err
	}

	servicePath := filepath.Join(serviceDir, name+".service")
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return err
	}

	exec.Command("systemctl", "--user", "daemon-reload").Run()
	return nil
}

func (l *Linux) uninstallSystemdService(name string) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	servicePath := filepath.Join(configDir, "systemd", "user", name+".service")
	if err := os.Remove(servicePath); err != nil && !os.IsNotExist(err) {
		return err
	}

	exec.Command("systemctl", "--user", "daemon-reload").Run()
	return nil
}

func (l *Linux) installOpenRCService(name, executable string) error {
	return fmt.Errorf("OpenRC service installation not implemented")
}

func (l *Linux) uninstallOpenRCService(name string) error {
	return fmt.Errorf("OpenRC service uninstallation not implemented")
}
