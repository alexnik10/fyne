package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/internal/cache"
)

// AccessibilityLabel identifies a card by its title and subtitle.
// Since: 2.9
func (c *Card) AccessibilityLabel() string {
	if c.Subtitle == "" {
		return c.Title
	}
	if c.Title == "" {
		return c.Subtitle
	}
	return c.Title + ". " + c.Subtitle
}

// AccessibilityRole identifies a content group.
// Since: 2.9
func (*Card) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleContainer }

// AccessibilityChildren exposes content and an explicitly described image.
// Since: 2.9
func (c *Card) AccessibilityChildren() []fyne.CanvasObject {
	var children []fyne.CanvasObject
	if c.Image != nil {
		children = append(children, c.Image)
	}
	if c.Content != nil {
		children = append(children, c.Content)
	}
	return children
}

// AccessibilityLabel leaves the group name to explicit metadata.
// Since: 2.9
func (*CheckGroup) AccessibilityLabel() string { return "" }

// AccessibilityRole identifies a group of independent checkboxes.
// Since: 2.9
func (*CheckGroup) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleContainer }

// AccessibilityChildren preserves actual checkbox commands and keyboard focus.
// Since: 2.9
func (r *CheckGroup) AccessibilityChildren() []fyne.CanvasObject {
	children := make([]fyne.CanvasObject, len(r.items))
	for i, item := range r.items {
		children[i] = item
	}
	return children
}

// AccessibilityLabel leaves the toolbar name to application metadata.
// Since: 2.9
func (*Toolbar) AccessibilityLabel() string { return "" }

// AccessibilityRole identifies a toolbar.
// Since: 2.9
func (*Toolbar) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleToolBar }

// AccessibilityChildren exposes the persistent rendered toolbar objects.
// Since: 2.9
func (t *Toolbar) AccessibilityChildren() []fyne.CanvasObject {
	if r := cache.Renderer(t); r != nil {
		return r.Objects()
	}
	return nil
}

// SetAccessibilityInfo names a toolbar action without adding visible button text.
// Since: 2.9
func (t *ToolbarAction) SetAccessibilityInfo(info fyne.AccessibilityInfo) {
	t.button.SetAccessibilityInfo(info)
}

// AccessibilityLabel leaves an icon name to explicit metadata.
// Since: 2.9
func (*Icon) AccessibilityLabel() string { return "" }

// AccessibilityRole identifies an image.
// Since: 2.9
func (*Icon) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleImage }

// AccessibilityMode excludes decorative icons unless they have an explicit name.
// Since: 2.9
func (i *Icon) AccessibilityMode() fyne.AccessibilityMode {
	mode := i.BaseWidget.AccessibilityMode()
	if mode == fyne.AccessibilityAuto && i.AccessibilityInfo().Name == "" {
		return fyne.AccessibilityExclude
	}
	return mode
}

// AccessibilityLabel describes the file represented by this icon.
// Since: 2.9
func (i *FileIcon) AccessibilityLabel() string {
	if i.URI != nil {
		return i.URI.Name()
	}
	return ""
}

// AccessibilityRole identifies an image of a file.
// Since: 2.9
func (*FileIcon) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleImage }
