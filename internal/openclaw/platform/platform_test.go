package platform

import (
	"testing"
)

func TestDetect(t *testing.T) {
	p := Detect()

	if p == nil {
		t.Fatal("Detect() returned nil")
	}

	if p.Name() == "" {
		t.Error("platform name should not be empty")
	}
}

func TestUnknownPlatform(t *testing.T) {
	u := &Unknown{}

	if u.Name() == "" {
		t.Error("Unknown.Name() returned empty string")
	}

	if u.ServiceManager() != "none" {
		t.Errorf("Unknown.ServiceManager() = %s, want none", u.ServiceManager())
	}

	status, err := u.ServiceStatus("test")
	if err != nil {
		t.Errorf("Unknown.ServiceStatus() error = %v", err)
	}
	if status != "unknown" {
		t.Errorf("Unknown.ServiceStatus() = %s, want unknown", status)
	}
}

func TestLinuxPlatformMethods(t *testing.T) {
	l := &Linux{}

	if l.Name() == "" {
		t.Error("Linux.Name() returned empty string")
	}

	sm := l.ServiceManager()
	if sm != "systemd" && sm != "openrc" && sm != "none" {
		t.Errorf("Linux.ServiceManager() = %s, unexpected value", sm)
	}
}

func TestDarwinPlatformMethods(t *testing.T) {
	d := &Darwin{}

	if d.Name() == "" {
		t.Error("Darwin.Name() returned empty string")
	}

	if d.ServiceManager() != "launchd" {
		t.Errorf("Darwin.ServiceManager() = %s, want launchd", d.ServiceManager())
	}
}

func TestWindowsPlatformMethods(t *testing.T) {
	w := &Windows{}

	if w.Name() == "" {
		t.Error("Windows.Name() returned empty string")
	}
}

func TestPlatformInterface(t *testing.T) {
	var _ Platform = (*Linux)(nil)
	var _ Platform = (*Darwin)(nil)
	var _ Platform = (*Windows)(nil)
	var _ Platform = (*Unknown)(nil)
}
