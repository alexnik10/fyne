package widget

import "fyne.io/fyne/v2"

// AccessibilityChildren prevents recycled visual cells from being mistaken for
// stable logical items. List accessibility requires a logical collection model.
//
// Since: 2.9
func (*List) AccessibilityChildren() []fyne.CanvasObject { return nil }

// AccessibilityChildren reserves Table's logical collection boundary.
//
// Since: 2.9
func (*Table) AccessibilityChildren() []fyne.CanvasObject { return nil }

// AccessibilityChildren exposes only independent accessory controls. Text,
// placeholder, selection and caret renderers are represented by Entry itself.
//
// Since: 2.9
func (e *Entry) AccessibilityChildren() []fyne.CanvasObject {
	if e.ActionItem == nil {
		return nil
	}
	return []fyne.CanvasObject{e.ActionItem}
}
