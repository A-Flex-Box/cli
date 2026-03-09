package installer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type Installer interface {
	Install(ctx context.Context, opts *InstallOptions, progress ProgressCallback) error
	Uninstall(ctx context.Context, opts *UninstallOptions, progress ProgressCallback) error
	IsInstalled() bool
	GetVersion() (string, error)
}

type ProgressCallback func(stage string, current, total int, message string)

func NewInstaller(method InstallMethod, sysInfo *SystemInfo) Installer {
	switch method {
	case MethodDocker:
		return &DockerInstaller{sysInfo: sysInfo}
	case MethodSource:
		return &SourceInstaller{sysInfo: sysInfo}
	default:
		return &NativeInstaller{sysInfo: sysInfo}
	}
}

type NativeInstaller struct {
	sysInfo     *SystemInfo
	installPath string
}

func (n *NativeInstaller) Install(ctx context.Context, opts *InstallOptions, progress ProgressCallback) error {
	steps := 6

	progress("install", 1, steps, "Checking prerequisites...")
	if err := n.checkPrerequisites(); err != nil {
		return err
	}

	progress("install", 2, steps, "Installing OpenClaw via npm...")
	cmd := exec.CommandContext(ctx, "npm", "install", "-g", "openclaw@latest")
	if opts.Version == "beta" {
		cmd = exec.CommandContext(ctx, "npm", "install", "-g", "openclaw@beta")
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("npm install failed: %w", err)
	}

	progress("install", 3, steps, "Verifying installation...")
	version, err := n.GetVersion()
	if err != nil {
		return fmt.Errorf("installation verification failed: %w", err)
	}

	progress("install", 4, steps, "Creating configuration directory...")
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".openclaw")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if !opts.NoDaemon {
		progress("install", 5, steps, "Installing daemon...")
		if err := n.installDaemon(ctx); err != nil {
			return fmt.Errorf("daemon installation failed: %w", err)
		}
	} else {
		progress("install", 5, steps, "Skipping daemon installation...")
	}

	progress("install", 6, steps, fmt.Sprintf("Installation complete! Version: %s", version))
	return nil
}

func (n *NativeInstaller) Uninstall(ctx context.Context, opts *UninstallOptions, progress ProgressCallback) error {
	steps := 4

	progress("uninstall", 1, steps, "Stopping services...")
	n.stopService(ctx)

	progress("uninstall", 2, steps, "Removing npm package...")
	cmd := exec.CommandContext(ctx, "npm", "uninstall", "-g", "openclaw")
	cmd.Run()

	progress("uninstall", 3, steps, "Removing daemon...")
	n.uninstallDaemon(ctx)

	if opts.Purge {
		progress("uninstall", 4, steps, "Removing configuration and data...")
		homeDir, _ := os.UserHomeDir()
		configDir := filepath.Join(homeDir, ".openclaw")
		os.RemoveAll(configDir)
	} else {
		progress("uninstall", 4, steps, "Keeping configuration and data...")
	}

	return nil
}

func (n *NativeInstaller) IsInstalled() bool {
	_, err := exec.LookPath("openclaw")
	return err == nil
}

func (n *NativeInstaller) GetVersion() (string, error) {
	output, err := exec.Command("openclaw", "--version").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (n *NativeInstaller) checkPrerequisites() error {
	if n.sysInfo.NodeVersion == "" {
		return fmt.Errorf("Node.js is required but not installed")
	}
	if n.sysInfo.NpmVersion == "" {
		return fmt.Errorf("npm is required but not installed")
	}
	return nil
}

func (n *NativeInstaller) installDaemon(ctx context.Context) error {
	homeDir, _ := os.UserHomeDir()
	openclawPath := filepath.Join(homeDir, ".openclaw")

	switch n.sysInfo.GetServiceType() {
	case "systemd":
		serviceDir := filepath.Join(homeDir, ".config", "systemd", "user")
		if err := os.MkdirAll(serviceDir, 0755); err != nil {
			return err
		}
		serviceContent := `[Unit]
Description=OpenClaw Gateway
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=/usr/local/bin/openclaw gateway --port 18789
Restart=always
RestartSec=5

[Install]
WantedBy=default.target
`
		if err := os.WriteFile(filepath.Join(serviceDir, "openclaw-gateway.service"), []byte(serviceContent), 0644); err != nil {
			return err
		}
		exec.Command("systemctl", "--user", "daemon-reload").Run()

	case "launchd":
		launchDir := filepath.Join(homeDir, "Library", "LaunchAgents")
		if err := os.MkdirAll(launchDir, 0755); err != nil {
			return err
		}
		plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>ai.openclaw.gateway</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/openclaw</string>
        <string>gateway</string>
        <string>--port</string>
        <string>18789</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>%s/logs/gateway.log</string>
    <key>StandardErrorPath</key>
    <string>%s/logs/gateway.err</string>
</dict>
</plist>`, openclawPath, openclawPath)
		if err := os.WriteFile(filepath.Join(launchDir, "ai.openclaw.gateway.plist"), []byte(plistContent), 0644); err != nil {
			return err
		}
		exec.Command("launchctl", "load", filepath.Join(launchDir, "ai.openclaw.gateway.plist")).Run()
	}

	return nil
}

func (n *NativeInstaller) uninstallDaemon(ctx context.Context) error {
	homeDir, _ := os.UserHomeDir()

	switch n.sysInfo.GetServiceType() {
	case "systemd":
		serviceDir := filepath.Join(homeDir, ".config", "systemd", "user")
		os.Remove(filepath.Join(serviceDir, "openclaw-gateway.service"))
		exec.Command("systemctl", "--user", "daemon-reload").Run()

	case "launchd":
		launchDir := filepath.Join(homeDir, "Library", "LaunchAgents")
		plistPath := filepath.Join(launchDir, "ai.openclaw.gateway.plist")
		exec.Command("launchctl", "unload", plistPath).Run()
		os.Remove(plistPath)
	}

	return nil
}

func (n *NativeInstaller) stopService(ctx context.Context) {
	switch n.sysInfo.GetServiceType() {
	case "systemd":
		exec.Command("systemctl", "--user", "stop", "openclaw-gateway.service").Run()
	case "launchd":
		exec.Command("launchctl", "bootout", "gui/$(id -u)/ai.openclaw.gateway").Run()
	}
}

type DockerInstaller struct {
	sysInfo *SystemInfo
}

func (d *DockerInstaller) Install(ctx context.Context, opts *InstallOptions, progress ProgressCallback) error {
	steps := 5

	progress("install", 1, steps, "Checking Docker...")
	if d.sysInfo.DockerVer == "" {
		return fmt.Errorf("Docker is not installed")
	}

	progress("install", 2, steps, "Pulling OpenClaw image...")
	image := "ghcr.io/openclaw/openclaw:latest"
	if opts.Version == "beta" {
		image = "ghcr.io/openclaw/openclaw:beta"
	}
	cmd := exec.CommandContext(ctx, "docker", "pull", image)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker pull failed: %w", err)
	}

	progress("install", 3, steps, "Creating OpenClaw directories...")
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".openclaw")
	os.MkdirAll(filepath.Join(configDir, "data"), 0755)
	os.MkdirAll(filepath.Join(configDir, "logs"), 0755)

	progress("install", 4, steps, "Creating Docker compose file...")
	composeContent := fmt.Sprintf(`version: '3.8'
services:
  openclaw:
    image: %s
    container_name: openclaw
    restart: unless-stopped
    ports:
      - "18789:18789"
    volumes:
      - %s:/root/.openclaw
    environment:
      - OPENCLAW_GATEWAY_PORT=18789
`, image, configDir)
	if err := os.WriteFile(filepath.Join(configDir, "docker-compose.yml"), []byte(composeContent), 0644); err != nil {
		return err
	}

	progress("install", 5, steps, "Installation complete! Run: docker-compose -f ~/.openclaw/docker-compose.yml up -d")
	return nil
}

func (d *DockerInstaller) Uninstall(ctx context.Context, opts *UninstallOptions, progress ProgressCallback) error {
	steps := 3

	progress("uninstall", 1, steps, "Stopping container...")
	exec.CommandContext(ctx, "docker", "stop", "openclaw").Run()
	exec.CommandContext(ctx, "docker", "rm", "openclaw").Run()

	progress("uninstall", 2, steps, "Removing image...")
	exec.CommandContext(ctx, "docker", "rmi", "ghcr.io/openclaw/openclaw").Run()

	if opts.Purge {
		progress("uninstall", 3, steps, "Removing data...")
		homeDir, _ := os.UserHomeDir()
		os.RemoveAll(filepath.Join(homeDir, ".openclaw"))
	} else {
		progress("uninstall", 3, steps, "Keeping configuration and data...")
	}

	return nil
}

func (d *DockerInstaller) IsInstalled() bool {
	output, err := exec.Command("docker", "ps", "-a", "--filter", "name=openclaw", "--format", "{{.Names}}").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "openclaw")
}

func (d *DockerInstaller) GetVersion() (string, error) {
	output, err := exec.Command("docker", "exec", "openclaw", "openclaw", "--version").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

type SourceInstaller struct {
	sysInfo *SystemInfo
}

func (s *SourceInstaller) Install(ctx context.Context, opts *InstallOptions, progress ProgressCallback) error {
	steps := 8

	progress("install", 1, steps, "Checking prerequisites...")
	if s.sysInfo.GitVersion == "" {
		return fmt.Errorf("Git is required for source installation")
	}
	if s.sysInfo.NodeVersion == "" {
		return fmt.Errorf("Node.js is required for source installation")
	}

	progress("install", 2, steps, "Cloning repository...")
	srcDir := opts.InstallPath
	if srcDir == "" {
		homeDir, _ := os.UserHomeDir()
		srcDir = filepath.Join(homeDir, "openclaw")
	}

	exec.CommandContext(ctx, "rm", "-rf", srcDir).Run()
	cmd := exec.CommandContext(ctx, "git", "clone", "https://github.com/openclaw/openclaw.git", srcDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	progress("install", 3, steps, "Installing dependencies...")
	cmd = exec.CommandContext(ctx, "pnpm", "install")
	cmd.Dir = srcDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		exec.CommandContext(ctx, "npm", "install", "-g", "pnpm").Run()
		cmd = exec.CommandContext(ctx, "pnpm", "install")
		cmd.Dir = srcDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("pnpm install failed: %w", err)
		}
	}

	progress("install", 4, steps, "Building UI...")
	cmd = exec.CommandContext(ctx, "pnpm", "ui:build")
	cmd.Dir = srcDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("UI build failed: %w", err)
	}

	progress("install", 5, steps, "Building OpenClaw...")
	cmd = exec.CommandContext(ctx, "pnpm", "build")
	cmd.Dir = srcDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	progress("install", 6, steps, "Linking binary...")
	cmd = exec.CommandContext(ctx, "pnpm", "link", "--global")
	cmd.Dir = srcDir
	cmd.Run()

	progress("install", 7, steps, "Running onboarding...")
	cmd = exec.CommandContext(ctx, "pnpm", "openclaw", "onboard", "--install-daemon")
	cmd.Dir = srcDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("onboarding failed: %w", err)
	}

	progress("install", 8, steps, "Installation complete!")
	return nil
}

func (s *SourceInstaller) Uninstall(ctx context.Context, opts *UninstallOptions, progress ProgressCallback) error {
	steps := 3

	progress("uninstall", 1, steps, "Unlinking binary...")
	exec.CommandContext(ctx, "pnpm", "unlink", "-g", "openclaw").Run()

	progress("uninstall", 2, steps, "Removing source...")
	homeDir, _ := os.UserHomeDir()
	os.RemoveAll(filepath.Join(homeDir, "openclaw"))

	if opts.Purge {
		progress("uninstall", 3, steps, "Removing configuration and data...")
		os.RemoveAll(filepath.Join(homeDir, ".openclaw"))
	} else {
		progress("uninstall", 3, steps, "Keeping configuration and data...")
	}

	return nil
}

func (s *SourceInstaller) IsInstalled() bool {
	_, err := exec.LookPath("openclaw")
	return err == nil
}

func (s *SourceInstaller) GetVersion() (string, error) {
	output, err := exec.Command("openclaw", "--version").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func GetDefaultInstallPath() string {
	homeDir, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(homeDir, "AppData", "Local", "openclaw")
	case "darwin":
		return filepath.Join(homeDir, ".openclaw")
	default:
		return filepath.Join(homeDir, ".openclaw")
	}
}
