package doctor

import (
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// lookPath returns path and nil if found; empty path and err if not.
func lookPath(name string) (path string, err error) {
	return exec.LookPath(name)
}

// runVersion runs binary with versionArgs and returns first line of stdout (trimmed).
func runVersion(path string, versionArgs ...string) string {
	cmd := exec.Command(path, versionArgs...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(strings.Split(string(out), "\n")[0])
	return strings.TrimSpace(line)
}

// portListening checks if addr (e.g. "127.0.0.1:3306") is listening.
func portListening(addr string) bool {
	c, err := net.Dial("tcp", addr)
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}

// GetOSDetail returns a human-readable OS string.
func GetOSDetail() string {
	switch runtime.GOOS {
	case "linux":
		return readLinuxOSRelease()
	case "darwin":
		return runDarwinOSVersion()
	default:
		return runtime.GOOS
	}
}

func readLinuxOSRelease() string {
	data, err := readFileTrim("/etc/os-release")
	if err != nil {
		return "linux"
	}
	lines := strings.Split(data, "\n")
	var name, version string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "PRETTY_NAME=") {
			s := strings.TrimPrefix(l, "PRETTY_NAME=")
			return strings.Trim(s, "\"")
		}
		if strings.HasPrefix(l, "ID=") {
			name = strings.Trim(strings.TrimPrefix(l, "ID="), "\"")
		}
		if strings.HasPrefix(l, "VERSION_ID=") {
			version = strings.Trim(strings.TrimPrefix(l, "VERSION_ID="), "\"")
		}
	}
	if name != "" {
		if version != "" {
			return name + " " + version
		}
		return name
	}
	return "linux"
}

func runDarwinOSVersion() string {
	cmd := exec.Command("sw_vers", "-productVersion")
	out, err := cmd.Output()
	if err != nil {
		return "darwin"
	}
	return "macOS " + strings.TrimSpace(string(out))
}

func readFileTrim(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}
