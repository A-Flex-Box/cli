package doctor

import (
	"runtime"
	"sort"
	"sync"
)

// DefaultRegistry is the global registry of checkers.
var DefaultRegistry = NewRegistry()

// Registry holds checkers and runs them concurrently.
type Registry struct {
	mu       sync.Mutex
	checkers []Checker
}

// NewRegistry returns a new empty registry.
func NewRegistry() *Registry {
	return &Registry{checkers: make([]Checker, 0)}
}

// Register adds c to the registry. Safe for concurrent use.
func (r *Registry) Register(c Checker) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checkers = append(r.checkers, c)
}

// Run executes all checkers concurrently and returns a Report.
func (r *Registry) Run() *Report {
	r.mu.Lock()
	list := make([]Checker, len(r.checkers))
	copy(list, r.checkers)
	r.mu.Unlock()

	osName, arch := runtime.GOOS, runtime.GOARCH
	osDetail := GetOSDetail()

	type resultWithOrder struct {
		Result Result
		Index  int
	}
	ch := make(chan resultWithOrder, len(list))
	for i, c := range list {
		i, c := i, c
		go func() {
			res := c.Check()
			ch <- resultWithOrder{Result: res, Index: i}
		}()
	}

	var tools []ToolEntry
	var svc []ServiceEntry
	order := make([]resultWithOrder, 0, len(list))
	for range list {
		order = append(order, <-ch)
	}
	close(ch)

	sort.Slice(order, func(i, j int) bool {
		return order[i].Index < order[j].Index
	})
	for _, o := range order {
		if o.Result.Tool != nil {
			tools = append(tools, *o.Result.Tool)
		}
		if o.Result.Service != nil {
			svc = append(svc, *o.Result.Service)
		}
	}

	return &Report{
		OS:       osName,
		Arch:     arch,
		OSDetail: osDetail,
		Tools:    tools,
		Svc:      svc,
	}
}
