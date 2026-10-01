package widget

import (
	"math"

	"fyne.io/fyne/v2"
)

// AccessibilityActivate runs the button's activation command.
//
// Since: 2.9
func (b *Button) AccessibilityActivate() {
	if b.Disabled() {
		return
	}
	b.tapAnimation()
	if b.OnTapped != nil {
		b.OnTapped()
	}
}

// AccessibilityActivate runs the link's activation command.
//
// Since: 2.9
func (hl *Hyperlink) AccessibilityActivate() { hl.invokeAction() }

// AccessibilityLabel returns the check's label.
//
// Since: 2.9
func (c *Check) AccessibilityLabel() string { return c.Text }

// AccessibilityRole returns the check role.
//
// Since: 2.9
func (*Check) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleCheck }

// AccessibilityChecked returns the current checked state.
//
// Since: 2.9
func (c *Check) AccessibilityChecked() bool { return c.Checked }

// AccessibilityToggle executes the check's toggle command.
//
// Since: 2.9
func (c *Check) AccessibilityToggle() {
	if !c.Disabled() {
		c.SetChecked(!c.Checked)
	}
}

// AccessibilityLabel returns the entry's placeholder as a fallback name.
// Prefer SetAccessibilityInfo to give the field a permanent name.
//
// Since: 2.9
func (e *Entry) AccessibilityLabel() string { return e.PlaceHolder }

// AccessibilityRole returns the entry role.
//
// Since: 2.9
func (*Entry) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleEntry }

// AccessibilityInfo includes the entry's current validation error.
//
// Since: 2.9
func (e *Entry) AccessibilityInfo() fyne.AccessibilityInfo {
	info := e.BaseWidget.AccessibilityInfo()
	if e.validationError != nil {
		info.Invalid = true
		info.Description = e.validationError.Error()
	}
	return info
}

// AccessibilityValue returns the value without exposing protected text.
//
// Since: 2.9
func (e *Entry) AccessibilityValue() (value string, readOnly, protected bool) {
	if e.Password {
		return "", e.Disabled(), true
	}
	return e.Text, e.Disabled(), false
}

// AccessibilitySetValue edits the entry through its normal setter and callbacks.
//
// Since: 2.9
func (e *Entry) AccessibilitySetValue(value string) {
	if !e.Disabled() {
		e.SetText(value)
	}
}

// AccessibilityLabel returns an empty fallback; name sliders explicitly.
//
// Since: 2.9
func (*Slider) AccessibilityLabel() string { return "" }

// AccessibilityRole returns the slider role.
//
// Since: 2.9
func (*Slider) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleSlider }

// AccessibilityRange returns the range and step.
//
// Since: 2.9
func (s *Slider) AccessibilityRange() (value, minimum, maximum, step float64) {
	return s.Value, s.Min, s.Max, s.Step
}

// AccessibilitySetRangeValue updates an enabled slider within its bounds.
//
// Since: 2.9
func (s *Slider) AccessibilitySetRangeValue(value float64) {
	if !s.Disabled() && !math.IsNaN(value) && !math.IsInf(value, 0) && value >= s.Min && value <= s.Max {
		s.SetValue(value)
	}
}

// AccessibilityChildren returns popup content without its decoration.
//
// Since: 2.9
func (p *PopUp) AccessibilityChildren() []fyne.CanvasObject { return []fyne.CanvasObject{p.Content} }

// AccessibilityLabel returns the popup's fallback name.
//
// Since: 2.9
func (*PopUp) AccessibilityLabel() string { return "" }

// AccessibilityRole describes modal popups as dialogs.
//
// Since: 2.9
func (p *PopUp) AccessibilityRole() fyne.AccessibleRole {
	if p.modal {
		return fyne.AccessibleRoleDialog
	}
	return fyne.AccessibleRoleContainer
}

// AccessibilityChildren exposes the form layout including its action buttons.
//
// Since: 2.9
func (f *Form) AccessibilityChildren() []fyne.CanvasObject {
	var children []fyne.CanvasObject
	if f.itemGrid != nil {
		children = append(children, f.itemGrid)
	}
	if f.buttonBox != nil {
		children = append(children, f.buttonBox)
	}
	return children
}

// AccessibilityChildInfo associates form labels and validation with fields.
//
// Since: 2.9
func (f *Form) AccessibilityChildInfo(child fyne.CanvasObject) fyne.AccessibilityInfo {
	for _, item := range f.Items {
		if item.Widget != child {
			continue
		}
		info := fyne.AccessibilityInfo{Name: item.Text, Description: item.HintText, Required: item.Required, Invalid: item.invalid}
		if item.validationError != nil {
			info.Description = item.validationError.Error()
		}
		return info
	}
	return fyne.AccessibilityInfo{}
}
