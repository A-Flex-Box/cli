package action

import "context"

type ProgressCallback func(stage string, current, total int, message string)

type Action interface {
	Execute(ctx context.Context, progress ProgressCallback) error
	Validate() error
	Steps() int
}

type Executor struct {
	action   Action
	progress ProgressCallback
}

func NewExecutor(action Action) *Executor {
	return &Executor{action: action}
}

func (e *Executor) SetProgress(cb ProgressCallback) {
	e.progress = cb
}

func (e *Executor) Execute(ctx context.Context) error {
	if err := e.action.Validate(); err != nil {
		return err
	}
	return e.action.Execute(ctx, e.progress)
}

func DefaultProgress(stage string, current, total int, message string) {
	if total > 0 {
		percent := (current * 100) / total
		println(stage, message, percent, "%")
	} else {
		println(stage, message)
	}
}
