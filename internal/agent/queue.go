package agent

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// PriorityTaskQueue implements a priority-based task queue.
type PriorityTaskQueue struct {
	mu       sync.Mutex
	queues   map[TaskPriority][]*Task
	size     int32
	notifier chan struct{}
	logger   *zap.Logger
}

// NewPriorityTaskQueue creates a new priority task queue.
func NewPriorityTaskQueue(logger *zap.Logger) *PriorityTaskQueue {
	return &PriorityTaskQueue{
		queues: map[TaskPriority][]*Task{
			PriorityLow:    {},
			PriorityNormal: {},
			PriorityHigh:   {},
			PriorityUrgent: {},
		},
		notifier: make(chan struct{}, 1),
		logger:   logger,
	}
}

// Enqueue adds a task to the queue based on its priority.
func (q *PriorityTaskQueue) Enqueue(task *Task) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Set default priority if not set
	if task.Priority < PriorityLow || task.Priority > PriorityUrgent {
		task.Priority = PriorityNormal
	}

	// Set timestamps
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}
	task.UpdatedAt = time.Now()
	task.State = TaskStatePending

	q.queues[task.Priority] = append(q.queues[task.Priority], task)
	atomic.AddInt32(&q.size, 1)

	// Notify waiters
	select {
	case q.notifier <- struct{}{}:
	default:
	}

	if q.logger != nil {
		q.logger.Debug("task enqueued",
			zap.String("taskId", task.ID),
			zap.String("type", task.Type),
			zap.Int("priority", int(task.Priority)),
			zap.Int32("queueSize", atomic.LoadInt32(&q.size)))
	}

	return nil
}

// Dequeue removes and returns the highest priority task.
func (q *PriorityTaskQueue) Dequeue() (*Task, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Check queues from highest to lowest priority
	for priority := PriorityUrgent; priority >= PriorityLow; priority-- {
		if len(q.queues[priority]) > 0 {
			task := q.queues[priority][0]
			q.queues[priority] = q.queues[priority][1:]
			atomic.AddInt32(&q.size, -1)

			if q.logger != nil {
				q.logger.Debug("task dequeued",
					zap.String("taskId", task.ID),
					zap.String("type", task.Type))
			}

			return task, nil
		}
	}

	return nil, ErrQueueEmpty
}

// Peek returns the next task without removing it.
func (q *PriorityTaskQueue) Peek() (*Task, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for priority := PriorityUrgent; priority >= PriorityLow; priority-- {
		if len(q.queues[priority]) > 0 {
			return q.queues[priority][0], nil
		}
	}

	return nil, ErrQueueEmpty
}

// Size returns the total number of tasks in the queue.
func (q *PriorityTaskQueue) Size() int {
	return int(atomic.LoadInt32(&q.size))
}

// Clear removes all tasks from the queue.
func (q *PriorityTaskQueue) Clear() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for priority := range q.queues {
		q.queues[priority] = []*Task{}
	}
	atomic.StoreInt32(&q.size, 0)

	return nil
}

// Wait returns a channel that is notified when a task is available.
func (q *PriorityTaskQueue) Wait() <-chan struct{} {
	return q.notifier
}

// Stats returns queue statistics.
func (q *PriorityTaskQueue) Stats() map[TaskPriority]int {
	q.mu.Lock()
	defer q.mu.Unlock()

	stats := make(map[TaskPriority]int)
	for priority, tasks := range q.queues {
		stats[priority] = len(tasks)
	}
	return stats
}

// DequeueWithContext dequeues a task, blocking if empty until context is done.
func (q *PriorityTaskQueue) DequeueWithContext(ctx context.Context) (*Task, error) {
	for {
		task, err := q.Dequeue()
		if err == nil {
			return task, nil
		}

		if err != ErrQueueEmpty {
			return nil, err
		}

		// Wait for a task or context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-q.notifier:
			// Try again
		}
	}
}

// SimpleTaskQueue implements a simple FIFO task queue.
type SimpleTaskQueue struct {
	mu    sync.Mutex
	tasks []*Task
}

// NewSimpleTaskQueue creates a new simple task queue.
func NewSimpleTaskQueue() *SimpleTaskQueue {
	return &SimpleTaskQueue{
		tasks: make([]*Task, 0),
	}
}

// Enqueue adds a task to the end of the queue.
func (q *SimpleTaskQueue) Enqueue(task *Task) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	task.State = TaskStatePending

	q.tasks = append(q.tasks, task)
	return nil
}

// Dequeue removes and returns the task from the front of the queue.
func (q *SimpleTaskQueue) Dequeue() (*Task, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.tasks) == 0 {
		return nil, ErrQueueEmpty
	}

	task := q.tasks[0]
	q.tasks = q.tasks[1:]
	return task, nil
}

// Peek returns the next task without removing it.
func (q *SimpleTaskQueue) Peek() (*Task, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.tasks) == 0 {
		return nil, ErrQueueEmpty
	}

	return q.tasks[0], nil
}

// Size returns the number of tasks in the queue.
func (q *SimpleTaskQueue) Size() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.tasks)
}

// Clear removes all tasks from the queue.
func (q *SimpleTaskQueue) Clear() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.tasks = make([]*Task, 0)
	return nil
}