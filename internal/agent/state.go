package agent

import (
	"errors"
	"sync"
)

// TaskStateMachine implements a state machine for task lifecycle.
type TaskStateMachine struct {
	mu        sync.RWMutex
	state     TaskState
	task      *Task
	transitions map[TaskState][]TaskState
}

// NewTaskStateMachine creates a new state machine for a task.
func NewTaskStateMachine(task *Task) *TaskStateMachine {
	// Define valid state transitions
	transitions := map[TaskState][]TaskState{
		TaskStatePending: {
			TaskStateParsing,
			TaskStateCancelled,
		},
		TaskStateParsing: {
			TaskStateReviewing,
			TaskStateFailed,
			TaskStateCancelled,
		},
		TaskStateReviewing: {
			TaskStateExecuting,
			TaskStatePending,  // Back to pending for re-parsing
			TaskStateFailed,
			TaskStateCancelled,
		},
		TaskStateExecuting: {
			TaskStateCompleted,
			TaskStateFailed,
			TaskStateCancelled,
		},
		TaskStateCompleted: {}, // Terminal state
		TaskStateFailed:    {}, // Terminal state
		TaskStateCancelled: {}, // Terminal state
	}

	return &TaskStateMachine{
		state:       task.State,
		task:        task,
		transitions: transitions,
	}
}

// CurrentState returns the current state.
func (sm *TaskStateMachine) CurrentState() TaskState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state
}

// Transition moves to a new state if valid.
func (sm *TaskStateMachine) Transition(to TaskState) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.canTransitionLocked(to) {
		return NewStateTransitionError(sm.state, to)
	}

	sm.state = to
	sm.task.State = to
	sm.task.UpdatedAt = now()

	return nil
}

// CanTransition checks if a transition is valid.
func (sm *TaskStateMachine) CanTransition(to TaskState) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.canTransitionLocked(to)
}

func (sm *TaskStateMachine) canTransitionLocked(to TaskState) bool {
	validNext, exists := sm.transitions[sm.state]
	if !exists {
		return false
	}

	for _, s := range validNext {
		if s == to {
			return true
		}
	}

	return false
}

// ValidTransitions returns all valid next states.
func (sm *TaskStateMachine) ValidTransitions() []TaskState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if transitions, exists := sm.transitions[sm.state]; exists {
		return transitions
	}
	return []TaskState{}
}

// IsTerminal returns true if current state is terminal.
func (sm *TaskStateMachine) IsTerminal() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.state == TaskStateCompleted ||
		sm.state == TaskStateFailed ||
		sm.state == TaskStateCancelled
}

// Graph returns the full transition graph.
func (sm *TaskStateMachine) Graph() map[TaskState][]TaskState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Return a copy
	result := make(map[TaskState][]TaskState)
	for k, v := range sm.transitions {
		result[k] = append([]TaskState{}, v...)
	}
	return result
}

// StateTransitionGraph returns a human-readable description of all possible transitions.
func StateTransitionGraph() string {
	return `
Task State Transition Graph:

  ┌─────────┐
  │ PENDING │────┐
  └────┬────┘    │
       │         ▼
       │   ┌───────────┐
       ▼   │ CANCELLED │ (terminal)
  ┌─────────┴───────────┘
  │  PARSING  │
  └─────┬─────┘
        │
        ▼
  ┌───────────┐
  │ REVIEWING │◄──────────┐
  └─────┬─────┘           │
        │                 │
        ▼                 │
  ┌───────────┐           │
  │ EXECUTING │           │
  └─────┬─────┘           │
        │                 │
   ┌────┴────┐            │
   ▼         ▼            │
┌──────────┐ ┌────────┐   │
│COMPLETED│ │ FAILED │───┘
└─────────┘ └────────┘
(terminal)  (terminal)
`
}

// Errors
var (
	ErrQueueEmpty = errors.New("queue is empty")

	ErrInvalidStateTransition = errors.New("invalid state transition")
)

// StateTransitionError represents an invalid state transition.
type StateTransitionError struct {
	From TaskState
	To   TaskState
}

// NewStateTransitionError creates a new state transition error.
func NewStateTransitionError(from, to TaskState) *StateTransitionError {
	return &StateTransitionError{From: from, To: to}
}

func (e *StateTransitionError) Error() string {
	return string(e.From) + " -> " + string(e.To) + ": " + ErrInvalidStateTransition.Error()
}