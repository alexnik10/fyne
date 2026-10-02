package app

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/internal/driver"
)

// FocusManager represents a standard manager of input focus for a canvas
type FocusManager struct {
	content fyne.CanvasObject
	focused fyne.Focusable
}

// NewFocusManager returns a new instance of the standard focus manager for a canvas.
func NewFocusManager(c fyne.CanvasObject) *FocusManager {
	return &FocusManager{content: c}
}

// Focus focuses the given obj.
func (f *FocusManager) Focus(obj fyne.Focusable) bool {
	if obj != nil {
		var hiddenAncestor fyne.CanvasObject
		hidden := false
		found := driver.WalkCompleteObjectTree(
			f.content,
			func(object fyne.CanvasObject, _, _ fyne.Position, _ fyne.Size) bool {
				if hiddenAncestor == nil && !object.Visible() {
					hiddenAncestor = object
				}
				if fo, _ := obj.(fyne.CanvasObject); object == fo {
					hidden = hiddenAncestor != nil
					return true
				}
				return false
			},
			func(object fyne.CanvasObject, _ fyne.Position, _ fyne.CanvasObject) {
				if hiddenAncestor == object {
					hiddenAncestor = nil
				}
			},
		)
		if !found {
			return false
		}
		if hidden {
			return true
		}
		if dis, ok := obj.(fyne.Disableable); ok && dis.Disabled() {
			type selectableText interface {
				SelectedText() string
			}
			if _, isSelectableText := obj.(selectableText); !isSelectableText || fyne.CurrentDevice().IsMobile() {
				return true
			}
		}
	}
	f.switchFocusTo(obj)
	return true
}

// Focused returns the currently focused object or nil if none.
func (f *FocusManager) Focused() fyne.Focusable {
	return f.focused
}

// FocusGained signals to the manager that its content got focus (due to window/overlay switch for instance).
func (f *FocusManager) FocusGained() {
	if focused := f.Focused(); focused != nil {
		focused.FocusGained()
	}
}

// FocusLost signals to the manager that its content lost focus (due to window/overlay switch for instance).
func (f *FocusManager) FocusLost() {
	if focused := f.Focused(); focused != nil {
		focused.FocusLost()
	}
}

// FocusNext will find the item after the current that can be focused and focus it.
// If current is nil then the first focusable item in the canvas will be focused.
func (f *FocusManager) FocusNext() {
	f.switchFocusTo(f.nextInChain(f.focused))
}

// FocusPrevious will find the item before the current that can be focused and focus it.
// If current is nil then the last focusable item in the canvas will be focused.
func (f *FocusManager) FocusPrevious() {
	f.switchFocusTo(f.previousInChain(f.focused))
}

func (f *FocusManager) nextInChain(current fyne.Focusable) fyne.Focusable {
	var next fyne.Focusable
	found := current == nil // if we have no starting point then pretend we matched already
	driver.WalkVisibleObjectTree(f.content, func(obj fyne.CanvasObject, _ fyne.Position, _ fyne.Position, _ fyne.Size) bool {
		if w, ok := obj.(fyne.Disableable); ok && w.Disabled() {
			// disabled widget cannot receive focus
			return false
		}

		focus, ok := obj.(fyne.Focusable)
		if !ok {
			return false
		}

		if found {
			next = focus
			return true
		}
		if next == nil {
			next = focus
		}

		if co, _ := current.(fyne.CanvasObject); obj == co {
			found = true
		}

		return false
	}, nil)

	return next
}

func (f *FocusManager) previousInChain(current fyne.Focusable) fyne.Focusable {
	// Find the predecessor in the forward chain. Reversing sibling order still
	// visits a focusable parent before its children, so it cannot reverse Tab.
	var previous fyne.Focusable
	driver.WalkVisibleObjectTree(f.content, func(obj fyne.CanvasObject, _ fyne.Position, _ fyne.Position, _ fyne.Size) bool {
		if w, ok := obj.(fyne.Disableable); ok && w.Disabled() {
			return false
		}
		focus, ok := obj.(fyne.Focusable)
		if !ok {
			return false
		}
		if co, _ := current.(fyne.CanvasObject); obj == co && previous != nil {
			return true
		}
		previous = focus
		return false
	}, nil)

	// With no predecessor (or no current focus), wrap to the last item.
	return previous
}

func (f *FocusManager) switchFocusTo(obj fyne.Focusable) {
	if f.focused == obj {
		return
	}

	if f.focused != nil {
		f.focused.FocusLost()
	}
	f.focused = obj
	if obj != nil {
		obj.FocusGained()
	}
}
