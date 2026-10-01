package widget

import "fyne.io/fyne/v2"

// Multiline editors consume Tab as text, so provide an explicit way to leave
// them without editing the document. Use Control even on platforms where the
// usual command modifier is different (Command+Tab belongs to the OS).
func (e *Entry) handleFocusShortcut(shortcut fyne.Shortcut) bool {
	key, ok := shortcut.(fyne.KeyboardShortcut)
	if !e.MultiLine || !ok || key.Key() != fyne.KeyTab {
		return false
	}
	modifiers := key.Mod()
	if modifiers != fyne.KeyModifierControl && modifiers != fyne.KeyModifierControl|fyne.KeyModifierShift {
		return false
	}
	c := fyne.CurrentApp().Driver().CanvasForObject(e.super())
	owner, focusable := e.super().(fyne.Focusable)
	if c == nil || !focusable || c.Focused() != owner {
		return true
	}
	if modifiers&fyne.KeyModifierShift != 0 {
		c.FocusPrevious()
	} else {
		c.FocusNext()
	}
	return true
}
