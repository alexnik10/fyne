package accessibility

import (
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"
)

func populateCapabilities(n *Node, obj fyne.CanvasObject) {
	if d, ok := obj.(fyne.Disableable); ok {
		n.Disabled = d.Disabled()
	}
	_, n.Focusable = obj.(fyne.Focusable)
	if f, ok := obj.(fyne.AccessibleFocusHandler); ok {
		n.Focusable = f.AccessibilityFocusable()
	}
	n.Focusable = n.Focusable && !n.Disabled
	_, n.Invoke = obj.(fyne.AccessibleActionable)
	if c, ok := obj.(fyne.AccessibleToggler); ok {
		n.Toggle, n.Checked = true, c.AccessibilityChecked()
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
		n.Document = copyDocument(text.AccessibilityText(), n.Protected)
	}
	if s, ok := obj.(fyne.AccessibleSelection); ok {
		n.Selection = true
		n.Multiple, n.SelectionRequired = s.AccessibilitySelection()
	}
	if s, ok := obj.(fyne.AccessibleSelectable); ok {
		n.Selectable = true
		_, n.Selected, n.SetPosition, n.SetSize = s.AccessibilitySelectionItem()
	}
	if e, ok := obj.(fyne.AccessibleExpandable); ok {
		n.Expandable, n.Expanded = true, e.AccessibilityExpanded()
	}
	if h, ok := obj.(fyne.AccessibleHierarchy); ok {
		n.Level, n.SetPosition, n.SetSize = h.AccessibilityHierarchy()
	}
	_, n.ScrollItem = obj.(fyne.AccessibleScrollItem)
	n.ReadOnly = n.ReadOnly || n.Disabled
}

func copyDocument(document fyne.AccessibilityTextInfo, protected bool) *fyne.AccessibilityTextInfo {
	length := utf8.RuneCountInString(document.Text)
	if protected {
		// Defend against custom controls accidentally returning clear text.
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
	return &document
}

func (t *Tree) performSelection(n Node, obj fyne.CanvasObject, action Action) bool {
	item, ok := obj.(fyne.AccessibleSelectable)
	owner, live := t.nodes[n.SelectionOwner]
	if !ok || !n.Selectable || !live || owner.Disabled {
		return false
	}
	mode := fyne.AccessibilitySelectReplace
	switch action {
	case AddToSelection:
		mode = fyne.AccessibilitySelectAdd
		if !owner.Multiple {
			for _, other := range t.nodes {
				if other.SelectionOwner == owner.ID && other.Selected && other.ID != n.ID {
					return false
				}
			}
		}
	case RemoveFromSelection:
		mode = fyne.AccessibilitySelectRemove
		if owner.SelectionRequired && n.Selected {
			remaining := false
			for _, other := range t.nodes {
				remaining = remaining || (other.SelectionOwner == owner.ID && other.Selected && other.ID != n.ID)
			}
			if !remaining {
				return false
			}
		}
	}
	return item.AccessibilitySelect(mode)
}

func performExpansion(obj fyne.CanvasObject, expanded bool) bool {
	if e, ok := obj.(fyne.AccessibleExpandable); ok {
		e.AccessibilitySetExpanded(expanded)
		return e.AccessibilityExpanded() == expanded
	}
	return false
}

func (t *Tree) resolveRelations(out []Node, focused fyne.Focusable) {
	focusedID := t.FocusedID(focused)
	for i := range out {
		n := &out[i]
		if item, ok := t.objects[n.ID].(fyne.AccessibleSelectable); ok {
			owner, _, _, _ := item.AccessibilitySelectionItem()
			n.SelectionOwner = t.ids[owner]
			if parent, exists := t.nodes[n.SelectionOwner]; !exists || !parent.Selection {
				n.Selectable = false
				n.SelectionOwner = 0
			}
		}
		n.Focused = n.ID == focusedID
		t.nodes[n.ID] = *n
	}
}
