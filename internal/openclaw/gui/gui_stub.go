//go:build !fyne
// +build !fyne

package gui

func RunMainWindow(app *App) error {
	return ErrGUINotAvailable
}

var ErrGUINotAvailable = &GUIError{Message: "GUI not available. Build with 'fyne' tag to enable GUI mode."}

type GUIError struct {
	Message string
}

func (e *GUIError) Error() string {
	return e.Message
}
