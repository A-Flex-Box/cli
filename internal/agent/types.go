// Package agent implements the Three Provinces Six Ministries (三省六部) Agent architecture.
// 
// The architecture is inspired by the ancient Chinese governmental system:
// - Oracle (中书省): Intent parsing and task decomposition
// - Reviewer (门下省): Review and validation
// - Executor (尚书省): Task scheduling and coordination
// - Ministers (六部): Domain-specific task handlers
package agent

import (
	"context"
	"time"
)

// AgentStatus represents the current state of an agent.
type AgentStatus string

const (
	StatusIdle     AgentStatus = "idle"
	StatusWorking  AgentStatus = "working"
	StatusWaiting  AgentStatus = "waiting"
	StatusError    AgentStatus = "error"
	StatusComplete AgentStatus = "complete"
)

// TaskPriority represents the priority level of a task.
type TaskPriority int

const (
	PriorityLow    TaskPriority = 0
	PriorityNormal TaskPriority = 1
	PriorityHigh   TaskPriority = 2
	PriorityUrgent TaskPriority = 3
)

// TaskState represents the state in a task's lifecycle.
type TaskState string

const (
	TaskStatePending   TaskState = "pending"
	TaskStateParsing   TaskState = "parsing"
	TaskStateReviewing TaskState = "reviewing"
	TaskStateExecuting TaskState = "executing"
	TaskStateCompleted TaskState = "completed"
	TaskStateFailed    TaskState = "failed"
	TaskStateCancelled TaskState = "cancelled"
)

// EventType represents the type of event in the system.
type EventType string

const (
	EventTaskCreated   EventType = "task.created"
	EventTaskParsed    EventType = "task.parsed"
	EventTaskReviewed  EventType = "task.reviewed"
	EventTaskExecuting EventType = "task.executing"
	EventTaskCompleted EventType = "task.completed"
	EventTaskFailed    EventType = "task.failed"
	EventAgentStatus   EventType = "agent.status"
	EventError         EventType = "system.error"
)

// Task represents a unit of work in the system.
type Task struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Input       string                 `json:"input"`
	Intent      string                 `json:"intent,omitempty"`
	Params      map[string]interface{} `json:"params,omitempty"`
	Priority    TaskPriority           `json:"priority"`
	State       TaskState              `json:"state"`
	Result      interface{}            `json:"result,omitempty"`
	Error       error                  `json:"error,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	CompletedAt *time.Time             `json:"completedAt,omitempty"`
	ParentID    string                 `json:"parentId,omitempty"`
	Children    []*Task                `json:"children,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`

	ctx    context.Context
	cancel context.CancelFunc
}

// Context returns the task's context.
func (t *Task) Context() context.Context {
	if t.ctx == nil {
		t.ctx, t.cancel = context.WithCancel(context.Background())
	}
	return t.ctx
}

// Cancel cancels the task's context.
func (t *Task) Cancel() {
	if t.cancel != nil {
		t.cancel()
	}
}

// Event represents a system event.
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Source    string                 `json:"source"`
	TaskID    string                 `json:"taskId,omitempty"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// Agent is the core interface for all agents in the system.
type Agent interface {
	// Name returns the agent's name.
	Name() string

	// Status returns the agent's current status.
	Status() AgentStatus

	// Handle processes a task and returns the result.
	Handle(ctx context.Context, task *Task) (*Task, error)

	// CanHandle checks if this agent can handle the given task type.
	CanHandle(taskType string) bool

	// Start initializes the agent.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the agent.
	Stop(ctx context.Context) error
}

// Minister is an agent that handles domain-specific tasks.
type Minister interface {
	Agent

	// Jurisdiction returns the domain this minister governs.
	Jurisdiction() string

	// Capabilities returns the list of capabilities this minister supports.
	Capabilities() []string
}

// Oracle (中书省) parses intent and decomposes tasks.
type Oracle interface {
	Agent

	// ParseIntent analyzes input and extracts intent.
	ParseIntent(ctx context.Context, input string) (string, map[string]interface{}, error)

	// Decompose breaks down a complex task into subtasks.
	Decompose(ctx context.Context, task *Task) ([]*Task, error)
}

// Reviewer (门下省) reviews and validates tasks.
type Reviewer interface {
	Agent

	// Review validates a task before execution.
	Review(ctx context.Context, task *Task) (bool, string, error)

	// ValidateResult checks if the task result is acceptable.
	ValidateResult(ctx context.Context, task *Task) (bool, error)
}

// Executor (尚书省) coordinates task execution.
type Executor interface {
	Agent

	// Schedule queues a task for execution.
	Schedule(ctx context.Context, task *Task) error

	// Assign routes a task to the appropriate minister.
	Assign(ctx context.Context, task *Task) (Minister, error)

	// Monitor returns the current task queue status.
	Monitor(ctx context.Context) (map[string]int, error)
}

// TaskQueue manages the queue of tasks.
type TaskQueue interface {
	// Enqueue adds a task to the queue.
	Enqueue(task *Task) error

	// Dequeue removes and returns the next task.
	Dequeue() (*Task, error)

	// Peek returns the next task without removing it.
	Peek() (*Task, error)

	// Size returns the current queue size.
	Size() int

	// Clear removes all tasks from the queue.
	Clear() error
}

// StateMachine manages task state transitions.
type StateMachine interface {
	// CurrentState returns the current state.
	CurrentState() TaskState

	// Transition moves to a new state if valid.
	Transition(to TaskState) error

	// CanTransition checks if a transition is valid.
	CanTransition(to TaskState) bool

	// ValidTransitions returns all valid next states.
	ValidTransitions() []TaskState
}

// AgentRegistry maintains a registry of available agents.
type AgentRegistry interface {
	// Register adds an agent to the registry.
	Register(agent Agent) error

	// Unregister removes an agent from the registry.
	Unregister(name string) error

	// Get retrieves an agent by name.
	Get(name string) (Agent, bool)

	// GetByType retrieves agents that can handle a task type.
	GetByType(taskType string) []Agent

	// List returns all registered agents.
	List() []Agent
}