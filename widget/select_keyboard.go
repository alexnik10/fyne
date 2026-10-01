package widget

import (
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/internal/goos"
)

// Windows combo boxes keep a collapsed arrow selection distinct from a popup's
// tentative highlight. Keep this policy out of the accessibility adapter.
func (s *Select) typedKeyWindows(event *fyne.KeyEvent) bool {
	if s.Disabled() {
		return true
	}
	switch event.Name {
	case fyne.KeyUp, fyne.KeyDown, fyne.KeyLeft, fyne.KeyRight:
		if len(s.Options) == 0 {
			return true
		}
		index := s.SelectedIndex()
		var next int
		if index < 0 {
			next = 0
		} else if event.Name == fyne.KeyUp || event.Name == fyne.KeyLeft {
			next = max(0, index-1)
		} else {
			next = min(len(s.Options)-1, index+1)
		}
		if next != index {
			s.SetSelectedIndex(next)
		}
	case fyne.KeySpace, fyne.KeyEnter, fyne.KeyReturn:
		if len(s.Options) > 0 {
			s.AccessibilitySetExpanded(true)
		}
	default:
		return false
	}
	return true
}

func isSelectDisclosureShortcut(shortcut fyne.Shortcut) bool {
	s, ok := shortcut.(*desktop.CustomShortcut)
	return ok && s.Modifier == fyne.KeyModifierAlt && (s.KeyName == fyne.KeyUp || s.KeyName == fyne.KeyDown)
}

// TypedShortcut handles Windows Alt+Up/Down and forwards other shortcuts to the canvas.
//
// Since: 2.9
func (s *Select) TypedShortcut(shortcut fyne.Shortcut) {
	if runtime.GOOS == goos.Windows && isSelectDisclosureShortcut(shortcut) {
		if len(s.Options) > 0 {
			s.AccessibilitySetExpanded(true)
		}
		return
	}
	if c, ok := fyne.CurrentApp().Driver().CanvasForObject(s.super()).(fyne.Shortcutable); ok {
		c.TypedShortcut(shortcut)
	}
}

// AcceptsTab lets a Windows Select popup commit before the canvas moves focus.
// Ordinary menus retain their existing Tab behavior.
//
// Since: 2.9
func (p *PopUpMenu) AcceptsTab() bool {
	return runtime.GOOS == goos.Windows && p.selectOwner != nil
}

// TypedShortcut handles Windows Alt+Up/Down while a Select popup is open.
//
// Since: 2.9
func (p *PopUpMenu) TypedShortcut(shortcut fyne.Shortcut) {
	if runtime.GOOS == goos.Windows && p.selectOwner != nil && isSelectDisclosureShortcut(shortcut) {
		p.TriggerLast()
		return
	}
	if c, ok := p.canvas.(fyne.Shortcutable); ok {
		c.TypedShortcut(shortcut)
	}
}

func (p *PopUpMenu) typedSelectKey(event *fyne.KeyEvent) bool {
	if runtime.GOOS != goos.Windows || p.selectOwner == nil {
		return false
	}
	switch event.Name {
	case fyne.KeyTab:
		var modifiers fyne.KeyModifier
		if d, ok := fyne.CurrentApp().Driver().(desktop.Driver); ok {
			modifiers = d.CurrentKeyModifiers()
		}
		p.commitSelectAndMoveFocus(modifiers&fyne.KeyModifierShift != 0)
	case fyne.KeyLeft:
		p.ActivatePrevious()
	case fyne.KeyRight:
		p.ActivateNext()
	default:
		return false
	}
	return true
}

func (p *PopUpMenu) commitSelectAndMoveFocus(backwards bool) {
	owner := p.selectOwner
	if owner == nil {
		return
	}
	p.TriggerLast() // Dismiss restores the owning canvas focus manager first.
	// A callback may open a dialog, remove the owner or explicitly move focus.
	// In those cases its new input scope must take precedence over Tab traversal.
	if p.canvas.Focused() != owner {
		return
	}
	if backwards {
		p.canvas.FocusPrevious()
	} else {
		p.canvas.FocusNext()
	}
}
