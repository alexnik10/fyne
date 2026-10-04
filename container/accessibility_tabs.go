package container

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/internal/cache"
	"fyne.io/fyne/v2/lang"
)

// The keyboard focus belongs to the tab strip; semantic items belong to TabItem
// objects, never to temporary overflow menus or renderer buttons.
type tabsAccessibility struct {
	focused bool
	nodes   map[*TabItem]*tabAccessibilityItem
}

func (a *tabsAccessibility) sync(t baseTabs) {
	if a.nodes == nil {
		a.nodes = make(map[*TabItem]*tabAccessibilityItem)
	}
	live := make(map[*TabItem]bool, len(t.items()))
	for _, item := range t.items() {
		live[item] = true
		if a.nodes[item] == nil {
			a.nodes[item] = &tabAccessibilityItem{tabs: t, item: item}
		}
	}
	for item := range a.nodes {
		if !live[item] {
			delete(a.nodes, item)
		}
	}
}

func (a *tabsAccessibility) children(t baseTabs) []fyne.CanvasObject {
	a.sync(t)
	children := make([]fyne.CanvasObject, 0, len(t.items())+2)
	for _, item := range t.items() {
		children = append(children, a.nodes[item])
	}
	if item := selected(t); item != nil && item.Content != nil {
		children = append(children, item.Content)
	}
	if r, ok := cache.CachedRenderer(t); ok {
		switch r := r.(type) {
		case *docTabsRenderer:
			children = append(children, r.box)
		case *appTabsRenderer:
			// The overflow control is already represented by the logical tab items.
		}
	}
	return children
}

func (a *tabsAccessibility) active(t baseTabs) fyne.CanvasObject {
	a.sync(t)
	if item := selected(t); item != nil {
		return a.nodes[item]
	}
	return nil
}

func tabsKey(t baseTabs, e *fyne.KeyEvent) {
	items := t.items()
	if len(items) == 0 {
		return
	}
	if e.Name == fyne.KeyDelete {
		if docs, ok := t.(*DocTabs); ok {
			if item := selected(t); item != nil && !item.Disabled() {
				docs.close(item)
			}
		}
		return
	}
	index, step := tabsKeyDirection(t, e.Name)
	if step == 0 {
		return
	}
	for range items {
		index = (index + step + len(items)) % len(items)
		if !items[index].Disabled() {
			selectIndex(t, index)
			return
		}
	}
}

func tabsKeyDirection(t baseTabs, key fyne.KeyName) (index, step int) {
	index = t.getCurrent()
	vertical := t.tabLocation() == TabLocationLeading || t.tabLocation() == TabLocationTrailing
	switch key {
	case fyne.KeyHome:
		return -1, 1
	case fyne.KeyEnd:
		return len(t.items()), -1
	case fyne.KeyLeft:
		if !vertical {
			return index, -1
		}
	case fyne.KeyRight:
		if !vertical {
			return index, 1
		}
	case fyne.KeyUp:
		if vertical {
			return index, -1
		}
	case fyne.KeyDown:
		if vertical {
			return index, 1
		}
	}
	return index, 0
}

// AccessibilityLabel returns the explicit metadata fallback.
// Since: 2.9
func (*AppTabs) AccessibilityLabel() string { return "" }

// AccessibilityRole identifies a tab control.
// Since: 2.9
func (*AppTabs) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleTab }

// AccessibilitySelection describes the single selected page.
// Since: 2.9
func (*AppTabs) AccessibilitySelection() (multiple, required bool) { return false, true }

// AccessibilityChildren exposes logical tabs and page controls.
// Since: 2.9
func (t *AppTabs) AccessibilityChildren() []fyne.CanvasObject { return t.children(t) }

// AccessibilityActiveDescendant follows the real tab-strip focus.
// Since: 2.9
func (t *AppTabs) AccessibilityActiveDescendant() fyne.CanvasObject {
	return t.active(t)
}

// FocusGained implements fyne.Focusable.
// Since: 2.9
func (t *AppTabs) FocusGained() { t.focused = true; t.Refresh() }

// FocusLost implements fyne.Focusable.
// Since: 2.9
func (t *AppTabs) FocusLost() { t.focused = false; t.Refresh() }

// TypedRune implements fyne.Focusable.
// Since: 2.9
func (*AppTabs) TypedRune(rune) {}

// TypedKey navigates the tab strip.
// Since: 2.9
func (t *AppTabs) TypedKey(e *fyne.KeyEvent) { tabsKey(t, e) }

// Refresh updates tabs and retires removed semantic objects.
func (t *AppTabs) Refresh() { t.sync(t); t.BaseWidget.Refresh() }

// AccessibilityLabel returns the explicit metadata fallback.
// Since: 2.9
func (*DocTabs) AccessibilityLabel() string { return "" }

// AccessibilityRole identifies a tab control.
// Since: 2.9
func (*DocTabs) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleTab }

// AccessibilitySelection describes the single selected document.
// Since: 2.9
func (*DocTabs) AccessibilitySelection() (multiple, required bool) { return false, true }

// AccessibilityChildren exposes logical tabs and page controls.
// Since: 2.9
func (t *DocTabs) AccessibilityChildren() []fyne.CanvasObject { return t.children(t) }

// AccessibilityActiveDescendant follows the real tab-strip focus.
// Since: 2.9
func (t *DocTabs) AccessibilityActiveDescendant() fyne.CanvasObject {
	return t.active(t)
}

// FocusGained implements fyne.Focusable.
// Since: 2.9
func (t *DocTabs) FocusGained() { t.focused = true; t.Refresh() }

// FocusLost implements fyne.Focusable.
// Since: 2.9
func (t *DocTabs) FocusLost() { t.focused = false; t.Refresh() }

// TypedRune implements fyne.Focusable.
// Since: 2.9
func (*DocTabs) TypedRune(rune) {}

// TypedKey navigates tabs and closes the selected document with Delete.
// Since: 2.9
func (t *DocTabs) TypedKey(e *fyne.KeyEvent) { tabsKey(t, e) }

// Refresh updates tabs and retires removed semantic objects.
func (t *DocTabs) Refresh() { t.sync(t); t.BaseWidget.Refresh() }

type tabAccessibilityItem struct {
	tabs  baseTabs
	item  *TabItem
	close *tabAccessibilityClose
}

func (i *tabAccessibilityItem) index() int {
	for n, item := range i.tabs.items() {
		if item == i.item {
			return n
		}
	}
	return -1
}
func (i *tabAccessibilityItem) AccessibilityLabel() string { return i.item.Text }
func (*tabAccessibilityItem) AccessibilityRole() fyne.AccessibleRole {
	return fyne.AccessibleRoleTabItem
}

func (i *tabAccessibilityItem) AccessibilitySelectionItem() (owner fyne.CanvasObject, isSelected bool, position, count int) {
	return i.tabs, selected(i.tabs) == i.item, i.index() + 1, len(i.tabs.items())
}

func (i *tabAccessibilityItem) AccessibilitySelect(mode fyne.AccessibilitySelectionMode) bool {
	if i.index() < 0 || i.Disabled() {
		return false
	}
	if mode == fyne.AccessibilitySelectRemove {
		return selected(i.tabs) != i.item
	}
	if mode == fyne.AccessibilitySelectAdd && selected(i.tabs) != i.item {
		return false
	}
	if mode != fyne.AccessibilitySelectAdd && mode != fyne.AccessibilitySelectReplace {
		return false
	}
	selectItem(i.tabs, i.item)
	return true
}
func (i *tabAccessibilityItem) AccessibilityFocusable() bool { return i.index() >= 0 && !i.Disabled() }
func (i *tabAccessibilityItem) AccessibilityFocus() bool {
	if !i.AccessibilityFocusable() {
		return false
	}
	c := fyne.CurrentApp().Driver().CanvasForObject(i.tabs)
	f, ok := i.tabs.(fyne.Focusable)
	if c == nil || !ok {
		return false
	}
	selectItem(i.tabs, i.item)
	c.Focus(f)
	return c.Focused() == f
}

func (i *tabAccessibilityItem) AccessibilityChildren() []fyne.CanvasObject {
	if _, ok := i.tabs.(*DocTabs); !ok {
		return nil
	}
	if i.close == nil {
		i.close = &tabAccessibilityClose{item: i}
	}
	return []fyne.CanvasObject{i.close}
}
func (i *tabAccessibilityItem) Disabled() bool { return i.item.Disabled() }
func (*tabAccessibilityItem) Disable()         {}
func (*tabAccessibilityItem) Enable()          {}
func (i *tabAccessibilityItem) Position() fyne.Position {
	if b := i.item.button; b != nil {
		d := fyne.CurrentApp().Driver()
		if d.CanvasForObject(b) != nil {
			p, o := d.AbsolutePositionForObject(b), d.AbsolutePositionForObject(i.tabs)
			return fyne.NewPos(p.X-o.X, p.Y-o.Y)
		}
	}
	return fyne.Position{}
}

func (i *tabAccessibilityItem) Size() fyne.Size {
	if b := i.item.button; b != nil && fyne.CurrentApp().Driver().CanvasForObject(b) != nil {
		return b.Size()
	}
	return fyne.Size{}
}
func (*tabAccessibilityItem) MinSize() fyne.Size { return fyne.Size{} }
func (*tabAccessibilityItem) Move(fyne.Position) {}
func (*tabAccessibilityItem) Resize(fyne.Size)   {}
func (*tabAccessibilityItem) Hide()              {}
func (*tabAccessibilityItem) Show()              {}
func (i *tabAccessibilityItem) Visible() bool    { return i.index() >= 0 }
func (*tabAccessibilityItem) Refresh()           {}

type tabAccessibilityClose struct{ item *tabAccessibilityItem }

func (b *tabAccessibilityClose) AccessibilityLabel() string {
	return lang.X("accessibility.tab.close", "Close {{.Name}}", map[string]any{"Name": b.item.item.Text})
}

func (*tabAccessibilityClose) AccessibilityRole() fyne.AccessibleRole {
	return fyne.AccessibleRoleButton
}

func (b *tabAccessibilityClose) AccessibilityActivate() {
	if b.item.AccessibilityFocusable() {
		b.item.tabs.(*DocTabs).close(b.item.item)
	}
}
func (b *tabAccessibilityClose) Disabled() bool        { return b.item.Disabled() }
func (*tabAccessibilityClose) Disable()                {}
func (*tabAccessibilityClose) Enable()                 {}
func (*tabAccessibilityClose) Position() fyne.Position { return fyne.Position{} }
func (*tabAccessibilityClose) Size() fyne.Size         { return fyne.Size{} }
func (*tabAccessibilityClose) MinSize() fyne.Size      { return fyne.Size{} }
func (*tabAccessibilityClose) Move(fyne.Position)      {}
func (*tabAccessibilityClose) Resize(fyne.Size)        {}
func (*tabAccessibilityClose) Hide()                   {}
func (*tabAccessibilityClose) Show()                   {}
func (b *tabAccessibilityClose) Visible() bool         { return b.item.Visible() }
func (*tabAccessibilityClose) Refresh()                {}
