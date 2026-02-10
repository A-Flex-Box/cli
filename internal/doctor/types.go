package doctor

import "runtime"

// InstallStatus indicates whether a tool/service binary is installed.
type InstallStatus string

const (
	InstallStatusInstalled   InstallStatus = "installed"
	InstallStatusNotInstall InstallStatus = "not install"
)

// ListeningState indicates if a service port is listening.
type ListeningState string

const (
	ListeningYes ListeningState = "yes"
	ListeningNo  ListeningState = "no"
	ListeningNA  ListeningState = "-"
)

// PortStatusType describes default port probe result.
type PortStatusType string

const (
	PortStatusListening    PortStatusType = "listening"
	PortStatusNotListening PortStatusType = "not listening"
	PortStatusNone         PortStatusType = "none"
)

// ToolEntry holds detected tool path and version (empty means not installed).
type ToolEntry struct {
	Name    string
	Path    string
	Version string
	Status  InstallStatus
}

// ServiceEntry holds service detection: CLI tool + optional port listening.
type ServiceEntry struct {
	Name       string
	Path       string
	Version    string
	Status     InstallStatus
	Port       string
	Listening  ListeningState
	PortStatus PortStatusType
}

// Result is the return value of a Checker. Exactly one of Tool or Service is set.
type Result struct {
	Tool    *ToolEntry
	Service *ServiceEntry
}

// Checker is the interface implemented by every tool/service probe.
// Check() may be called concurrently.
type Checker interface {
	// Name returns the display name (e.g. "go", "docker").
	Name() string
	// Category is "tool" or "service" for grouping and ordering.
	Category() string
	// Check runs the detection and returns a single result.
	Check() Result
}

// Report is the full doctor report.
type Report struct {
	OS       string
	Arch     string
	OSDetail string

	Tools []ToolEntry
	Svc   []ServiceEntry
}

func osArch() (os, arch string) {
	return runtime.GOOS, runtime.GOARCH
}
