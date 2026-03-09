package platform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Windows struct{}

func (w *Windows) Name() string {
	if ver := w.getVersion(); ver != "" {
		return "Windows " + ver
	}
	return "Windows"
}

func (w *Windows) ServiceManager() string {
	if w.isWSL() {
		return "wsl-systemd"
	}
	return "none"
}

func (w *Windows) InstallService(name, executable string) error {
	if w.isWSL() {
		return fmt.Errorf("use systemd within WSL for service management")
	}
	return fmt.Errorf("Windows native service installation not supported, use WSL")
}

func (w *Windows) UninstallService(name string) error {
	if w.isWSL() {
		return fmt.Errorf("use systemd within WSL for service management")
	}
	return fmt.Errorf("Windows native service uninstallation not supported, use WSL")
}

func (w *Windows) StartService(ctx context.Context, name string) error {
	if w.isWSL() {
		return exec.CommandContext(ctx, "wsl", "systemctl", "--user", "start", name).Run()
	}
	return fmt.Errorf("use WSL for service management")
}

func (w *Windows) StopService(ctx context.Context, name string) error {
	if w.isWSL() {
		return exec.CommandContext(ctx, "wsl", "systemctl", "--user", "stop", name).Run()
	}
	return fmt.Errorf("use WSL for service management")
}

func (w *Windows) ServiceStatus(name string) (string, error) {
	if w.isWSL() {
		output, err := exec.Command("wsl", "systemctl", "--user", "is-active", name).Output()
		if err != nil {
			return "stopped", nil
		}
		if string(output) == "active\n" {
			return "running", nil
		}
		return "stopped", nil
	}
	return "unknown", nil
}

func (w *Windows) OpenURL(url string) error {
	return exec.Command("cmd", "/c", "start", url).Start()
}

func (w *Windows) isWSL() bool {
	if _, err := exec.LookPath("wsl.exe"); err != nil {
		return false
	}
	if _, err := os.Stat("/proc/sys/fs/binfmt_misc/WSLInterop"); err == nil {
		return true
	}
	return false
}

func (w *Windows) getVersion() string {
	return ""
}

func (w *Windows) GetWSLDistributions() ([]string, error) {
	output, err := exec.Command("wsl", "-l", "-q").Output()
	if err != nil {
		return nil, err
	}

	var distros []string
	lines := filepath.SplitList(string(output))
	for _, line := range lines {
		line = line
		if line != "" {
			distros = append(distros, line)
		}
	}
	return distros, nil
}

func (w *Windows) RunInWSL(distro, command string, args ...string) error {
	cmdArgs := []string{"-d", distro, "--", command}
	cmdArgs = append(cmdArgs, args...)
	return exec.Command("wsl", cmdArgs...).Run()
}
