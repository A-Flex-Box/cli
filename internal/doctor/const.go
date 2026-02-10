package doctor

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
