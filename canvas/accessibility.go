package canvas

import "fyne.io/fyne/v2"

// AccessibilityLabel returns the displayed text.
// Since: 2.9
func (t *Text) AccessibilityLabel() string { return t.Text }

// AccessibilityRole identifies static text.
// Since: 2.9
func (*Text) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleText }

// AccessibilityMode omits decorative text without hiding its visual rendering.
// Since: 2.9
func (t *Text) AccessibilityMode() fyne.AccessibilityMode {
	if t.Decorative {
		return fyne.AccessibilityExclude
	}
	return fyne.AccessibilityAuto
}

// AccessibilityLabel returns the alternative text supplied by the application.
// Since: 2.9
func (i *Image) AccessibilityLabel() string { return i.AltText }

// AccessibilityRole identifies an image.
// Since: 2.9
func (*Image) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleImage }

// AccessibilityMode leaves unlabelled decorative images out of the semantic tree.
// Since: 2.9
func (i *Image) AccessibilityMode() fyne.AccessibilityMode {
	if i.AltText == "" {
		return fyne.AccessibilityExclude
	}
	return fyne.AccessibilityAuto
}
