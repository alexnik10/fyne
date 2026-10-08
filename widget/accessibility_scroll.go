package widget

import "fyne.io/fyne/v2"

// AccessibilityScroll describes the list's current vertical viewport.
//
// Since: 2.9
func (l *List) AccessibilityScroll() fyne.AccessibilityScrollInfo {
	return l.scroller.AccessibilityScroll()
}

// AccessibilityScrollTo scrolls without changing selection or keyboard focus.
//
// Since: 2.9
func (l *List) AccessibilityScrollTo(offset fyne.Position) bool {
	return l.scroller.AccessibilityScrollTo(offset)
}

// AccessibilityScroll describes the tree's current viewport.
//
// Since: 2.9
func (t *Tree) AccessibilityScroll() fyne.AccessibilityScrollInfo {
	return t.scroller.AccessibilityScroll()
}

// AccessibilityScrollTo scrolls without changing selection or keyboard focus.
//
// Since: 2.9
func (t *Tree) AccessibilityScrollTo(offset fyne.Position) bool {
	return t.scroller.AccessibilityScrollTo(offset)
}

// AccessibilityScroll describes the table's data viewport, excluding headers.
//
// Since: 2.9
func (t *Table) AccessibilityScroll() fyne.AccessibilityScrollInfo {
	info := t.content.AccessibilityScroll()
	if t.content != nil {
		info.ViewportPosition = t.content.Position()
	}
	return info
}

// AccessibilityScrollTo scrolls without changing selection or keyboard focus.
//
// Since: 2.9
func (t *Table) AccessibilityScrollTo(offset fyne.Position) bool {
	return t.content.AccessibilityScrollTo(offset)
}
