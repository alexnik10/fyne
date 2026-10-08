package widget

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/test"
)

func TestNewCalendar(t *testing.T) {
	now := time.Now()
	c := NewCalendar(now, func(time.Time) {})
	assert.Equal(t, now.Day(), c.currentTime.Day())
	assert.Equal(t, int(now.Month()), int(c.currentTime.Month()))
	assert.Equal(t, now.Year(), c.currentTime.Year())

	_ = test.WidgetRenderer(c) // and render
	assert.Equal(t, now.Format("January 2006"), c.monthLabel.Text)
}

func TestNewCalendar_ButtonDate(t *testing.T) {
	date := time.Now()
	c := NewCalendar(date, func(time.Time) {})
	_ = test.WidgetRenderer(c) // and render

	endNextMonth := date.AddDate(0, 1, 0).AddDate(0, 0, -(date.Day() - 1))
	last := endNextMonth.AddDate(0, 0, -1)

	firstDate := firstDateButton(c.dates)
	assert.Equal(t, "1", firstDate.Text)
	lastDate := c.dates.Objects[len(c.dates.Objects)-1].(*calendarDay)
	assert.Equal(t, strconv.Itoa(last.Day()), lastDate.Text)
}

func TestNewCalendar_Next(t *testing.T) {
	date := time.Now()
	c := NewCalendar(date, func(time.Time) {})
	_ = test.WidgetRenderer(c) // and render

	assert.Equal(t, date.Format("January 2006"), c.monthLabel.Text)

	test.Tap(c.monthNext)
	date = date.AddDate(0, 1, 0)
	assert.Equal(t, date.Format("January 2006"), c.monthLabel.Text)
}

func TestNewCalendar_Previous(t *testing.T) {
	date := time.Now()
	c := NewCalendar(date, func(time.Time) {})
	_ = test.WidgetRenderer(c) // and render

	assert.Equal(t, date.Format("January 2006"), c.monthLabel.Text)

	test.Tap(c.monthPrevious)
	date = date.AddDate(0, -1, 0)
	assert.Equal(t, date.Format("January 2006"), c.monthLabel.Text)
}

func TestNewCalendar_Resize(t *testing.T) {
	date := time.Now()
	c := NewCalendar(date, func(time.Time) {})
	r := test.WidgetRenderer(c) // and render
	gridLayout := c.dates.Layout.(*calendarLayout)

	baseSize := c.MinSize()
	r.Layout(baseSize)
	minSize := gridLayout.cellSize

	r.Layout(baseSize.AddWidthHeight(100, 0))
	assert.Greater(t, gridLayout.cellSize.Width, minSize.Width)
	assert.Equal(t, gridLayout.cellSize.Height, minSize.Height)

	r.Layout(baseSize.AddWidthHeight(0, 100))
	assert.Equal(t, gridLayout.cellSize.Width, minSize.Width)
	assert.Greater(t, gridLayout.cellSize.Height, minSize.Height)

	r.Layout(baseSize.AddWidthHeight(100, 100))
	assert.Greater(t, gridLayout.cellSize.Width, minSize.Width)
	assert.Greater(t, gridLayout.cellSize.Height, minSize.Height)
}

func TestCalendar_KeyboardNavigation(t *testing.T) {
	test.NewTempApp(t)
	date := time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC)
	var chosen time.Time
	c := NewCalendar(date, func(value time.Time) { chosen = value })
	before, after := NewButton("Before", nil), NewButton("After", nil)
	w := test.NewWindow(&fyne.Container{Layout: layout.NewVBoxLayout(), Objects: []fyne.CanvasObject{before, c, after}})
	defer w.Close()
	canvas := w.Canvas()
	canvas.Unfocus()
	for _, expected := range []fyne.Focusable{before, c.monthPrevious, c.monthNext} {
		canvas.FocusNext()
		require.Same(t, expected, canvas.Focused())
	}
	canvas.FocusNext()
	day := canvas.Focused().(*calendarDay)
	require.Equal(t, 31, day.date.Day(), "initial Tab stop is the selected date")
	canvas.FocusNext()
	require.Same(t, after, canvas.Focused(), "Tab leaves the date grid immediately")
	canvas.FocusPrevious()
	require.Same(t, day, canvas.Focused())
	canvas.FocusPrevious()
	require.Same(t, c.monthNext, canvas.Focused())
	canvas.FocusNext()
	day.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	day = canvas.Focused().(*calendarDay)
	require.Equal(t, "2026-02-01", day.date.Format(calendarDateKeyFormat))
	assert.True(t, chosen.IsZero(), "navigation does not select a date")
	canvas.FocusNext()
	require.Same(t, after, canvas.Focused())
	canvas.FocusPrevious()
	require.Same(t, day, canvas.Focused(), "return to the last active date")

	for _, key := range []fyne.KeyName{fyne.KeyReturn, fyne.KeyEnter, fyne.KeySpace} {
		chosen = time.Time{}
		day.TypedKey(&fyne.KeyEvent{Name: key})
		assert.Equal(t, day.date, chosen, "activation key %s", key)
	}

	// All dates remain reachable through the accessibility focus contract.
	first := firstDateButton(c.dates)
	first.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	day = canvas.Focused().(*calendarDay)
	require.Equal(t, 8, day.date.Day())
	require.True(t, first.AccessibilityFocus())
	require.Same(t, first, canvas.Focused())
	canvas.FocusNext()
	require.Same(t, after, canvas.Focused())
}

func TestCalendar_TabStopInShortMonth(t *testing.T) {
	test.NewTempApp(t)
	c := NewCalendar(time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC), nil)
	w := test.NewWindow(c)
	defer w.Close()
	w.Canvas().Focus(c.monthNext)
	c.monthNext.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	require.Equal(t, time.February, c.currentTime.Month())
	w.Canvas().FocusNext()
	day := w.Canvas().Focused().(*calendarDay)
	require.Equal(t, 28, day.date.Day())
	w.Canvas().FocusPrevious()
	require.Same(t, c.monthNext, w.Canvas().Focused())
	c.monthNext.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnter})
	w.Canvas().FocusNext()
	day = w.Canvas().Focused().(*calendarDay)
	require.Equal(t, time.March, day.date.Month())
	require.Equal(t, 28, day.date.Day(), "remember the last focused day")
}

func firstDateButton(c *fyne.Container) *calendarDay {
	for _, b := range c.Objects {
		if nonBlank, ok := b.(*calendarDay); ok {
			return nonBlank
		}
	}

	return nil
}
