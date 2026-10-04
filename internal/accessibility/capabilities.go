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
	populateScroll(n, obj)
	if menu, ok := obj.(fyne.AccessibleMenuItem); ok {
		submenu, checkable := menu.AccessibilityMenuItem()
		n.Invoke = n.Invoke && !submenu
		n.Toggle = n.Toggle && checkable && !submenu
		n.Expandable = n.Expandable && submenu
	}
	if shortcut, ok := obj.(fyne.AccessibleShortcut); ok {
		n.Shortcut = shortcut.AccessibilityShortcut()
	}
	n.ReadOnly = n.ReadOnly || n.Disabled
	if g, ok := obj.(fyne.AccessibleGrid); ok {
		n.Grid = true
		n.Rows, n.Columns = g.AccessibilityGrid()
	}
	if table, ok := obj.(fyne.AccessibleTable); ok {
		n.Table = true
		n.RowHeaders, n.ColumnHeaders = table.AccessibilityTable()
	}
	if cell, ok := obj.(fyne.AccessibleGridItem); ok {
		n.GridItem = true
		_, n.Row, n.Column, n.RowSpan, n.ColumnSpan = cell.AccessibilityGridItem()
	}
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
		n.ItemContainer = t.nodes[n.ID].ItemContainer
		if item, ok := t.objects[n.ID].(fyne.AccessibleSelectable); ok {
			owner, _, _, _ := item.AccessibilitySelectionItem()
			n.SelectionOwner = t.ids[owner]
			if parent, exists := t.nodes[n.SelectionOwner]; !exists || !parent.Selection {
				n.Selectable = false
				n.SelectionOwner = 0
			}
		}
		n.Focused = n.ID == focusedID
		n.ScrollItem = n.ScrollItem || (!isSemanticPopup(t.objects[n.ID]) && t.scrollAncestor(n.Parent) != 0)
		if cell, ok := t.objects[n.ID].(fyne.AccessibleGridItem); ok {
			owner, _, _, _, _ := cell.AccessibilityGridItem()
			n.GridOwner = t.ids[owner]
			grid, live := t.nodes[n.GridOwner]
			n.GridItem = live && grid.Grid && n.Row >= 0 && n.Column >= 0 && n.Row < grid.Rows && n.Column < grid.Columns && n.RowSpan > 0 && n.ColumnSpan > 0
			n.Table = n.GridItem && grid.Table
		}
		t.nodes[n.ID] = *n
	}
}

func isSemanticPopup(obj fyne.CanvasObject) bool {
	popup, ok := obj.(fyne.AccessiblePopup)
	return ok && popup.AccessibilityPopup()
}

func performActivation(n Node, obj fyne.CanvasObject, action Action) bool {
	if action == Activate && n.Invoke {
		if a, ok := obj.(fyne.AccessibleActionable); ok {
			a.AccessibilityActivate()
			return true
		}
	}
	if action == Toggle && n.Toggle {
		if a, ok := obj.(fyne.AccessibleToggler); ok {
			a.AccessibilityToggle()
			return true
		}
	}
	return false
}

func (t *Tree) checkFocus(focused fyne.Focusable) {
	if focused != nil && t.FocusedID(focused) == 0 {
		obj, _ := focused.(fyne.CanvasObject)
		t.Issues = append(t.Issues, Issue{obj, "unrepresented-focus", "Keyboard focus has no exposed semantic node or active descendant."})
	}
}
