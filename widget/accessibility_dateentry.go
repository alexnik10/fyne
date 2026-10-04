package widget

import (
	"time"

	"fyne.io/fyne/v2"
)

// AccessibilityExpanded reports whether the calendar is open.
// Since: 2.9
func (e *DateEntry) AccessibilityExpanded() bool { return e.popUp != nil && e.popUp.Visible() }

// AccessibilitySetExpanded opens the normal calendar with actual keyboard focus.
// Since: 2.9
func (e *DateEntry) AccessibilitySetExpanded(expanded bool) { e.setCalendarExpanded(expanded, true) }

func (e *DateEntry) setCalendarExpanded(expanded, focus bool) {
	if e.Disabled() || e.AccessibilityExpanded() == expanded {
		return
	}
	if !expanded {
		if e.popUp != nil {
			e.popUp.Hide()
		}
		return
	}
	c := fyne.CurrentApp().Driver().CanvasForObject(e.super())
	if c == nil {
		return
	}
	date := time.Now()
	if e.Date != nil {
		date = *e.Date
	}
	e.dropDown = NewCalendar(date, e.applyDate)
	if focus {
		c.Focus(e)
	}
	e.popUp = NewPopUp(e.dropDown, c)
	e.dropDown.dismiss = e.popUp.Hide
	e.popUp.ShowAtPosition(e.popUpPos())
	e.popUp.Resize(fyne.NewSize(e.Size().Width, e.popUp.MinSize().Height))
	if !focus {
		return
	}
	for _, object := range e.dropDown.dates.Objects {
		if day, ok := object.(*calendarDay); ok && day.date.Day() == date.Day() {
			c.Focus(day)
			break
		}
	}
}

// AccessibilitySetValueChecked validates a replacement before changing the date.
// Since: 2.9
func (e *DateEntry) AccessibilitySetValueChecked(value string) bool {
	if e.Disabled() {
		return false
	}
	if value == "" {
		e.SetDate(nil)
		return true
	}
	date, err := time.Parse(getLocaleDateFormat(), value)
	if err != nil {
		return false
	}
	e.SetDate(&date)
	return true
}

// TypedShortcut opens the calendar with Alt+Up/Down and retains editor shortcuts.
// Since: 2.9
func (e *DateEntry) TypedShortcut(s fyne.Shortcut) {
	if isSelectDisclosureShortcut(s) {
		e.AccessibilitySetExpanded(!e.AccessibilityExpanded())
		return
	}
	e.Entry.TypedShortcut(s)
}
