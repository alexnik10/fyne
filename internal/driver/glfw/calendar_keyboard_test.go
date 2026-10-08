//go:build !no_glfw && !mobile

package glfw

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/go-gl/glfw/v3.4/glfw"
	"github.com/stretchr/testify/require"
)

func TestWindow_CalendarKeyboardNavigation(t *testing.T) {
	previousApp := fyne.CurrentApp()
	runOnMain(func() { fyne.SetCurrentApp(&calendarKeyboardApp{App: previousApp}) })
	defer runOnMain(func() { fyne.SetCurrentApp(previousApp) })
	w := createWindow("Calendar keyboard navigation")
	defer w.Close()
	var before, after *widget.Button
	var chosen time.Time
	runOnMain(func() {
		before, after = widget.NewButton("Before", nil), widget.NewButton("After", nil)
		calendar := widget.NewCalendar(time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC), func(value time.Time) { chosen = value })
		w.window.SetContent(container.NewVBox(before, calendar, after))
	})
	press := func(key glfw.Key, mods glfw.ModifierKey) {
		w.keyPressed(nil, key, 0, glfw.Press, mods)
		w.keyPressed(nil, key, 0, glfw.Release, mods)
	}
	w.Canvas().Focus(before)
	for i := 0; i < 3; i++ {
		press(glfw.KeyTab, 0)
	}
	day := w.Canvas().Focused()
	require.Implements(t, (*fyne.TabStop)(nil), day)
	press(glfw.KeyTab, 0)
	require.Same(t, after, w.Canvas().Focused())
	press(glfw.KeyTab, glfw.ModShift)
	require.Same(t, day, w.Canvas().Focused())
	press(glfw.KeyRight, 0)
	for _, key := range []glfw.Key{glfw.KeyEnter, glfw.KeyKPEnter, glfw.KeySpace} {
		runOnMain(func() { chosen = time.Time{} })
		press(key, 0)
		var selected string
		runOnMain(func() { selected = chosen.Format("2006-01-02") })
		require.Equal(t, "2026-02-01", selected)
	}
	day = w.Canvas().Focused()
	press(glfw.KeyTab, 0)
	require.Same(t, after, w.Canvas().Focused())
	press(glfw.KeyTab, glfw.ModShift)
	require.Same(t, day, w.Canvas().Focused())
}

type calendarKeyboardApp struct{ fyne.App }

func (*calendarKeyboardApp) Driver() fyne.Driver { return d }
