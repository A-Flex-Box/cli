package agent

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// Oracle (中书省) is responsible for intent parsing and task decomposition.
type OracleAgent struct {
	name     string
	status   atomic.Value
	bus      *EventBus
	logger   *zap.Logger
	registry AgentRegistry
	mu       sync.Mutex
}

// NewOracle creates a new Oracle agent (中书省).
func NewOracle(bus *EventBus, registry AgentRegistry, logger *zap.Logger) *OracleAgent {
	o := &OracleAgent{
		name:     "oracle",
		bus:      bus,
		logger:   logger,
		registry: registry,
	}
	o.status.Store(StatusIdle)
	return o
}

// Name returns the agent's name.
func (o *OracleAgent) Name() string {
	return o.name
}

// Status returns the agent's current status.
func (o *OracleAgent) Status() AgentStatus {
	return o.status.Load().(AgentStatus)
}

// Handle processes a task by parsing its intent.
func (o *OracleAgent) Handle(ctx context.Context, task *Task) (*Task, error) {
	o.status.Store(StatusWorking)
	defer o.status.Store(StatusIdle)

	if o.bus != nil {
		o.bus.Emit(EventTaskParsed, o.name, task.ID, map[string]interface{}{
			"input": task.Input,
		})
	}

	// Parse intent
	intent, params, err := o.ParseIntent(ctx, task.Input)
	if err != nil {
		task.State = TaskStateFailed
		task.Error = fmt.Errorf("intent parsing failed: %w", err)
		if o.bus != nil {
			o.bus.Emit(EventTaskFailed, o.name, task.ID, map[string]interface{}{
				"error": err.Error(),
			})
		}
		return task, err
	}

	task.Intent = intent
	task.Params = params
	task.State = TaskStateParsing
	task.UpdatedAt = time.Now()

	if o.logger != nil {
		o.logger.Info("intent parsed",
			zap.String("taskId", task.ID),
			zap.String("intent", intent),
			zap.Any("params", params))
	}

	return task, nil
}

// ParseIntent analyzes input and extracts intent and parameters.
func (o *OracleAgent) ParseIntent(ctx context.Context, input string) (string, map[string]interface{}, error) {
	// Default implementation - can be extended with NLU
	intent := "unknown"
	params := make(map[string]interface{})

	// Simple intent classification based on keywords
	keywords := map[string]string{
		"deploy":    "deployment",
		"create":    "creation",
		"delete":    "deletion",
		"check":     "inspection",
		"audit":     "audit",
		"security":  "security",
		"resource":  "resource",
		"scale":     "scaling",
		"backup":    "backup",
		"restore":   "restore",
		"monitor":   "monitoring",
		"configure": "configuration",
	}

	for keyword, intentType := range keywords {
		if contains(input, keyword) {
			intent = intentType
			break
		}
	}

	params["raw_input"] = input
	params["parsed_at"] = time.Now().Format(time.RFC3339)

	return intent, params, nil
}

// Decompose breaks down a complex task into subtasks.
func (o *OracleAgent) Decompose(ctx context.Context, task *Task) ([]*Task, error) {
	if o.logger != nil {
		o.logger.Debug("decomposing task", zap.String("taskId", task.ID))
	}

	// Default implementation - creates subtasks based on intent
	var subtasks []*Task

	switch task.Intent {
	case "deployment":
		subtasks = []*Task{
			{
				ID:       generateID(),
				Type:     "infrastructure",
				Input:    "prepare infrastructure",
				Priority: task.Priority,
				ParentID: task.ID,
			},
			{
				ID:       generateID(),
				Type:     "security",
				Input:    "security check",
				Priority: task.Priority,
				ParentID: task.ID,
			},
			{
				ID:       generateID(),
				Type:     "resource",
				Input:    "allocate resources",
				Priority: task.Priority,
				ParentID: task.ID,
			},
		}
	case "audit":
		subtasks = []*Task{
			{
				ID:       generateID(),
				Type:     "audit",
				Input:    "run audit",
				Priority: task.Priority,
				ParentID: task.ID,
			},
		}
	default:
		subtasks = []*Task{
			{
				ID:       generateID(),
				Type:     task.Intent,
				Input:    task.Input,
				Priority: task.Priority,
				ParentID: task.ID,
			},
		}
	}

	task.Children = subtasks
	return subtasks, nil
}

// CanHandle returns true for all task types (Oracle handles everything).
func (o *OracleAgent) CanHandle(taskType string) bool {
	return true
}

// Start initializes the Oracle agent.
func (o *OracleAgent) Start(ctx context.Context) error {
	o.status.Store(StatusIdle)
	if o.logger != nil {
		o.logger.Info("Oracle agent started")
	}
	return nil
}

// Stop shuts down the Oracle agent.
func (o *OracleAgent) Stop(ctx context.Context) error {
	o.status.Store(StatusIdle)
	if o.logger != nil {
		o.logger.Info("Oracle agent stopped")
	}
	return nil
}