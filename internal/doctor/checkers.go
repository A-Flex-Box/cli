package doctor

// Standard tool checkers (binary in PATH + version).

type toolChecker struct {
	name        string
	bin         string
	versionArgs []string
}

func (t toolChecker) Name() string     { return t.name }
func (t toolChecker) Category() string { return "tool" }
func (t toolChecker) Check() Result {
	path, err := lookPath(t.bin)
	if err != nil {
		return Result{Tool: &ToolEntry{Name: t.name, Status: InstallStatusNotInstall}}
	}
	ver := ""
	if len(t.versionArgs) > 0 {
		ver = runVersion(path, t.versionArgs...)
	}
	return Result{Tool: &ToolEntry{Name: t.name, Path: path, Version: ver, Status: InstallStatusInstalled}}
}

// Standard service checkers (binary + default port listening).

type serviceChecker struct {
	name        string
	bin         string
	versionArgs []string
	port        string // default port for this service
}

func (s serviceChecker) Name() string     { return s.name }
func (s serviceChecker) Category() string { return "service" }
func (s serviceChecker) Check() Result {
	path, err := lookPath(s.bin)
	status := InstallStatusInstalled
	ver := ""
	if err != nil {
		path = ""
		status = InstallStatusNotInstall
	} else if len(s.versionArgs) > 0 {
		ver = runVersion(path, s.versionArgs...)
	}
	listening := ListeningNA
	portStatus := PortStatusNone
	if s.port != "" {
		listening = ListeningNo
		portStatus = PortStatusNotListening
		if portListening("127.0.0.1:" + s.port) {
			listening = ListeningYes
			portStatus = PortStatusListening
		}
	}
	return Result{
		Service: &ServiceEntry{
			Name:       s.name,
			Path:       path,
			Version:    ver,
			Status:     status,
			Port:       s.port,
			Listening:  listening,
			PortStatus: portStatus,
		},
	}
}

// cppChecker tries g++ then clang++.
type cppChecker struct{}

func (cppChecker) Name() string     { return "cpp" }
func (cppChecker) Category() string { return "tool" }
func (cppChecker) Check() Result {
	e := toolChecker{name: "cpp", bin: "g++", versionArgs: []string{"--version"}}.Check()
	if e.Tool != nil && e.Tool.Status == InstallStatusInstalled {
		return e
	}
	return toolChecker{name: "cpp", bin: "clang++", versionArgs: []string{"--version"}}.Check()
}

// pythonChecker prefers python3, falls back to python.
type pythonChecker struct{}

func (pythonChecker) Name() string     { return "py" }
func (pythonChecker) Category() string { return "tool" }
func (pythonChecker) Check() Result {
	e := toolChecker{name: "py", bin: "python3", versionArgs: []string{"--version"}}.Check()
	if e.Tool != nil && e.Tool.Status == InstallStatusInstalled {
		return e
	}
	return toolChecker{name: "py", bin: "python", versionArgs: []string{"--version"}}.Check()
}

// esChecker: Elasticsearch often has no CLI in PATH; we only check port 9200.
type esChecker struct{}

func (esChecker) Name() string     { return "es" }
func (esChecker) Category() string { return "service" }
func (esChecker) Check() Result {
	listening := ListeningNo
	portStatus := PortStatusNotListening
	if portListening("127.0.0.1:9200") {
		listening = ListeningYes
		portStatus = PortStatusListening
	}
	return Result{
		Service: &ServiceEntry{
			Name:       "es",
			Path:       "",
			Version:    "",
			Status:     InstallStatusNotInstall,
			Port:       "9200",
			Listening:  listening,
			PortStatus: portStatus,
		},
	}
}

// containerdChecker: binary is "containerd"; default client port often 10000 or socket.
type containerdChecker struct{}

func (containerdChecker) Name() string     { return "containerd" }
func (containerdChecker) Category() string { return "service" }
func (containerdChecker) Check() Result {
	s := serviceChecker{
		name: "containerd", bin: "containerd", versionArgs: []string{"--version"}, port: "10000",
	}.Check()
	// containerd often listens on 10000 when exposed; otherwise socket-only
	return s
}

func init() {
	// Tools
	DefaultRegistry.Register(toolChecker{"go", "go", []string{"version"}})
	DefaultRegistry.Register(toolChecker{"git", "git", []string{"--version"}})
	DefaultRegistry.Register(toolChecker{"make", "make", []string{"--version"}})
	DefaultRegistry.Register(toolChecker{"gcc", "gcc", []string{"--version"}})
	DefaultRegistry.Register(cppChecker{})
	DefaultRegistry.Register(pythonChecker{})
	DefaultRegistry.Register(toolChecker{"conda", "conda", []string{"--version"}})
	// Services (binary + port)
	DefaultRegistry.Register(serviceChecker{"docker", "docker", []string{"--version"}, "2375"})
	DefaultRegistry.Register(containerdChecker{})
	DefaultRegistry.Register(serviceChecker{"k8s", "kubectl", []string{"version", "--client", "--short"}, "6443"})
	DefaultRegistry.Register(serviceChecker{"etcd", "etcd", []string{"--version"}, "2379"})
	DefaultRegistry.Register(serviceChecker{"mysql", "mysql", []string{"--version"}, "3306"})
	DefaultRegistry.Register(serviceChecker{"pg", "psql", []string{"--version"}, "5432"})
	DefaultRegistry.Register(esChecker{})
}
