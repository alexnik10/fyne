package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/internal/cache"
)

// AccessibilityLabel returns the placeholder as a fallback for an unlabelled select.
//
// Since: 2.9
func (s *Select) AccessibilityLabel() string { return s.PlaceHolder }

// AccessibilityRole returns the non-editable combo box role.
//
// Since: 2.9
func (*Select) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleComboBox }

// AccessibilityValue returns the current choice as a read-only string.
//
// Since: 2.9
func (s *Select) AccessibilityValue() (value string, readOnly, protected bool) {
	return s.Selected, true, false
}

// AccessibilitySetValue does nothing: choices are changed through selection.
//
// Since: 2.9
func (*Select) AccessibilitySetValue(string) {}

// AccessibilitySelection describes a single selection which can be cleared.
//
// Since: 2.9
func (*Select) AccessibilitySelection() (multiple, required bool) { return false, false }

// AccessibilityExpanded reports whether the option popup is open.
//
// Since: 2.9
func (s *Select) AccessibilityExpanded() bool { return s.popUp != nil && s.popUp.Visible() }

// AccessibilitySetExpanded uses the normal popup and dismissal commands.
//
// Since: 2.9
func (s *Select) AccessibilitySetExpanded(expanded bool) {
	if s.Disabled() {
		return
	}
	if expanded {
		s.showPopUp()
		if s.popUp != nil {
			s.popUp.AccessibilityFocus()
		}
	} else if s.popUp != nil {
		s.popUp.Dismiss()
	}
}

// AccessibilityChildren exposes stable logical options, including the selected
// option while collapsed. Collapsed options have empty bounds and cannot focus.
//
// Since: 2.9
func (s *Select) AccessibilityChildren() []fyne.CanvasObject {
	s.syncAccessibleOptions()
	children := make([]fyne.CanvasObject, len(s.accessibleOptions))
	for i, option := range s.accessibleOptions {
		children[i] = option
	}
	return children
}

// AccessibilityFocusable reports whether keyboard focus can enter this select.
//
// Since: 2.9
func (s *Select) AccessibilityFocusable() bool { return !s.Disabled() }

// AccessibilityFocus focuses the real keyboard owner, including its open popup.
//
// Since: 2.9
func (s *Select) AccessibilityFocus() bool {
	c := fyne.CurrentApp().Driver().CanvasForObject(s.super())
	if c == nil || s.Disabled() {
		return false
	}
	var target fyne.Focusable = s
	if s.AccessibilityExpanded() {
		target = s.popUp
	}
	c.Focus(target)
	return c.Focused() == target
}

// Refresh updates the choice and invalidates an open popup when its options change.
func (s *Select) Refresh() {
	s.syncAccessibleOptions()
	s.BaseWidget.Refresh()
}

// Disable prevents changes and dismisses any open option popup.
func (s *Select) Disable() {
	if s.popUp != nil {
		s.popUp.Dismiss()
	}
	s.DisableableWidget.Disable()
}

func (s *Select) syncAccessibleOptions() {
	unchanged := len(s.Options) == len(s.accessibleOptions)
	for idx, item := range s.accessibleOptions {
		unchanged = unchanged && idx < len(s.Options) && s.Options[idx] == item.label
	}
	if unchanged {
		return
	}
	if s.popUp != nil {
		pop := s.popUp
		s.popUp = nil
		pop.Hide()
	}
	old := make(map[string][]*selectOption)
	for _, item := range s.accessibleOptions {
		old[item.label] = append(old[item.label], item)
	}
	items := make([]*selectOption, len(s.Options))
	for idx, label := range s.Options {
		var item *selectOption
		if previous := old[label]; len(previous) != 0 {
			item, old[label] = previous[0], previous[1:]
		} else {
			item = &selectOption{owner: s, label: label}
		}
		item.index = idx
		items[idx] = item
	}
	s.accessibleOptions = items
}

// AccessibilityOverlayOwner associates a Select popup with its combo box.
//
// Since: 2.9
func (p *PopUpMenu) AccessibilityOverlayOwner() fyne.CanvasObject {
	if p.selectEntryOwner != nil {
		return p.selectEntryOwner
	}
	if p.selectOwner == nil {
		return nil
	}
	return p.selectOwner
}

// AccessibilityFocusable reports whether this popup can own keyboard focus.
//
// Since: 2.9
func (p *PopUpMenu) AccessibilityFocusable() bool { return p.Visible() }

// AccessibilityFocus initializes the active choice and focuses the popup.
// Adapters call this once on opening, after normal popup layout completes.
//
// Since: 2.9
func (p *PopUpMenu) AccessibilityFocus() bool {
	if !p.Visible() {
		return false
	}
	if p.selectOwner != nil && p.activeItem == nil && len(p.Items) != 0 {
		p.activateItem(p.Items[max(p.selectOwner.SelectedIndex(), 0)].(*menuItem))
		p.revealSelectItem()
	}
	if p.selectOwner == nil && p.activeItem == nil {
		p.ActivateNext()
	}
	p.canvas.Focus(p)
	return p.canvas.Focused() == p
}

// AccessibilityActiveDescendant reports the option handling popup keyboard input.
//
// Since: 2.9
func (p *PopUpMenu) AccessibilityActiveDescendant() fyne.CanvasObject {
	if p.selectEntryOwner != nil {
		for i, item := range p.Items {
			if item == p.activeItem && i < len(p.selectEntryOwner.accessibleOptions) {
				return p.selectEntryOwner.accessibleOptions[i]
			}
		}
		return p.selectEntryOwner
	}
	if p.selectOwner != nil {
		for idx, item := range p.Items {
			if item == p.activeItem && idx < len(p.selectOwner.accessibleOptions) {
				return p.selectOwner.accessibleOptions[idx]
			}
		}
		return p.selectOwner
	}
	return p.Menu.AccessibilityActiveDescendant()
}

func (p *PopUpMenu) revealSelectItem() {
	if (p.selectOwner == nil && p.selectEntryOwner == nil) || p.activeItem == nil || !p.Visible() {
		return
	}
	r, ok := cache.Renderer(p).(*menuRenderer)
	if !ok {
		return
	}
	top, bottom := p.activeItem.Position().Y, p.activeItem.Position().Y+p.activeItem.Size().Height
	offset := r.scroll.Offset
	if top < offset.Y {
		offset.Y = top
	} else if bottom > offset.Y+r.scroll.Size().Height {
		offset.Y = bottom - r.scroll.Size().Height
	}
	r.scroll.ScrollToOffset(offset)
}

type selectOption struct {
	DisableableWidget
	owner *Select
	label string
	index int
}

func (o *selectOption) AccessibilityLabel() string           { return o.label }
func (*selectOption) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleListItem }
func (o *selectOption) Disabled() bool                       { return o.owner.Disabled() }

func (o *selectOption) AccessibilitySelectionItem() (owner fyne.CanvasObject, selected bool, position, count int) {
	return o.owner, o.owner.SelectedIndex() == o.index, o.index + 1, len(o.owner.Options)
}

func (o *selectOption) AccessibilitySelect(mode fyne.AccessibilitySelectionMode) bool {
	s := o.owner
	if s.Disabled() || o.index >= len(s.accessibleOptions) || s.accessibleOptions[o.index] != o {
		return false
	}
	selected := s.SelectedIndex() == o.index
	if mode == fyne.AccessibilitySelectAdd && s.SelectedIndex() >= 0 && !selected {
		return false
	}
	if mode == fyne.AccessibilitySelectRemove {
		if selected {
			s.ClearSelected()
		}
		return true
	}
	if s.popUp != nil {
		s.popUp.Dismiss()
	}
	if !selected {
		s.SetSelectedIndex(o.index)
	}
	return true
}

func (o *selectOption) AccessibilityFocusable() bool {
	return !o.Disabled() && o.owner.AccessibilityExpanded()
}

func (o *selectOption) AccessibilityFocus() bool {
	if !o.AccessibilityFocusable() {
		return false
	}
	p := o.owner.popUp
	p.activateItem(p.Items[o.index].(*menuItem))
	p.revealSelectItem()
	return o.owner.AccessibilityFocus()
}
func (o *selectOption) Position() fyne.Position { pos, _ := o.bounds(); return pos }
func (o *selectOption) Size() fyne.Size         { _, size := o.bounds(); return size }
func (o *selectOption) bounds() (fyne.Position, fyne.Size) {
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
