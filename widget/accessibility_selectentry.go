package widget

import (
	"slices"

	"fyne.io/fyne/v2"
)

// AccessibilityRole identifies an editable combo box.
// Since: 2.9
func (*SelectEntry) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleComboBox }

// AccessibilitySelection permits one predefined choice or free text.
// Since: 2.9
func (*SelectEntry) AccessibilitySelection() (multiple, required bool) { return false, false }

// AccessibilityExpanded reports whether the choices are displayed.
// Since: 2.9
func (e *SelectEntry) AccessibilityExpanded() bool { return e.popUp != nil && e.popUp.Visible() }

// AccessibilitySetExpanded opens or dismisses the existing choice popup.
// Since: 2.9
func (e *SelectEntry) AccessibilitySetExpanded(expanded bool) { e.setOptionsExpanded(expanded, true) }

func (e *SelectEntry) setOptionsExpanded(expanded, focus bool) {
	if e.Disabled() || e.AccessibilityExpanded() == expanded {
		return
	}
	if !expanded {
		if e.popUp != nil {
			e.popUp.Dismiss()
		}
		return
	}
	c := fyne.CurrentApp().Driver().CanvasForObject(e.super())
	if c == nil || len(e.options) == 0 {
		return
	}
	e.syncAccessibleOptions()
	if focus {
		c.Focus(e)
	}
	e.popUp = NewPopUpMenu(e.dropDown, c)
	e.popUp.selectEntryOwner = e
	e.popUp.ShowAtPosition(e.popUpPos())
	e.popUp.Resize(fyne.NewSize(e.Size().Width, e.popUp.MinSize().Height))
	if focus {
		e.popUp.AccessibilityFocus()
	}
}

// AccessibilityChildren keeps option identity independent of popup menu renderers.
// Since: 2.9
func (e *SelectEntry) AccessibilityChildren() []fyne.CanvasObject {
	e.syncAccessibleOptions()
	children := e.Entry.AccessibilityChildren()
	for _, option := range e.accessibleOptions {
		children = append(children, option)
	}
	return children
}

// TypedShortcut opens choices using Alt+Up/Down while preserving editor shortcuts.
// Since: 2.9
func (e *SelectEntry) TypedShortcut(s fyne.Shortcut) {
	if isSelectDisclosureShortcut(s) {
		e.AccessibilitySetExpanded(!e.AccessibilityExpanded())
		return
	}
	e.Entry.TypedShortcut(s)
}
func (e *SelectEntry) accessibilitySelectedIndex() int { return slices.Index(e.options, e.Text) }
func (e *SelectEntry) syncAccessibleOptions() {
	old := make(map[string][]*selectEntryOption)
	for _, option := range e.accessibleOptions {
		old[option.label] = append(old[option.label], option)
	}
	next := make([]*selectEntryOption, len(e.options))
	for i, label := range e.options {
		item := &selectEntryOption{owner: e, label: label}
		if items := old[label]; len(items) > 0 {
			item, old[label] = items[0], items[1:]
		}
		item.index = i
		next[i] = item
	}
	e.accessibleOptions = next
}

type selectEntryOption struct {
	DisableableWidget
	owner *SelectEntry
	label string
	index int
}

func (o *selectEntryOption) AccessibilityLabel() string           { return o.label }
func (*selectEntryOption) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleListItem }
func (o *selectEntryOption) Disabled() bool                       { return o.owner.Disabled() }

func (o *selectEntryOption) AccessibilitySelectionItem() (owner fyne.CanvasObject, selected bool, position, count int) {
	return o.owner, o.owner.accessibilitySelectedIndex() == o.index, o.index + 1, len(o.owner.options)
}

func (o *selectEntryOption) AccessibilitySelect(mode fyne.AccessibilitySelectionMode) bool {
	s := o.owner
	if s.Disabled() || o.index >= len(s.accessibleOptions) || s.accessibleOptions[o.index] != o {
		return false
	}
	selected := s.accessibilitySelectedIndex() == o.index
	if mode == fyne.AccessibilitySelectAdd && s.accessibilitySelectedIndex() >= 0 && !selected {
		return false
	}
	if mode == fyne.AccessibilitySelectRemove {
		if selected {
			s.SetText("")
		}
		return true
	}
	if s.popUp != nil {
		s.popUp.Dismiss()
	}
	if !selected {
		s.SetText(o.label)
	}
	return true
}

func (o *selectEntryOption) AccessibilityFocusable() bool {
	return !o.Disabled() && o.owner.AccessibilityExpanded()
}

func (o *selectEntryOption) AccessibilityFocus() bool {
	if !o.AccessibilityFocusable() {
		return false
	}
	p := o.owner.popUp
	p.activateItem(p.Items[o.index].(*menuItem))
	p.revealSelectItem()
	p.canvas.Focus(p)
	return p.canvas.Focused() == p
}
func (o *selectEntryOption) Position() fyne.Position { pos, _ := o.bounds(); return pos }
func (o *selectEntryOption) Size() fyne.Size         { _, size := o.bounds(); return size }
func (o *selectEntryOption) bounds() (fyne.Position, fyne.Size) {
	s := o.owner
	if !s.AccessibilityExpanded() || o.index >= len(s.popUp.Items) {
		return fyne.Position{}, fyne.Size{}
	}
	d := fyne.CurrentApp().Driver()
	item := s.popUp.Items[o.index]
	pos, origin := d.AbsolutePositionForObject(item), d.AbsolutePositionForObject(s.super())
	popup := d.AbsolutePositionForObject(s.popUp)
	left, top := max(pos.X, popup.X), max(pos.Y, popup.Y)
	right := min(pos.X+item.Size().Width, popup.X+s.popUp.Size().Width)
	bottom := min(pos.Y+item.Size().Height, popup.Y+s.popUp.Size().Height)
	return fyne.NewPos(left-origin.X, top-origin.Y), fyne.NewSize(max(0, right-left), max(0, bottom-top))
}
