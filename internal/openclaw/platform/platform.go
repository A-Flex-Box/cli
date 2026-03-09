package platform

import (
	"context"
	"os/exec"
	"runtime"
)

type Platform interface {
	Name() string
	ServiceManager() string
	InstallService(name, executable string) error
	UninstallService(name string) error
	StartService(ctx context.Context, name string) error
	StopService(ctx context.Context, name string) error
	ServiceStatus(name string) (string, error)
	OpenURL(url string) error
}

func Detect() Platform {
	switch runtime.GOOS {
	case "linux":
		return &Linux{}
	case "darwin":
		return &Darwin{}
	case "windows":
		return &Windows{}
	default:
		return &Unknown{}
	}
}

type Unknown struct{}

func (u *Unknown) Name() string                     { return runtime.GOOS }
func (u *Unknown) ServiceManager() string           { return "none" }
func (u *Unknown) InstallService(_, _ string) error { return nil }
func (u *Unknown) UninstallService(_ string) error  { return nil }
func (u *Unknown) StartService(_ context.Context, _ string) error {
	return nil
}
func (u *Unknown) StopService(_ context.Context, _ string) error {
	return nil
}
func (u *Unknown) ServiceStatus(_ string) (string, error) {
	return "unknown", nil
}
func (u *Unknown) OpenURL(url string) error {
	return exec.Command("xdg-open", url).Start()
}
