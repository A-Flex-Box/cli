package platform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Darwin struct{}

func (d *Darwin) Name() string {
	if ver := d.getVersion(); ver != "" {
		return "macOS " + ver
	}
	return "macOS"
}

func (d *Darwin) ServiceManager() string {
	return "launchd"
}

func (d *Darwin) InstallService(name, executable string) error {
	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>ai.openclaw.%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/%s.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/%s.error.log</string>
</dict>
</plist>
`, name, executable, name, name)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	launchAgentsDir := filepath.Join(homeDir, "Library", "LaunchAgents")
	if err := os.MkdirAll(launchAgentsDir, 0755); err != nil {
		return err
	}

	plistPath := filepath.Join(launchAgentsDir, "ai.openclaw."+name+".plist")
	return os.WriteFile(plistPath, []byte(plistContent), 0644)
}

func (d *Darwin) UninstallService(name string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	plistPath := filepath.Join(homeDir, "Library", "LaunchAgents", "ai.openclaw."+name+".plist")
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	exec.Command("launchctl", "unload", plistPath).Run()
	return nil
}

func (d *Darwin) StartService(ctx context.Context, name string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	plistPath := filepath.Join(homeDir, "Library", "LaunchAgents", "ai.openclaw."+name+".plist")
	return exec.CommandContext(ctx, "launchctl", "load", plistPath).Run()
}

func (d *Darwin) StopService(ctx context.Context, name string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	plistPath := filepath.Join(homeDir, "Library", "LaunchAgents", "ai.openclaw."+name+".plist")
	return exec.CommandContext(ctx, "launchctl", "unload", plistPath).Run()
}

func (d *Darwin) ServiceStatus(name string) (string, error) {
	output, err := exec.Command("launchctl", "list", "ai.openclaw."+name).Output()
	if err != nil {
		return "stopped", nil
	}
	if len(output) > 0 {
		return "running", nil
	}
	return "stopped", nil
}

func (d *Darwin) OpenURL(url string) error {
	return exec.Command("open", url).Start()
}

func (d *Darwin) getVersion() string {
	output, err := exec.Command("sw_vers", "-productVersion").Output()
	if err != nil {
		return ""
	}
	return string(output)
}
