package agent

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// Reviewer (门下省) is responsible for reviewing and validating tasks.
type ReviewerAgent struct {
	name      string
	status    atomic.Value
	bus       *EventBus
	logger    *zap.Logger
	whitelist []string // Allowed task types
	mu        sync.Mutex
}

// NewReviewer creates a new Reviewer agent (门下省).
func NewReviewer(bus *EventBus, logger *zap.Logger) *ReviewerAgent {
	r := &ReviewerAgent{
		name:      "reviewer",
		bus:       bus,
		logger:    logger,
		whitelist: []string{"deployment", "creation", "inspection", "audit", "security", "resource", "scaling", "backup", "restore", "monitoring", "configuration"},
	}
	r.status.Store(StatusIdle)
	return r
}

// Name returns the agent's name.
func (r *ReviewerAgent) Name() string {
	return r.name
}

// Status returns the agent's current status.
func (r *ReviewerAgent) Status() AgentStatus {
	return r.status.Load().(AgentStatus)
}

// Handle processes a task by reviewing it.
func (r *ReviewerAgent) Handle(ctx context.Context, task *Task) (*Task, error) {
	r.status.Store(StatusWorking)
	defer r.status.Store(StatusIdle)

	approved, reason, err := r.Review(ctx, task)
	if err != nil {
		task.State = TaskStateFailed
		task.Error = err
		if r.bus != nil {
			r.bus.Emit(EventTaskFailed, r.name, task.ID, map[string]interface{}{
				"error": err.Error(),
			})
		}
		return task, err
	}

	if !approved {
		task.State = TaskStateFailed
		task.Error = &ReviewRejectionError{Reason: reason}
		if r.bus != nil {
			r.bus.Emit(EventTaskReviewed, r.name, task.ID, map[string]interface{}{
				"approved": false,
				"reason":   reason,
			})
		}
		return task, &ReviewRejectionError{Reason: reason}
	}

	task.State = TaskStateReviewing
	task.UpdatedAt = time.Now()

	if r.bus != nil {
		r.bus.Emit(EventTaskReviewed, r.name, task.ID, map[string]interface{}{
			"approved": true,
		})
	}

	if r.logger != nil {
		r.logger.Info("task reviewed and approved",
			zap.String("taskId", task.ID),
			zap.String("intent", task.Intent))
	}

	return task, nil
}

// Review validates a task before execution.
func (r *ReviewerAgent) Review(ctx context.Context, task *Task) (bool, string, error) {
	// Check if task type is allowed
	if task.Type != "" && !r.isAllowed(task.Type) {
		return false, "task type not allowed", nil
	}

	// Check if intent is valid
	if task.Intent == "" {
		return false, "no intent specified", nil
	}

	// Check input validity
	if task.Input == "" {
		return false, "no input provided", nil
	}

	// Check dangerous operations
	if isDangerousOperation(task.Input) {
		return false, "dangerous operation detected", nil
	}

	return true, "", nil
}

// ValidateResult checks if the task result is acceptable.
func (r *ReviewerAgent) ValidateResult(ctx context.Context, task *Task) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if task completed successfully
	if task.State != TaskStateCompleted {
		return false, nil
	}

	// Check for errors
	if task.Error != nil {
		return false, task.Error
	}

	// Validate result based on task type
	switch task.Type {
	case "audit":
		// Audit results should have findings
		if task.Result == nil {
			return true, nil // Empty audit is acceptable
		}
	case "security":
		// Security tasks should complete without errors
		return task.Error == nil, nil
	}

	return true, nil
}

// CanHandle returns true if the task type is reviewable.
func (r *ReviewerAgent) CanHandle(taskType string) bool {
	return true // Reviewer handles all task types
}

// Start initializes the Reviewer agent.
func (r *ReviewerAgent) Start(ctx context.Context) error {
	r.status.Store(StatusIdle)
	if r.logger != nil {
		r.logger.Info("Reviewer agent started")
	}
	return nil
}

// Stop shuts down the Reviewer agent.
func (r *ReviewerAgent) Stop(ctx context.Context) error {
	r.status.Store(StatusIdle)
	if r.logger != nil {
		r.logger.Info("Reviewer agent stopped")
	}
	return nil
}

// isAllowed checks if a task type is in the whitelist.
func (r *RevieweAgent) isAllowed(taskType string) bool {
	for _, t := range r.whitelist {
		if t == taskType {
			return true
		}
	}
	return false
}

// SetWhitelist sets the allowed task types.
func (r *ReviewerAgent) SetWhitelist(allowed []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.whitelist = allowed
}

// ReviewRejectionError represents a task rejection.
type ReviewRejectionError struct {
	Reason string
}

func (e *ReviewRejectionError) Error() string {
	return "task rejected: " + e.Reason
}

// isDangerousOperation checks for potentially dangerous operations.
func isDangerousOperation(input string) bool {
	dangerous := []string{
		"rm -rf /",
		"DROP TABLE",
		"DROP DATABASE",
		"DELETE FROM",
		"TRUNCATE",
		":(){ :|:& };:",
		"mkfs",
		"dd if=/dev/zero",
	}

	lowerInput := toLower(input)
	for _, pattern := range dangerous {
		if contains(lowerInput, toLower(pattern)) {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	// Simple lowercase conversion for ASCII
	result := make([]byte, len(s))
	for i, c := range s {
		if c >= 'A' && c <= 'Z' {
			result[i] = byte(c + 32)
		} else {
			result[i] = byte(c)
		}
	}
	return string(result)
}