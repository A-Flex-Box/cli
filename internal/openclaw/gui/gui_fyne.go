//go:build fyne
// +build fyne

package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/A-Flex-Box/cli/internal/openclaw/installer"
)

func RunMainWindow(app *App) error {
	a := app.NewWithID("ai.openclaw.installer")
	w := a.NewWindow("OpenClaw Installer")

	tabs := container.NewAppTabs(
		container.NewTabItem("Install", newInstallTab(app)),
		container.NewTabItem("Uninstall", newUninstallTab(app)),
		container.NewTabItem("Config", newConfigTab(app)),
		container.NewTabItem("Plugins", newPluginsTab(app)),
		container.NewTabItem("About", newAboutTab(app)),
	)

	w.SetContent(tabs)
	w.Resize(fyne.NewSize(800, 600))
	w.ShowAndRun()

	return nil
}

func newInstallTab(app *App) *container.TabItem {
	methodSelect := widget.NewSelect([]string{"native", "docker", "source"}, nil)
	methodSelect.SetSelected("native")

	versionSelect := widget.NewSelect([]string{"stable", "beta", "dev"}, nil)
	versionSelect.SetSelected("stable")

	progress := widget.NewProgressBar()
	status := widget.NewLabel("Ready to install")

	installBtn := widget.NewButton("Install", func() {
		status.SetText("Installing...")
		progress.SetValue(0)

		opts := &installer.InstallOptions{
			Method:  installer.InstallMethod(methodSelect.Selected),
			Version: installer.InstallVersion(versionSelect.Selected),
		}

		cb := func(stage string, current, total int, message string) {
			progress.SetValue(float64(current) / float64(total))
			status.SetText(fmt.Sprintf("[%s] %s", stage, message))
		}

		if err := app.Install(opts, cb); err != nil {
			status.SetText(fmt.Sprintf("Error: %v", err))
		} else {
			progress.SetValue(1)
			status.SetText("Installation complete!")
		}
	})

	content := container.NewVBox(
		widget.NewLabel("Installation Method:"),
		methodSelect,
		widget.NewLabel("Version:"),
		versionSelect,
		widget.NewSeparator(),
		installBtn,
		progress,
		status,
	)

	return container.NewTabItem("Install", content)
}

func newUninstallTab(app *App) *container.TabItem {
	purgeCheck := widget.NewCheck("Remove all configuration and data", nil)

	status := widget.NewLabel("Ready to uninstall")

	uninstallBtn := widget.NewButton("Uninstall", func() {
		opts := &installer.UninstallOptions{
			Purge: purgeCheck.Checked,
		}

		cb := func(stage string, current, total int, message string) {
			status.SetText(fmt.Sprintf("[%s] %s", stage, message))
		}

		if err := app.Uninstall(opts, cb); err != nil {
			status.SetText(fmt.Sprintf("Error: %v", err))
		} else {
			status.SetText("OpenClaw has been uninstalled.")
		}
	})

	content := container.NewVBox(
		widget.NewLabel("Uninstall OpenClaw"),
		purgeCheck,
		widget.NewSeparator(),
		uninstallBtn,
		status,
	)

	return container.NewTabItem("Uninstall", content)
}

func newConfigTab(app *App) *container.TabItem {
	content := container.NewVBox(
		widget.NewLabel("Configuration settings will be displayed here"),
	)
	return container.NewTabItem("Config", content)
}

func newPluginsTab(app *App) *container.TabItem {
	plugins := app.PluginManager.List()

	var items []string
	for _, p := range plugins {
		items = append(items, fmt.Sprintf("%s - %s", p.Name, p.Description))
	}

	list := widget.NewList(
		func() int { return len(items) },
		func() fyne.CanvasObject {
			return widget.NewLabel("Plugin Item")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(items[id])
		},
	)

	return container.NewTabItem("Plugins", list)
}

func newAboutTab(app *App) *container.TabItem {
	version := "Not installed"
	if v, err := app.GetVersion(); err == nil {
		version = v
	}

	content := container.NewVBox(
		widget.NewLabelWithStyle("OpenClaw Installer", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewLabel(fmt.Sprintf("Version: %s", version)),
		widget.NewLabel(fmt.Sprintf("Platform: %s/%s", app.SysInfo.OS, app.SysInfo.Arch)),
		widget.NewSeparator(),
		widget.NewLabel("OpenClaw - AI Agent Framework"),
		widget.NewLabel("https://openclaw.ai"),
	)

	return container.NewTabItem("About", content)
}
