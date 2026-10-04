package container

import (
	"math"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
)

const dividerKeyboardStep = 0.01

func (*divider) AccessibilityLabel() string {
	return lang.X("accessibility.split.divider", "Resize panes")
}

func (*divider) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleSeparator }

func (d *divider) Disabled() bool { return !d.split.Leading.Visible() || !d.split.Trailing.Visible() }

func (d *divider) AccessibilityRange() (value, minimum, maximum, step float64) {
	length, leading, trailing := d.split.Size().Width, d.split.Leading.MinSize().Width, d.split.Trailing.MinSize().Width
	if !d.split.Horizontal {
		length, leading, trailing = d.split.Size().Height, d.split.Leading.MinSize().Height, d.split.Trailing.MinSize().Height
	}
	available := float64(length - dividerThickness(d))
	if available <= 0 {
		return 0, 0, 0, dividerKeyboardStep
	}
	minimum, maximum = min(1, float64(leading)/available), max(0, 1-float64(trailing)/available)
	if maximum < minimum {
		maximum = minimum
	}
	return min(max(d.split.Offset, minimum), maximum), minimum, maximum, dividerKeyboardStep
}

func (d *divider) AccessibilitySetRangeValue(value float64) {
	_, minimum, maximum, _ := d.AccessibilityRange()
	if d.Disabled() || math.IsNaN(value) || math.IsInf(value, 0) || value < minimum || value > maximum {
		return
	}
	d.split.SetOffset(value)
}

func (d *divider) AccessibilityValue() (value string, readOnly, protected bool) {
	v, _, _, _ := d.AccessibilityRange()
	return strconv.FormatFloat(v*100, 'f', 0, 64) + "%", d.Disabled(), false
}

func (d *divider) AccessibilitySetValue(value string) {
	if number, err := strconv.ParseFloat(value, 64); err == nil {
		d.AccessibilitySetRangeValue(number / 100)
	}
}

func (d *divider) FocusGained() { d.focused = true; d.Refresh() }

func (d *divider) FocusLost() { d.focused = false; d.Refresh() }

func (*divider) TypedRune(rune) {}

func (d *divider) TypedKey(event *fyne.KeyEvent) {
	value, minimum, maximum, step := d.AccessibilityRange()
	switch event.Name {
	case fyne.KeyLeft, fyne.KeyUp:
		value -= step
	case fyne.KeyRight, fyne.KeyDown:
		value += step
	case fyne.KeyHome:
		value = minimum
	case fyne.KeyEnd:
		value = maximum
	default:
		return
	}
	d.AccessibilitySetRangeValue(min(max(value, minimum), maximum))
}
