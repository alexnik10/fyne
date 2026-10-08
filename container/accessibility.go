package container

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// Accessibility is a layout-preserving semantic wrapper. Child controls keep
// their original identities, commands and keyboard focus. Excluding semantics
// does not remove a control from keyboard traversal; use it only for decoration.
//
// Since: 2.9
type Accessibility struct {
	widget.BaseWidget
	Content fyne.CanvasObject
	holder  *fyne.Container
}

// NewAccessibility wraps content with the specified semantic composition mode.
// Single cannot make an interactive control out of this wrapper: it supplies no
// role or commands. Use the underlying widget's Accessible interfaces for that.
//
// Since: 2.9
func NewAccessibility(mode fyne.AccessibilityMode, content fyne.CanvasObject) *Accessibility {
	c := &Accessibility{Content: content}
	c.ExtendBaseWidget(c)
	c.SetAccessibilityMode(mode)
	return c
}

// NewAccessibilityGroup creates a named group without merging child actions.
//
// Since: 2.9
func NewAccessibilityGroup(name string, content fyne.CanvasObject) *Accessibility {
	c := NewAccessibility(fyne.AccessibilityGroup, content)
	c.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: name})
	return c
}

// AccessibilityChildren returns the wrapped content, including an empty result
// when Content is nil. It does not depend on the renderer's lifetime.
func (c *Accessibility) AccessibilityChildren() []fyne.CanvasObject {
	if c.Content == nil {
		return nil
	}
	return []fyne.CanvasObject{c.Content}
}

// CreateRenderer implements fyne.Widget.
func (c *Accessibility) CreateRenderer() fyne.WidgetRenderer {
	c.holder = NewStack()
	if c.Content != nil {
		c.holder.Add(c.Content)
	}
	return widget.NewSimpleRenderer(c.holder)
}

// Refresh updates the wrapped content without replacing child objects.
func (c *Accessibility) Refresh() {
	if c.holder != nil {
		c.holder.Objects = c.AccessibilityChildren()
	}
	c.BaseWidget.Refresh()
}
