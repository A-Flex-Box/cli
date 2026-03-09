package gui

import (
	"context"

	"github.com/A-Flex-Box/cli/internal/openclaw/config"
	"github.com/A-Flex-Box/cli/internal/openclaw/installer"
	"github.com/A-Flex-Box/cli/internal/openclaw/plugin"
)

type App struct {
	SysInfo       *installer.SystemInfo
	Config        *config.OpenClawConfig
	PluginManager *plugin.Manager
	Installer     installer.Installer
}

func NewApp() *App {
	sysInfo := installer.DetectSystem()
	return &App{
		SysInfo:       sysInfo,
		Config:        config.DefaultConfig(),
		PluginManager: plugin.NewManager(),
		Installer:     installer.NewInstaller(installer.MethodNative, sysInfo),
	}
}

func (a *App) Run() error {
	return RunMainWindow(a)
}

func (a *App) Install(opts *installer.InstallOptions, progress installer.ProgressCallback) error {
	inst := installer.NewInstaller(opts.Method, a.SysInfo)
	return inst.Install(context.Background(), opts, progress)
}

func (a *App) Uninstall(opts *installer.UninstallOptions, progress installer.ProgressCallback) error {
	inst := installer.NewInstaller(installer.MethodNative, a.SysInfo)
	return inst.Uninstall(context.Background(), opts, progress)
}

func (a *App) IsInstalled() bool {
	return a.Installer.IsInstalled()
}

func (a *App) GetVersion() (string, error) {
	return a.Installer.GetVersion()
}
