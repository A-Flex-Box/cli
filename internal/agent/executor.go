package agent

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// Executor (尚书省) coordinates task execution and routes to ministers.
type ExecutorAgent struct {
	name      string
	status    atomic.Value
	bus       *EventBus
	logger    *zap.Logger
	queue     TaskQueue
	registry  AgentRegistry
	ministers map[string]Minister
	running   atomic.Bool
	mu        sync.RWMutex
}

// NewExecutor creates a new Executor agent (尚书省).
func NewExecutor(bus *EventBus, registry AgentRegistry, logger *zap.Logger) *ExecutorAgent {
	e := &ExecutorAgent{
		name:      "executor",
		bus:       bus,
		logger:    logger,
		queue:     NewPriorityTaskQueue(logger),
		registry:  registry,
		ministers: make(map[string]Minister),
	}
	e.status.Store(StatusIdle)
	return e
}

// Name returns the agent's name.
func (e *ExecutorAgent) Name() string {
	return e.name
}

// Status returns the agent's current status.
func (e *ExecutorAgent) Status() AgentStatus {
	return e.status.Load().(AgentStatus)
}

// Handle processes a task by coordinating its execution.
func (e *ExecutorAgent) Handle(ctx context.Context, task *Task) (*Task, error) {
	e.status.Store(StatusWorking)
	defer e.status.Store(StatusIdle)

	// Schedule the task
	if err := e.Schedule(ctx, task); err != nil {
		return task, fmt.Errorf("schedule failed: %w", err)
	}

	// Assign to a minister
	minister, err := e.Assign(ctx, task)
	if err != nil {
		task.State = TaskStateFailed
		task.Error = err
		if e.bus != nil {
			e.bus.Emit(EventTaskFailed, e.name, task.ID, map[string]interface{}{
				"error": err.Error(),
			})
		}
		return task, err
	}

	// Execute via minister
	e.status.Store(StatusWorking)
	task.State = TaskStateExecuting
	task.UpdatedAt = time.Now()

	if e.bus != nil {
		e.bus.Emit(EventTaskExecuting, e.name, task.ID, map[string]interface{}{
			"minister": minister.Name(),
		})
	}

	result, err := minister.Handle(ctx, task)
	if err != nil {
		task.State = TaskStateFailed
		task.Error = err
		if e.bus != nil {
			e.bus.Emit(EventTaskFailed, e.name, task.ID, map[string]interface{}{
				"error":   err.Error(),
				"minister": minister.Name(),
			})
		}
		return task, err
	}

	now := time.Now()
	result.State = TaskStateCompleted
	result.UpdatedAt = now
	result.CompletedAt = &now

	if e.bus != nil {
		e.bus.Emit(EventTaskCompleted, e.name, task.ID, map[string]interface{}{
			"minister": minister.Name(),
		})
	}

	return result, nil
}

// Schedule queues a task for execution.
func (e *ExecutorAgent) Schedule(ctx context.Context, task *Task) error {
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}
	task.UpdatedAt = time.Now()

	if err := e.queue.Enqueue(task); err != nil {
		return fmt.Errorf("enqueue failed: %w", err)
	}

	if e.logger != nil {
		e.logger.Debug("task scheduled",
			zap.String("taskId", task.ID),
			zap.String("type", task.Type))
	}

	return nil
}

// Assign routes a task to the appropriate minister.
func (e *ExecutorAgent) Assign(ctx context.Context, task *Task) (Minister, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Try to find a minister by task type
	if minister, ok := e.ministers[task.Type]; ok {
		return minister, nil
	}

	// Try to find by intent
	if minister, ok := e.ministers[task.Intent]; ok {
		return minister, nil
	}

	// Find a minister that can handle this task type
	for _, minister := range e.ministers {
		if minister.CanHandle(task.Type) {
			return minister, nil
		}
	}

	return nil, fmt.Errorf("no minister available for task type: %s", task.Type)
}

// Monitor returns the current task queue status.
func (e *ExecutorAgent) Monitor(ctx context.Context) (map[string]int, error) {
	if pq, ok := e.queue.(*PriorityTaskQueue); ok {
		stats := pq.Stats()
		result := make(map[string]int)
		for priority, count := range stats {
			result[string(priority)] = count
		}
		return result, nil
	}

	return map[string]int{
		"total": e.queue.Size(),
	}, nil
}

// RegisterMinister registers a minister with the executor.
func (e *ExecutorAgent) RegisterMinister(minister Minister) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.ministers[minister.Jurisdiction()] = minister

	if e.logger != nil {
		e.logger.Info("minister registered",
			zap.String("name", minister.Name()),
			zap.String("jurisdiction", minister.Jurisdiction()))
	}

	return nil
}

// UnregisterMinister removes a minister from the executor.
func (e *ExecutorAgent) UnregisterMinister(jurisdiction string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.ministers, jurisdiction)
	return nil
}

// CanHandle returns true for all task types (Executor coordinates all).
func (e *ExecutorAgent) CanHandle(taskType string) bool {
	return true
}

// Start initializes the Executor agent.
func (e *ExecutorAgent) Start(ctx context.Context) error {
	e.status.Store(StatusIdle)
	e.running.Store(true)

	// Start all ministers
	for _, minister := range e.ministers {
		if err := minister.Start(ctx); err != nil {
			return fmt.Errorf("failed to start minister %s: %w", minister.Name(), err)
		}
	}

	if e.logger != nil {
		e.logger.Info("Executor agent started",
			zap.Int("ministers", len(e.ministers)))
	}

	return nil
}

// Stop shuts down the Executor agent.
func (e *ExecutorAgent) Stop(ctx context.Context) error {
	e.running.Store(false)
	e.status.Store(StatusIdle)

	// Stop all ministers
	for _, minister := range e.ministers {
		if err := minister.Stop(ctx); err != nil {
			if e.logger != nil {
				e.logger.Warn("failed to stop minister",
					zap.String("name", minister.Name()),
					zap.Error(err))
			}
		}
	}

	if e.logger != nil {
		e.logger.Info("Executor agent stopped")
	}

	return nil
}

// ProcessQueue starts processing tasks from the queue.
func (e *ExecutorAgent) ProcessQueue(ctx context.Context) error {
	pq, ok := e.queue.(*PriorityTaskQueue)
	if !ok {
		return fmt.Errorf("queue does not support context-aware dequeue")
	}

	for e.running.Load() {
		task, err := pq.DequeueWithContext(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			continue
		}

		// Process the task
		_, err = e.Handle(ctx, task)
		if err != nil {
			if e.logger != nil {
				e.logger.Error("task processing failed",
					zap.String("taskId", task.ID),
					zap.Error(err))
			}
		}
	}

	return nil
}

// GetMinister retrieves a minister by jurisdiction.
func (e *ExecutorAgent) GetMinister(jurisdiction string) (Minister, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	minister, ok := e.ministers[jurisdiction]
	return minister, ok
}

// ListMinisters returns all registered ministers.
func (e *ExecutorAgent) ListMinisters() []Minister {
	e.mu.RLock()
	defer e.mu.RUnlock()

	ministers := make([]Minister, 0, len(e.ministers))
	for _, minister := range e.ministers {
		ministers = append(ministers, minister)
	}
	return ministers
}