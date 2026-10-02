package widget

import "fyne.io/fyne/v2"

// AccessibilityLabel returns an empty fallback; name groups explicitly or in a Form.
//
// Since: 2.9
func (*RadioGroup) AccessibilityLabel() string { return "" }

// AccessibilityRole returns the grouping role.
//
// Since: 2.9
func (*RadioGroup) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleContainer }

// AccessibilitySelection describes the single selection and Required policy.
//
// Since: 2.9
func (r *RadioGroup) AccessibilitySelection() (multiple, required bool) { return false, r.Required }

// AccessibilityChildren exposes the actual keyboard-focusable radio items.
//
// Since: 2.9
func (r *RadioGroup) AccessibilityChildren() []fyne.CanvasObject {
	r.syncItems()
	return append([]fyne.CanvasObject(nil), r.items...)
}

// Keep items with the same label and occurrence alive across reorder/renderer
// replacement. A removed or renamed option never becomes a different live item.
func (r *RadioGroup) syncItems() {
	old := make(map[string][]*radioItem)
	for _, obj := range r.items {
		item, ok := obj.(*radioItem)
		if !ok {
			continue
		}
		old[item.Label] = append(old[item.Label], item)
	}
	items := make([]fyne.CanvasObject, len(r.Options))
	for idx, label := range r.Options {
		var item *radioItem
		if previous := old[label]; len(previous) != 0 {
			item, old[label] = previous[0], previous[1:]
		} else {
			item = newRadioItem(label, nil)
			item.group = r
			item.onTap = func(tapped *radioItem) { r.itemTapped(tapped, tapped.index) }
		}
		item.index = idx
		items[idx] = item
	}
	r.items = items
}

func (i *radioItem) AccessibilityLabel() string           { return i.Label }
func (*radioItem) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleRadio }
func (i *radioItem) AccessibilitySelectionItem() (owner fyne.CanvasObject, selected bool, position, count int) {
	if i.group == nil {
		return nil, i.Selected, 0, 0
	}
	return i.group, i.group.selectedIndex() == i.index, i.index + 1, len(i.group.Options)
}

func (i *radioItem) AccessibilitySelect(mode fyne.AccessibilitySelectionMode) bool {
	r := i.group
	if r == nil || r.Disabled() || i.Disabled() || i.index >= len(r.items) || r.items[i.index] != i {
		return false
	}
	selected := r.selectedIndex() == i.index
	if mode == fyne.AccessibilitySelectRemove {
		if selected && r.Required {
			return false
		}
		if selected {
			r.itemTapped(i, i.index)
		}
		return true
	}
	if mode == fyne.AccessibilitySelectAdd && r.selectedIndex() >= 0 && !selected {
		return false
	}
	if !selected {
		r.itemTapped(i, i.index)
	}
	return true
}
