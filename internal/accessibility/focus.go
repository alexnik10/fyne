package accessibility

import "fyne.io/fyne/v2"

// FocusScope initializes focus once when an input-capturing overlay opens. Keep
// one instance per canvas. Adapters opt into this together with accessibility;
// ordinary builds retain their existing popup behaviour. The canvas remains the
// only owner of keyboard focus and restores its previous manager on dismissal.
type FocusScope struct {
	top fyne.CanvasObject
}

// Update must be called on the Fyne event thread after the overlay is laid out.
func (s *FocusScope) Update(canvas fyne.Canvas) {
	top := canvas.Overlays().Top()
	if top == s.top {
		return
	}
	s.top = top
	if top != nil {
		if f, ok := canvas.Focused().(fyne.AccessibleFocusHandler); ok {
			f.AccessibilityFocus()
		}
	}
	if top != nil && canvas.Focused() == nil {
		canvas.FocusNext()
	}
}
