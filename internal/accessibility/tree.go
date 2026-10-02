// Package accessibility builds platform-independent semantic snapshots.
// Build and Perform must be called on the Fyne event thread. Adapters copy Node
// values to their own synchronized snapshots; they must not call widget methods
// from platform query threads.
package accessibility

import (
	"math"
	"reflect"
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
	_ // reserved for native text commands
	Select
	AddToSelection
	RemoveFromSelection
	Expand
	Collapse
	ScrollIntoView
)

// Root can retain identity for background content without exposing it. This is
// used for overlays, whose input scope must also be the accessibility scope.
type Root struct {
	Object     fyne.CanvasObject
	Suppressed bool
	// Scope exposes only this object and its subtree, retaining ancestor metadata.
	Scope fyne.CanvasObject
}

// Node is a value-only snapshot. ID zero denotes the platform window root.
type Node struct {
	ID, Parent                                      uint32
	Name, Description                               string
	Role                                            fyne.AccessibleRole
	Position                                        fyne.Position
	Size                                            fyne.Size
	BoundsPosition                                  fyne.Position
	BoundsSize                                      fyne.Size
	Disabled, Focusable, Focused, Required, Invalid bool
	Invoke, Toggle, Value, Range                    bool
	Checked, ReadOnly, Protected                    bool
	Text                                            string
	Document                                        *fyne.AccessibilityTextInfo
	Number, Min, Max, Step                          float64
	Selection, Multiple, SelectionRequired          bool
	Selectable, Selected, Expandable, Expanded      bool
	SelectionOwner                                  uint32
	SetPosition, SetSize                            int
	Level                                           int
	ScrollItem                                      bool
}

type Tree struct {
	next     uint32
	ids      map[fyne.CanvasObject]uint32
	objects  map[uint32]fyne.CanvasObject
	nodes    map[uint32]Node
	children map[fyne.CanvasObject][]fyne.CanvasObject
	elements map[fyne.CanvasObject]map[string]uint32
	Issues   []Issue
}

// Issue describes an authoring problem detected while building the tree.
// Messages contain no widget text or values.
type Issue struct {
	Object        fyne.CanvasObject
	Code, Message string
}

// Build produces a preorder snapshot, preserving IDs for still attached objects.
// Transparent widgets use renderer children; accessible leaves remain boundaries.
func (t *Tree) Build(roots []Root, focused fyne.Focusable) []Node {
	if t.ids == nil {
		t.ids = make(map[fyne.CanvasObject]uint32)
		t.children = make(map[fyne.CanvasObject][]fyne.CanvasObject)
		t.elements = make(map[fyne.CanvasObject]map[string]uint32)
	}
	seen := make(map[fyne.CanvasObject]bool)
	t.objects = make(map[uint32]fyne.CanvasObject)
	t.nodes = make(map[uint32]Node)
	t.Issues = nil
	var out []Node
	var visit func(fyne.CanvasObject, fyne.Position, uint32, bool, fyne.CanvasObject, []fyne.AccessibleChildDescriber, *bounds)
	visit = func(obj fyne.CanvasObject, pos fyne.Position, parent uint32, hidden bool, scope fyne.CanvasObject, describers []fyne.AccessibleChildDescriber, clip *bounds) {
		if !validObject(obj) || seen[obj] {
			return
		}
		seen[obj] = true // also guards malformed child cycles
		mode := compositionMode(obj)
		hidden = hidden || !obj.Visible() || mode == fyne.AccessibilityExclude
		pos = pos.Add(obj.Position())
		if hidden {
			// Preserve IDs while an attached subtree is hidden/suppressed. Do not
			// construct hidden renderers or ask hidden controls for live semantics.
			for _, child := range t.children[obj] {
				visit(child, pos, parent, true, scope, describers, clip)
			}
			return
		}
		if obj == scope {
			scope = nil
		}
		_, accessible := obj.(fyne.Accessible)
		if !accessible && mode == fyne.AccessibilityAuto && scope == nil {
			t.checkUnrepresented(obj)
		}
		if (accessible || mode == fyne.AccessibilityGroup) && mode != fyne.AccessibilityTransparent && scope == nil {
			n := t.snapshotNode(obj, pos, parent, describers, clip)
			out = append(out, n)
			parent = n.ID
		}
		if d, ok := obj.(fyne.AccessibleChildDescriber); ok {
			describers = append(append([]fyne.AccessibleChildDescriber(nil), describers...), d)
		}
		if elements, ok := obj.(fyne.AccessibleElements); ok && mode != fyne.AccessibilitySingle {
			out = append(out, t.snapshotElements(obj, elements, pos, parent, scope, describers, clip)...)
			delete(t.children, obj)
			return
		}
		delete(t.elements, obj)
		children := semanticChildren(obj, mode)
		clear(t.children[obj])
		t.children[obj] = append(t.children[obj][:0], children...)
		clip = childClip(obj, pos, clip)
		for _, c := range children {
			visit(c, pos, parent, false, scope, describers, clip)
		}
	}
	for _, root := range roots {
		visit(root.Object, fyne.Position{}, 0, root.Suppressed, root.Scope, nil, nil)
	}
	for obj := range t.children {
		if !seen[obj] {
			delete(t.children, obj)
		}
	}
	for obj := range t.ids {
		if !seen[obj] {
			delete(t.ids, obj)
		}
	}
	for obj := range t.elements {
		if !seen[obj] {
			delete(t.elements, obj)
		}
	}
	t.resolveRelations(out, focused)
	if focused != nil && t.FocusedID(focused) == 0 {
		obj, _ := focused.(fyne.CanvasObject)
		t.Issues = append(t.Issues, Issue{obj, "unrepresented-focus", "Keyboard focus has no exposed semantic node or active descendant."})
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
	if info.NameSet || info.Name != "" {
		n.Name = info.Name
	}
	if info.DescriptionSet || info.Description != "" {
		n.Description = info.Description
	}
	if info.RequiredSet || info.Required {
		n.Required = info.Required
	}
	if info.InvalidSet || info.Invalid {
		n.Invalid = info.Invalid
	}
}

// FocusedID resolves the real canvas focus without creating a second focus model.
func (t *Tree) FocusedID(focused fyne.Focusable) uint32 {
	if delegate, ok := focused.(fyne.AccessibleActiveElement); ok {
		owner, _ := focused.(fyne.CanvasObject)
		if key := delegate.AccessibilityActiveElement(); key != "" {
			return t.liveID(t.elements[owner][key])
		}
		return t.liveID(t.ids[owner])
	}
	var target fyne.CanvasObject
	if delegate, ok := focused.(fyne.AccessibleActiveDescendant); ok {
		target = delegate.AccessibilityActiveDescendant()
	}
	if target == nil {
		target, _ = focused.(fyne.CanvasObject)
	}
	if id := t.ids[target]; id != 0 {
		if _, live := t.objects[id]; live {
			return id
		}
	}
	return 0
}

// NodeForObject returns the currently exposed node for an original object.
func (t *Tree) NodeForObject(obj fyne.CanvasObject) (Node, bool) {
	if obj == nil || !reflect.TypeOf(obj).Comparable() {
		return Node{}, false
	}
	n, ok := t.nodes[t.ids[obj]]
	return n, ok
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
	if n.Disabled {
		return false
	}
	if d, ok := obj.(fyne.Disableable); ok && d.Disabled() {
		return false
	}
	switch action {
	case Focus:
		if f, ok := obj.(fyne.AccessibleFocusHandler); ok {
			return f.AccessibilityFocusable() && f.AccessibilityFocus()
		}
		if f, ok := obj.(fyne.Focusable); ok && canvas != nil {
			canvas.Focus(f)
			return canvas.Focused() == f
		}
	case Select, AddToSelection, RemoveFromSelection:
		return t.performSelection(n, obj, action)
	case Expand, Collapse:
		return performExpansion(obj, action == Expand)
	case ScrollIntoView:
		if s, ok := obj.(fyne.AccessibleScrollItem); ok {
			return s.AccessibilityScrollIntoView()
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
