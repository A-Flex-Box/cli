//go:build fyne
// +build fyne

package screens

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/A-Flex-Box/cli/internal/openclaw/installer"
)

type InstallScreen struct {
	app       *InstallApp
	container *fyne.Container
}

type InstallApp struct {
	SysInfo   *installer.SystemInfo
	Installer installer.Installer
}

func NewInstallApp() *InstallApp {
	sysInfo := installer.DetectSystem()
	return &InstallApp{
		SysInfo:   sysInfo,
		Installer: installer.NewInstaller(installer.MethodNative, sysInfo),
	}
}

func NewInstallScreen(app *InstallApp) *InstallScreen {
	screen := &InstallScreen{app: app}
	screen.container = screen.buildUI()
	return screen
}

func (s *InstallScreen) buildUI() *fyne.Container {
	methodSelect := widget.NewSelect([]string{"native", "docker", "source"}, nil)
	methodSelect.SetSelected("native")

	versionSelect := widget.NewSelect([]string{"stable", "beta", "dev"}, nil)
	versionSelect.SetSelected("stable")

	detectedInfo := widget.NewLabel(fmt.Sprintf("Detected: %s (%s)", s.app.SysInfo.Distro, s.app.SysInfo.Version))

	nodeStatus := "✗ Not installed"
	if s.app.SysInfo.NodeVersion != "" {
		nodeStatus = "✓ " + s.app.SysInfo.NodeVersion
	}
	npmStatus := "✗ Not installed"
	if s.app.SysInfo.NpmVersion != "" {
		npmStatus = "✓ " + s.app.SysInfo.NpmVersion
	}
	dockerStatus := "✗ Not installed"
	if s.app.SysInfo.DockerVer != "" {
		dockerStatus = "✓ " + s.app.SysInfo.DockerVer
	}

	depsContainer := container.NewVBox(
		widget.NewLabelWithStyle("System Dependencies", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel(fmt.Sprintf("Node.js: %s", nodeStatus)),
		widget.NewLabel(fmt.Sprintf("npm: %s", npmStatus)),
		widget.NewLabel(fmt.Sprintf("Docker: %s", dockerStatus)),
	)

	progress := widget.NewProgressBar()
	status := widget.NewLabel("Ready to install")
	progress.Hide()

	componentBtn := widget.NewButton("Select Components...", func() {
		// Will be handled by parent window
	})

	installBtn := widget.NewButton("Install", func() {
		status.SetText("Installing...")
		progress.Show()
		progress.SetValue(0)

		opts := &installer.InstallOptions{
			Method:  installer.InstallMethod(methodSelect.Selected),
			Version: installer.InstallVersion(versionSelect.Selected),
		}

		cb := func(stage string, current, total int, message string) {
			progress.SetValue(float64(current) / float64(total))
			status.SetText(fmt.Sprintf("[%s] %s", stage, message))
		}

		if err := s.app.Installer.Install(nil, opts, cb); err != nil {
			status.SetText(fmt.Sprintf("Error: %v", err))
		} else {
			progress.SetValue(1)
			status.SetText("Installation complete!")
		}
	})

	form := container.NewVBox(
		widget.NewLabelWithStyle("Installation Method", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		methodSelect,
		widget.NewLabelWithStyle("Version", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		versionSelect,
		widget.NewSeparator(),
		detectedInfo,
		widget.NewSeparator(),
		depsContainer,
		widget.NewSeparator(),
		componentBtn,
		widget.NewSeparator(),
		installBtn,
		progress,
		status,
	)

	return container.NewPadded(form)
}

func (s *InstallScreen) Container() *fyne.Container {
	return s.container
}
