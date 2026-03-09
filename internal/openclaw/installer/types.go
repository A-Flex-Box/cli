package installer

type InstallMethod string

const (
	MethodNative InstallMethod = "native"
	MethodDocker InstallMethod = "docker"
	MethodSource InstallMethod = "source"
)

type InstallVersion string

const (
	VersionStable InstallVersion = "stable"
	VersionBeta   InstallVersion = "beta"
	VersionDev    InstallVersion = "dev"
)

type InstallStatus string

const (
	InstallStatusNone       InstallStatus = "none"
	InstallStatusInstalling InstallStatus = "installing"
	InstallStatusInstalled  InstallStatus = "installed"
	InstallStatusFailed     InstallStatus = "failed"
	InstallStatusUninstall  InstallStatus = "uninstall"
)

type InstallOptions struct {
	Method      InstallMethod
	Version     InstallVersion
	Components  []string
	InstallPath string
	NoDaemon    bool
	NoGui       bool
}

type UninstallOptions struct {
	Purge bool
	NoGui bool
}

type DependencyError struct {
	Name        string
	Description string
	Required    bool
	HowToFix    string
}

type DownloadProgress struct {
	Component string
	Current   int64
	Total     int64
	Done      bool
	Error     error
}
