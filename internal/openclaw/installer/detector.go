package installer

import (
	"bytes"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type SystemInfo struct {
	OS      string
	Arch    string
	Distro  string
	Version string

	NodeVersion string
	NpmVersion  string
	DockerVer   string
	GitVersion  string

	Systemd   bool
	Launchd   bool
	WSL       bool
	Tailscale bool

	InUsePorts []int
}

func DetectSystem() *SystemInfo {
	info := &SystemInfo{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	info.detectDistro()
	info.detectNode()
	info.detectNpm()
	info.detectDocker()
	info.detectGit()
	info.detectServiceManager()
	info.detectWSL()
	info.detectTailscale()

	return info
}

func (s *SystemInfo) detectDistro() {
	if s.OS != "linux" {
		return
	}

	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "ID=") {
				s.Distro = strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
			}
			if strings.HasPrefix(line, "VERSION_ID=") {
				s.Version = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
			}
		}
	}

	if s.Distro == "" {
		s.Distro = "unknown"
	}
}

func (s *SystemInfo) detectNode() {
	if version, err := exec.Command("node", "--version").Output(); err == nil {
		s.NodeVersion = strings.TrimSpace(string(version))
	}
}

func (s *SystemInfo) detectNpm() {
	if version, err := exec.Command("npm", "--version").Output(); err == nil {
		s.NpmVersion = strings.TrimSpace(string(version))
	}
}

func (s *SystemInfo) detectDocker() {
	if version, err := exec.Command("docker", "--version").Output(); err == nil {
		s.DockerVer = strings.TrimSpace(string(version))
	}
}

func (s *SystemInfo) detectGit() {
	if version, err := exec.Command("git", "--version").Output(); err == nil {
		s.GitVersion = strings.TrimSpace(string(version))
	}
}

func (s *SystemInfo) detectServiceManager() {
	if s.OS == "linux" {
		if _, err := os.Stat("/run/systemd/system"); err == nil {
			s.Systemd = true
		}
	}
	if s.OS == "darwin" {
		if _, err := exec.LookPath("launchctl"); err == nil {
			s.Launchd = true
		}
	}
}

func (s *SystemInfo) detectWSL() {
	if s.OS != "linux" {
		return
	}
	if data, err := os.ReadFile("/proc/version"); err == nil {
		if strings.Contains(strings.ToLower(string(data)), "microsoft") {
			s.WSL = true
		}
	}
}

func (s *SystemInfo) detectTailscale() {
	if _, err := exec.LookPath("tailscale"); err == nil {
		s.Tailscale = true
	}
}

func (s *SystemInfo) CheckDependencies(method InstallMethod) []DependencyError {
	var errors []DependencyError

	switch method {
	case MethodNative:
		if s.NodeVersion == "" {
			errors = append(errors, DependencyError{
				Name:        "Node.js",
				Description: "Node.js >= 22 is required",
				Required:    true,
				HowToFix:    "Install Node.js 22+ from https://nodejs.org or use nvm",
			})
		}
		if s.NpmVersion == "" {
			errors = append(errors, DependencyError{
				Name:        "npm",
				Description: "npm is required for installation",
				Required:    true,
				HowToFix:    "npm is included with Node.js",
			})
		}

	case MethodDocker:
		if s.DockerVer == "" {
			errors = append(errors, DependencyError{
				Name:        "Docker",
				Description: "Docker is required for container installation",
				Required:    true,
				HowToFix:    "Install Docker from https://docs.docker.com/get-docker/",
			})
		}

	case MethodSource:
		if s.GitVersion == "" {
			errors = append(errors, DependencyError{
				Name:        "Git",
				Description: "Git is required for cloning repository",
				Required:    true,
				HowToFix:    "Install Git from https://git-scm.com",
			})
		}
		if s.NodeVersion == "" {
			errors = append(errors, DependencyError{
				Name:        "Node.js",
				Description: "Node.js >= 22 is required for building",
				Required:    true,
				HowToFix:    "Install Node.js 22+ from https://nodejs.org",
			})
		}
	}

	return errors
}

func (s *SystemInfo) HasDisplay() bool {
	if s.OS == "windows" {
		return true
	}
	if s.OS == "darwin" {
		return os.Getenv("DISPLAY") != "" || os.Getenv("TERM") != ""
	}
	return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
}

func (s *SystemInfo) GetServiceType() string {
	switch {
	case s.Systemd:
		return "systemd"
	case s.Launchd:
		return "launchd"
	case s.WSL:
		return "wsl"
	default:
		return "none"
	}
}

func IsPortInUse(port int) bool {
	cmd := exec.Command("sh", "-c", "netstat -tuln 2>/dev/null || ss -tuln 2>/dev/null")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return bytes.Contains(output, []byte(":"+strconv.Itoa(port)))
}

func (s *SystemInfo) DetectPortsInUse(ports []int) []int {
	var inUse []int
	for _, port := range ports {
		if IsPortInUse(port) {
			inUse = append(inUse, port)
		}
	}
	return inUse
}
