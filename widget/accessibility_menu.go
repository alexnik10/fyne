package widget

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/internal/cache"
)

// AccessibilityLabel returns the menu label.
// Since: 2.9
func (m *Menu) AccessibilityLabel() string { return m.label }

// AccessibilityRole identifies a menu.
// Since: 2.9
func (*Menu) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleMenu }

// AccessibilityChildren keeps the scroll viewport and excludes decorative objects.
// Submenus are owned by their menu item, not by renderer siblings.
// Since: 2.9
func (m *Menu) AccessibilityChildren() []fyne.CanvasObject {
	if r, ok := cache.Renderer(m.super()).(*menuRenderer); ok {
		return []fyne.CanvasObject{r.scroll}
	}
	return nil
}

// AccessibilityActiveDescendant returns the item receiving menu keyboard commands.
// Since: 2.9
func (m *Menu) AccessibilityActiveDescendant() fyne.CanvasObject {
	if m.activeItem == nil {
		return nil
	}
	if m.activeItem.isSubmenuOpen() {
		if child := m.activeItem.Child().AccessibilityActiveDescendant(); child != nil {
			return child
		}
	}
	return m.activeItem
}

func (i *menuItem) AccessibilityLabel() string           { return i.Item.Label }
func (*menuItem) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleMenuItem }
func (i *menuItem) AccessibilityMenuItem() (submenu, checkable bool) {
	return i.Item.ChildMenu != nil, i.Item.Checkable || i.Item.Checked
}

func (i *menuItem) AccessibilityShortcut() string {
	s, ok := i.Item.Shortcut.(fyne.KeyboardShortcut)
	if !ok {
		return ""
	}
	var keys []string
	mod := s.Mod()
	if mod&fyne.KeyModifierControl != 0 {
		keys = append(keys, "Ctrl")
	}
	if mod&fyne.KeyModifierAlt != 0 {
		keys = append(keys, "Alt")
	}
	if mod&fyne.KeyModifierShift != 0 {
		keys = append(keys, "Shift")
	}
	if mod&fyne.KeyModifierSuper != 0 {
		keys = append(keys, "Super")
	}
	keys = append(keys, string(s.Key()))
	return strings.Join(keys, "+")
}

func (i *menuItem) Disabled() bool {
	return i.Item.Disabled || (i.parent.parentItem != nil && i.parent.parentItem.Disabled())
}
func (i *menuItem) Disable() { i.Item.Disabled = true; i.Refresh() }
func (i *menuItem) Enable()  { i.Item.Disabled = false; i.Refresh() }
func (i *menuItem) AccessibilityActivate() {
	if !i.Disabled() && i.Item.ChildMenu == nil {
		i.trigger()
	}
}
func (i *menuItem) AccessibilityChecked() bool { return i.Item.Checked }
func (i *menuItem) AccessibilityToggle() {
	if i.Item.Checkable || i.Item.Checked {
		i.AccessibilityActivate()
	}
}
func (i *menuItem) AccessibilityExpanded() bool { return i.isSubmenuOpen() }
func (i *menuItem) AccessibilitySetExpanded(expanded bool) {
	if i.Disabled() || i.Item.ChildMenu == nil {
		return
	}
	if expanded {
		i.parent.activateItem(i)
		i.Child().Show()
		i.Child().ActivateNext()
		i.parent.Refresh()
	} else if i.child != nil {
		i.child.DeactivateChild()
		i.child.Hide()
		i.parent.Refresh()
	}
}
func (i *menuItem) AccessibilityFocusable() bool { return !i.Disabled() && i.parent.Visible() }
func (i *menuItem) AccessibilityFocus() bool {
	if !i.AccessibilityFocusable() {
		return false
	}
	c := fyne.CurrentApp().Driver().CanvasForObject(i)
	if c == nil {
		return false
	}
	root := i.parent
	for root.parentItem != nil {
		root = root.parentItem.parent
	}
	// A popup owns focus itself. Main-menu items retain their existing keyboard
	// owner, whose active descendant resolves through this same menu tree.
	if f, ok := root.super().(fyne.Focusable); ok {
		c.Focus(f)
	}
	focus, ok := c.Focused().(fyne.AccessibleActiveDescendant)
	if !ok {
		return false
	}
	i.parent.activateItem(i)
	return focus.AccessibilityActiveDescendant() == i
}

func (i *menuItem) AccessibilityChildren() []fyne.CanvasObject {
	if i.child == nil {
		return nil
	}
	if i.semanticChild == nil || i.semanticChild.Menu != i.child {
		i.semanticChild = &menuAccessibilitySubmenu{Menu: i.child, item: i}
	}
	return []fyne.CanvasObject{i.semanticChild}
}

// The visual submenu is a sibling so it can escape the scrolling viewport.
// The semantic wrapper places it under the opener with the same screen origin.
type menuAccessibilitySubmenu struct {
	*Menu
	item *menuItem
}

func (m *menuAccessibilitySubmenu) Position() fyne.Position {
	d := fyne.CurrentApp().Driver()
	p, o := d.AbsolutePositionForObject(m.Menu), d.AbsolutePositionForObject(m.item)
	return fyne.NewPos(p.X-o.X, p.Y-o.Y)
}

func (*menuAccessibilitySubmenu) AccessibilityPopup() bool { return true }

func (m *menuAccessibilitySubmenu) Visible() bool { return m.Menu.Visible() && !m.item.Disabled() }
