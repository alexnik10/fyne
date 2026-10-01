// Package accessibility builds platform-independent semantic snapshots.
// Build and Perform must be called on the Fyne event thread. Adapters copy Node
// values to their own synchronized snapshots; they must not call widget methods
// from platform query threads.
package accessibility

import (
	"math"
	"reflect"
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"
)

type Action int

const (
	Focus Action = iota
	Activate
	Toggle
	SetValue
	SetRangeValue
)

// Root can retain identity for background content without exposing it. This is
// used for overlays, whose input scope must also be the accessibility scope.
type Root struct {
	Object     fyne.CanvasObject
	Suppressed bool
}

// Node is a value-only snapshot. ID zero denotes the platform window root.
type Node struct {
	ID, Parent                                      uint32
	Name, Description                               string
	Role                                            fyne.AccessibleRole
	Position                                        fyne.Position
	Size                                            fyne.Size
	Disabled, Focusable, Focused, Required, Invalid bool
	Invoke, Toggle, Value, Range                    bool
	Checked, ReadOnly, Protected                    bool
	Text                                            string
	Document                                        *fyne.AccessibilityTextInfo
	Number, Min, Max, Step                          float64
}

type Tree struct {
	next    uint32
	ids     map[fyne.CanvasObject]uint32
	objects map[uint32]fyne.CanvasObject
	nodes   map[uint32]Node
}

// Build produces a preorder snapshot, preserving IDs for still attached objects.
// Renderer internals are deliberately not a fallback for semantic children.
func (t *Tree) Build(roots []Root, focused fyne.Focusable) []Node {
	if t.ids == nil {
		t.ids = make(map[fyne.CanvasObject]uint32)
	}
	seen := make(map[fyne.CanvasObject]bool)
	t.objects = make(map[uint32]fyne.CanvasObject)
	t.nodes = make(map[uint32]Node)
	var out []Node
	var visit func(fyne.CanvasObject, fyne.Position, uint32, bool, []fyne.AccessibleChildDescriber)
	visit = func(obj fyne.CanvasObject, pos fyne.Position, parent uint32, hidden bool, describers []fyne.AccessibleChildDescriber) {
		if obj == nil || !reflect.TypeOf(obj).Comparable() || seen[obj] {
			return
		}
		v := reflect.ValueOf(obj)
		if v.Kind() == reflect.Pointer && v.IsNil() {
			return
		}
		seen[obj] = true // also guards malformed child cycles
		hidden = hidden || !obj.Visible()
		pos = pos.Add(obj.Position())
		if a, ok := obj.(fyne.Accessible); ok && !hidden {
			id := t.ids[obj]
			if id == 0 {
				t.next++
				id = t.next
				t.ids[obj] = id
			}
			n := Node{ID: id, Parent: parent, Name: a.AccessibilityLabel(), Role: a.AccessibilityRole(), Position: pos, Size: obj.Size()}
			// Layout-only containers do not need to be spoken as "Container".
			if _, plain := obj.(*fyne.Container); plain {
				n.Name = ""
			}
			for _, d := range describers {
				applyInfo(&n, d.AccessibilityChildInfo(obj))
			}
			if d, ok := obj.(fyne.AccessibleDescribed); ok {
				applyInfo(&n, d.AccessibilityInfo())
			}
			if d, ok := obj.(fyne.Disableable); ok {
				n.Disabled = d.Disabled()
			}
			if f, ok := obj.(fyne.Focusable); ok {
				n.Focusable = !n.Disabled
				n.Focused = f == focused
			}
			_, n.Invoke = obj.(fyne.AccessibleActionable)
			if c, ok := obj.(fyne.AccessibleToggler); ok {
				n.Toggle = true
				n.Checked = c.AccessibilityChecked()
			}
			if value, ok := obj.(fyne.AccessibleValue); ok {
				n.Value = true
				n.Text, n.ReadOnly, n.Protected = value.AccessibilityValue()
				if n.Protected {
					n.Text = ""
				}
			}
			if r, ok := obj.(fyne.AccessibleRange); ok {
				n.Range = true
				n.Number, n.Min, n.Max, n.Step = r.AccessibilityRange()
			}
			if text, ok := obj.(fyne.AccessibleText); ok {
				document := text.AccessibilityText()
				length := utf8.RuneCountInString(document.Text)
				if n.Protected {
					// Defend against a custom control accidentally returning clear text.
					document.Text = strings.Repeat("•", length)
					document.WordBoundaries = []int{0}
					if length != 0 {
						document.WordBoundaries = append(document.WordBoundaries, length)
					}
				}
				document.Caret = min(max(document.Caret, 0), length)
				document.SelectionStart = min(max(document.SelectionStart, 0), length)
				document.SelectionEnd = min(max(document.SelectionEnd, document.SelectionStart), length)
				document.Positions = append([]fyne.AccessibilityTextPosition(nil), document.Positions...)
				document.WordBoundaries = append([]int(nil), document.WordBoundaries...)
				n.Document = &document
			}
			n.ReadOnly = n.ReadOnly || n.Disabled
			out = append(out, n)
			t.nodes[id] = n
			t.objects[id] = obj
			parent = id
		}
		if d, ok := obj.(fyne.AccessibleChildDescriber); ok {
			describers = append(append([]fyne.AccessibleChildDescriber(nil), describers...), d)
		}
		var children []fyne.CanvasObject
		if c, ok := obj.(fyne.AccessibleChildren); ok {
			children = c.AccessibilityChildren()
		} else if c, ok := obj.(*fyne.Container); ok {
			children = c.Objects
		}
		for _, c := range children {
			visit(c, pos, parent, hidden, describers)
		}
	}
	for _, root := range roots {
		visit(root.Object, fyne.Position{}, 0, root.Suppressed, nil)
	}
	for obj := range t.ids {
		if !seen[obj] {
			delete(t.ids, obj)
		}
	}
	return out
}

// SelectText revalidates a text command against the current input scope.
func (t *Tree) SelectText(id uint32, start, end int) bool {
	n, ok := t.nodes[id]
	if !ok || n.Disabled || n.Document == nil || start < 0 || end < start || end > utf8.RuneCountInString(n.Document.Text) {
		return false
	}
	if text, ok := t.objects[id].(fyne.AccessibleText); ok {
		text.AccessibilitySelectText(start, end)
		return true
	}
	return false
}

// ScrollText reveals a range without changing the canvas focus or selection.
func (t *Tree) ScrollText(id uint32, start, end int, alignTop bool) bool {
	n, ok := t.nodes[id]
	if !ok || n.Document == nil || start < 0 || end < start || end > utf8.RuneCountInString(n.Document.Text) {
		return false
	}
	if s, ok := t.objects[id].(fyne.AccessibleTextScroller); ok {
		s.AccessibilityScrollText(start, end, alignTop)
		return true
	}
	return false
}

func applyInfo(n *Node, info fyne.AccessibilityInfo) {
	if info.Name != "" {
		n.Name = info.Name
	}
	if info.Description != "" {
		n.Description = info.Description
	}
	n.Required = n.Required || info.Required
	n.Invalid = n.Invalid || info.Invalid
}

// FocusedID resolves the real canvas focus without creating a second focus model.
func (t *Tree) FocusedID(focused fyne.Focusable) uint32 {
	for id, obj := range t.objects {
		if f, ok := obj.(fyne.Focusable); ok && f == focused {
			return id
		}
	}
	return 0
}

// Perform revalidates an action against the latest tree. Rebuild immediately
// before calling so hidden, detached and background controls cannot be invoked.
// A true result means the command was dispatched, not that validation succeeded.
func (t *Tree) Perform(id uint32, action Action, text string, number float64, canvas fyne.Canvas) bool {
	obj, ok := t.objects[id]
	if !ok || !obj.Visible() {
		return false
	}
	n := t.nodes[id]
	if d, ok := obj.(fyne.Disableable); ok && d.Disabled() {
		return false
	}
	switch action {
	case Focus:
		if f, ok := obj.(fyne.Focusable); ok && canvas != nil {
			canvas.Focus(f)
			return canvas.Focused() == f
		}
	case Activate:
		if a, ok := obj.(fyne.AccessibleActionable); ok {
			a.AccessibilityActivate()
			return true
		}
	case Toggle:
		if a, ok := obj.(fyne.AccessibleToggler); ok {
			a.AccessibilityToggle()
			return true
		}
	case SetValue:
		if a, ok := obj.(fyne.AccessibleValue); ok && !n.ReadOnly {
			a.AccessibilitySetValue(text)
			return true
		}
	case SetRangeValue:
		if a, ok := obj.(fyne.AccessibleRange); ok && !n.ReadOnly && !math.IsNaN(number) && !math.IsInf(number, 0) && number >= n.Min && number <= n.Max {
			a.AccessibilitySetRangeValue(number)
			return true
		}
	}
	return false
}
