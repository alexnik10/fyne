package widget

import (
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
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

// AccessibilityText returns text and selection without exposing a password.
//
// Since: 2.9
func (e *Entry) AccessibilityText() fyne.AccessibilityTextInfo {
	provider := e.textProvider()
	length := utf8.RuneCountInString(e.Text)
	caret := min(max(e.CursorTextOffset(), 0), length)
	info := fyne.AccessibilityTextInfo{
		Text: e.Text, Caret: caret, SelectionStart: caret,
		SelectionEnd: caret, Revision: e.accessibilityTextRevision, ViewportSize: e.Size(),
	}
	if e.Password {
		info.Text = strings.Repeat(passwordChar, length)
	}
	info.WordBoundaries = entryWordBoundaries(info.Text)
	if e.sel != nil && e.sel.selecting {
		anchor := min(max(textPosFromRowCol(e.sel.selectRow, e.sel.selectColumn, provider), 0), length)
		info.SelectionStart, info.SelectionEnd = min(anchor, caret), max(anchor, caret)
	}
	if e.content == nil {
		return info
	}
	origin := e.content.Position()
	if e.scroller != nil && e.scroller.Visible() {
		origin = origin.Add(e.scroller.Position())
		info.ViewportPosition, info.ViewportSize = e.scroller.Position(), e.scroller.Size()
	}
	th := e.Theme()
	pad, size := th.Size(theme.SizeNameInnerPadding), th.Size(theme.SizeNameText)
	info.Positions = make([]fyne.AccessibilityTextPosition, length+1)
	for offset := range info.Positions {
		row, col := e.rowColFromTextPos(offset)
		y, height := provider.rowGeometry(row)
		x := provider.lineSizeToColumn(col, row, size, pad).Width
		info.Positions[offset] = fyne.AccessibilityTextPosition{
			Position: origin.Add(fyne.NewPos(x, y+pad-th.Size(theme.SizeNameInputBorder))), Height: height, Line: row,
		}
	}
	return info
}

// AccessibilitySelectText changes selection through the entry's normal cursor
// and scrolling machinery. It does not call OnChanged or replace text.
//
// Since: 2.9
func (e *Entry) AccessibilitySelectText(start, end int) {
	if e.Disabled() {
		return
	}
	length := utf8.RuneCountInString(e.Text)
	start, end = min(max(start, 0), length), min(max(end, 0), length)
	e.syncSelectable()
	e.sel.selectRow, e.sel.selectColumn = e.rowColFromTextPos(start)
	e.CursorRow, e.CursorColumn = e.rowColFromTextPos(end)
	e.syncSelectable()
	e.sel.selecting = start != end
	e.Refresh()
}

// AccessibilityScrollText reveals a range without changing its selection.
//
// Since: 2.9
func (e *Entry) AccessibilityScrollText(start, end int, alignTop bool) {
	if e.scroller == nil || e.scroller.Content == nil {
		return
	}
	info := e.AccessibilityText()
	if len(info.Positions) == 0 {
		return
	}
	offset := end
	if alignTop {
		offset = start
	}
	p := info.Positions[min(max(offset, 0), len(info.Positions)-1)]
	target := p.Position.Subtract(e.scroller.Position()).Add(e.scroller.Offset)
	if !alignTop {
		target.Y -= e.scroller.Size().Height - p.Height
	}
	target.X = max(0, target.X-e.Theme().Size(theme.SizeNameInnerPadding))
	target.Y = max(0, target.Y)
	e.scroller.ScrollToOffset(target)
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

// AccessibilityValue exposes the numeric value as text as well as a range.
// Some clients subscribe to Value changes only on the focused element.
//
// Since: 2.9
func (s *Slider) AccessibilityValue() (value string, readOnly, protected bool) {
	return strconv.FormatFloat(s.Value, 'f', -1, 64), s.Disabled(), false
}

// AccessibilitySetValue accepts the same numeric values as RangeValue.
//
// Since: 2.9
func (s *Slider) AccessibilitySetValue(value string) {
	if number, err := strconv.ParseFloat(value, 64); err == nil {
		s.AccessibilitySetRangeValue(number)
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
