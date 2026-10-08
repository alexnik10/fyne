package widget

import (
	"math"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
)

// AccessibilityLabel leaves the progress name to application metadata.
// Since: 2.9
func (*ProgressBar) AccessibilityLabel() string { return "" }

// AccessibilityRole identifies a progress indicator.
// Since: 2.9
func (*ProgressBar) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleProgressBar }

// AccessibilityRange exposes progress as a read-only range.
// Since: 2.9
func (p *ProgressBar) AccessibilityRange() (value, minimum, maximum, step float64) {
	return p.Value, p.Min, p.Max, math.NaN()
}

// AccessibilitySetRangeValue cannot change application-owned progress.
// Since: 2.9
func (*ProgressBar) AccessibilitySetRangeValue(float64) {}

// AccessibilityValue exposes the displayed progress and marks both value patterns read-only.
// Since: 2.9
func (p *ProgressBar) AccessibilityValue() (value string, readOnly, protected bool) {
	if p.TextFormatter != nil {
		return p.TextFormatter(), true, false
	}
	percent := float64(0)
	if p.Max > p.Min {
		percent = min(100, max(0, (p.Value-p.Min)/(p.Max-p.Min)*100))
	}
	return strconv.FormatFloat(percent, 'f', 0, 64) + "%", true, false
}

// AccessibilitySetValue cannot change application-owned progress.
// Since: 2.9
func (*ProgressBar) AccessibilitySetValue(string) {}

// AccessibilityLabel leaves the indicator name to application metadata.
// Since: 2.9
func (*ProgressBarInfinite) AccessibilityLabel() string { return "" }

// AccessibilityRole identifies an indeterminate progress indicator.
// Since: 2.9
func (*ProgressBarInfinite) AccessibilityRole() fyne.AccessibleRole {
	return fyne.AccessibleRoleProgressBar
}

// AccessibilityValue reports activity without inventing a numeric percentage.
// Since: 2.9
func (p *ProgressBarInfinite) AccessibilityValue() (value string, readOnly, protected bool) {
	return accessibilityActivity(p.Running()), true, false
}

// AccessibilitySetValue cannot start or stop application work.
// Since: 2.9
func (*ProgressBarInfinite) AccessibilitySetValue(string) {}

// AccessibilityLabel leaves the activity name to application metadata.
// Since: 2.9
func (*Activity) AccessibilityLabel() string { return "" }

// AccessibilityRole identifies an indeterminate progress indicator.
// Since: 2.9
func (*Activity) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleProgressBar }

// AccessibilityValue reports the current activity state.
// Since: 2.9
func (a *Activity) AccessibilityValue() (value string, readOnly, protected bool) {
	return accessibilityActivity(a.started), true, false
}

// AccessibilitySetValue cannot start or stop application work.
// Since: 2.9
func (*Activity) AccessibilitySetValue(string) {}

func accessibilityActivity(active bool) string {
	if active {
		return lang.X("accessibility.activity.running", "In progress")
	}
	return lang.X("accessibility.activity.stopped", "Stopped")
}
