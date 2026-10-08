package test

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/internal/accessibility"
)

// AccessibilityNode is a platform-independent semantic snapshot. Position and
// Size describe layout; BoundsPosition and BoundsSize include ancestor clipping.
// ID zero is reserved for the platform window. IDs belong to one test tree.
//
// Since: 2.9
type AccessibilityNode struct {
	ID, Parent                                       uint32
	Name, Description                                string
	Shortcut                                         string
	LiveSetting                                      fyne.AccessibilityLiveSetting
	LiveRevision                                     uint64
	Role                                             fyne.AccessibleRole
	Position                                         fyne.Position
	Size                                             fyne.Size
	BoundsPosition                                   fyne.Position
	BoundsSize                                       fyne.Size
	Disabled, Focusable, Focused, Required, Invalid  bool
	Invoke, Toggle, Value, Range                     bool
	Checked, ReadOnly, Protected                     bool
	Text                                             string
	Document                                         *fyne.AccessibilityTextInfo
	Number, Min, Max, Step                           float64
	Selection, Multiple, SelectionRequired           bool
	Selectable, Selected, Expandable, Expanded       bool
	SelectionOwner                                   uint32
	SetPosition, SetSize                             int
	Level                                            int
	ScrollItem                                       bool
	Scroll                                           bool
	HorizontalScrollPercent, VerticalScrollPercent   float64
	HorizontalViewSize, VerticalViewSize             float64
	ItemContainer, VirtualizedItem                   bool
	Grid, GridItem, Table, RowHeaders, ColumnHeaders bool
	GridOwner                                        uint32
	Rows, Columns, Row, Column, RowSpan, ColumnSpan  int
}

// AccessibilityAction is a command supported by a semantic node.
//
// Since: 2.9
type AccessibilityAction uint8

// Semantic commands use the original widget behavior, without pointer simulation.
const (
	AccessibilityFocus AccessibilityAction = iota
	AccessibilityActivate
	AccessibilityToggle
	AccessibilitySetValue
	AccessibilitySetRangeValue
	_ // Text commands use SelectText and ScrollText.
	AccessibilitySelect
	AccessibilityAddToSelection
	AccessibilityRemoveFromSelection
	AccessibilityExpand
	AccessibilityCollapse
	AccessibilityScrollIntoView
)

// AccessibilityTree inspects the same semantic tree used by native adapters.
// Keep one instance to compare IDs across refreshes and reordering. Like other
// test helpers, call it on the Fyne event thread; it is not concurrency-safe.
// Creating or reading it does not move keyboard focus.
//
// Since: 2.9
type AccessibilityTree struct {
	canvas fyne.Canvas
	tree   accessibility.Tree
}

// AccessibilityIssue describes an authoring problem. Code is a stable diagnostic
// identifier; Object identifies the original object. Message contains no UI text.
//
// Since: 2.9
type AccessibilityIssue struct {
	Object        fyne.CanvasObject
	Code, Message string
}

// Issues rebuilds the tree and reports missing semantics, ignored metadata and
// keyboard focus without an exposed semantic target. An empty result does not
// replace platform and screen-reader testing.
func (t *AccessibilityTree) Issues() []AccessibilityIssue {
	t.Snapshot()
	out := make([]AccessibilityIssue, len(t.tree.Issues))
	for i, issue := range t.tree.Issues {
		out[i] = AccessibilityIssue(issue)
	}
	return out
}

// NewAccessibilityTree creates an inspector for a laid-out test canvas.
//
// Since: 2.9
func NewAccessibilityTree(canvas fyne.Canvas) *AccessibilityTree {
	return &AccessibilityTree{canvas: canvas}
}

// Snapshot rebuilds the current tree, including visibility and modal scope.
func (t *AccessibilityTree) Snapshot() []AccessibilityNode {
	if t.canvas == nil {
		return nil
	}
	nodes := t.tree.Build(accessibility.Roots(t.canvas), t.canvas.Focused())
	out := make([]AccessibilityNode, len(nodes))
	for i, n := range nodes {
		out[i] = AccessibilityNode(n)
	}
	return out
}

// Node returns the current semantics of an original widget or logical child.
func (t *AccessibilityTree) Node(object fyne.CanvasObject) (AccessibilityNode, bool) {
	t.Snapshot()
	n, ok := t.tree.NodeForObject(object)
	return AccessibilityNode(n), ok
}

// Element returns current semantics by stable owner and key, without needing the
// transient semantic object or a renderer cell. Hidden elements are not returned.
func (t *AccessibilityTree) Element(owner fyne.CanvasObject, key string) (AccessibilityNode, bool) {
	t.Snapshot()
	if t.tree.RequestElement(owner, key) {
		t.Snapshot()
	}
	n, ok := t.tree.NodeForElement(owner, key)
	return AccessibilityNode(n), ok
}

// Perform dispatches a command after rebuilding the tree, rejecting stale,
// hidden, excluded, disabled and out-of-scope nodes. Value and number are used
// only by SetValue and SetRangeValue. True means the command was dispatched.
func (t *AccessibilityTree) Perform(id uint32, action AccessibilityAction, value string, number float64) bool {
	t.Snapshot()
	if t.tree.Realize(id) {
		t.Snapshot()
	}
	return t.tree.Perform(id, accessibility.Action(action), value, number, t.canvas)
}

// SelectText changes the selection using rune offsets, after revalidation.
func (t *AccessibilityTree) SelectText(id uint32, start, end int) bool {
	t.Snapshot()
	return t.tree.SelectText(id, start, end)
}

// ScrollText reveals a rune range without moving keyboard focus.
func (t *AccessibilityTree) ScrollText(id uint32, start, end int, alignTop bool) bool {
	t.Snapshot()
	return t.tree.ScrollText(id, start, end, alignTop)
}

// Scroll changes the viewport by small increments or pages, after revalidation.
func (t *AccessibilityTree) Scroll(id uint32, horizontal, vertical fyne.AccessibilityScrollAmount) bool {
	t.Snapshot()
	return t.tree.Scroll(id, horizontal, vertical)
}

// SetScrollPercent changes either axis in [0,100]. -1 leaves an axis unchanged.
// Invalid or unsupported requested axes reject the entire command.
func (t *AccessibilityTree) SetScrollPercent(id uint32, horizontal, vertical float64) bool {
	t.Snapshot()
	return t.tree.SetScrollPercent(id, horizontal, vertical)
}
