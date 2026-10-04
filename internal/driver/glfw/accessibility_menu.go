package glfw

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/internal/cache"
)

func (*MenuBar) AccessibilityLabel() string             { return "" }
func (*MenuBar) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleMenuBar }
func (b *MenuBar) AccessibilityChildren() []fyne.CanvasObject {
	if r, ok := cache.Renderer(b).(*menuBarRenderer); ok {
		return []fyne.CanvasObject{r.cont}
	}
	return nil
}
func (i *menuBarItem) AccessibilityLabel() string           { return i.Menu.Label }
func (*menuBarItem) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleMenuItem }
func (i *menuBarItem) AccessibilityExpanded() bool {
	return i.Parent.active && i.Parent.activeItem == i
}

func (i *menuBarItem) AccessibilitySetExpanded(expanded bool) {
	if expanded {
		i.Parent.activateChild(i)
		i.Parent.canvas.Focus(i)
	} else if i.AccessibilityExpanded() {
		i.Parent.deactivate()
	}
}

func (i *menuBarItem) AccessibilityActiveDescendant() fyne.CanvasObject {
	if i.AccessibilityExpanded() {
		return i.Child().AccessibilityActiveDescendant()
	}
	return nil
}

func (i *menuBarItem) AccessibilityChildren() []fyne.CanvasObject {
	if i.child == nil {
		return nil
	}
	if i.semanticChild == nil || i.semanticChild.CanvasObject != i.child {
		i.semanticChild = &menuBarSubmenu{CanvasObject: i.child, item: i}
	}
	return []fyne.CanvasObject{i.semanticChild}
}

type menuBarSubmenu struct {
	fyne.CanvasObject
	item *menuBarItem
}

func (s *menuBarSubmenu) AccessibilityLabel() string           { return s.item.Menu.Label }
func (*menuBarSubmenu) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleMenu }
func (s *menuBarSubmenu) AccessibilityChildren() []fyne.CanvasObject {
	return s.item.child.AccessibilityChildren()
}

func (s *menuBarSubmenu) Position() fyne.Position {
	d := fyne.CurrentApp().Driver()
	p, o := d.AbsolutePositionForObject(s.CanvasObject), d.AbsolutePositionForObject(s.item)
	return fyne.NewPos(p.X-o.X, p.Y-o.Y)
}

func (*menuBarSubmenu) AccessibilityPopup() bool { return true }
