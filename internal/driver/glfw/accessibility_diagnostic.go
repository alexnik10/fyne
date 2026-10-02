package glfw

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/internal/accessibility/diagnostic"
)

func (w *window) diagnosticSpaceKey(key fyne.KeyName, event action) {
	if key != fyne.KeySpace {
		return
	}
	kind := "space_press"
	switch event {
	case release:
		kind = "space_release"
	case repeat:
		kind = "space_repeat"
	}
	w.diagnosticSpaceEvent(kind)
}

func (w *window) diagnosticSpaceEvent(kind string) {
	if control, ok := w.canvas.Focused().(diagnostic.Control); ok {
		diagnostic.Add(diagnostic.Sample{Kind: kind, Window: diagnostic.WindowID(w), Control: control.DiagnosticControlID()})
	}
}
