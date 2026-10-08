package widget

import (
	"runtime"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/driver/software"
	"fyne.io/fyne/v2/internal/goos"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectWindowsKeyboard(t *testing.T) {
	test.NewTempApp(t)
	s := NewSelect([]string{"English", "Russian", "German"}, nil)
	w := test.NewWindow(s)
	defer w.Close()
	w.Canvas().Focus(s)
	press := func(key fyne.KeyName) {
		event := &fyne.KeyEvent{Name: key}
		if runtime.GOOS == goos.Windows {
			s.TypedKey(event) // Also exercise public dispatch in Windows CI.
		} else {
			require.True(t, s.typedKeyWindows(event))
		}
	}
	changes := 0
	s.OnChanged = func(string) { changes++ }
	for _, step := range []struct {
		key  fyne.KeyName
		want string
	}{
		{fyne.KeyUp, "English"},
		{fyne.KeyUp, "English"},
		{fyne.KeyDown, "Russian"},
		{fyne.KeyRight, "German"},
		{fyne.KeyDown, "German"},
		{fyne.KeyRight, "German"},
		{fyne.KeyLeft, "Russian"},
		{fyne.KeyUp, "English"},
	} {
		press(step.key)
		assert.Equal(t, step.want, s.Selected)
		assert.False(t, s.AccessibilityExpanded())
	}
	assert.Equal(t, 5, changes, "arrows at either boundary must not repeat OnChanged")
	s.SetSelected("Russian")
	for _, key := range []fyne.KeyName{fyne.KeySpace, fyne.KeyEnter, fyne.KeyReturn} {
		press(key)
		require.NotNil(t, s.popUp)
		assert.Equal(t, "Russian", s.popUp.activeItem.Item.Label)
		s.popUp.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
		assert.Equal(t, "Russian", s.Selected, "highlight is tentative")
		s.popUp.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
		assert.False(t, s.AccessibilityExpanded())
		assert.Equal(t, "Russian", s.Selected)
		assert.Same(t, s, w.Canvas().Focused())
	}
	press(fyne.KeySpace)
	s.popUp.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	s.popUp.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnter})
	assert.Equal(t, "German", s.Selected)
	assert.Same(t, s, w.Canvas().Focused())
	s.Disable()
	press(fyne.KeyDown)
	press(fyne.KeySpace)
	assert.Equal(t, "German", s.Selected)
	assert.False(t, s.AccessibilityExpanded())
	s.Enable()
	s.SetOptions(nil)
	for _, key := range []fyne.KeyName{fyne.KeyUp, fyne.KeyDown, fyne.KeyLeft, fyne.KeyRight, fyne.KeySpace} {
		press(key)
		assert.False(t, s.AccessibilityExpanded())
	}
}

func TestSelectPopupTabCommitsAndTraverses(t *testing.T) {
	app := test.NewTempApp(t)
	before, after := NewButton("Before", nil), NewButton("After", nil)
	s := NewSelect([]string{"English", "Russian", "German"}, nil)
	w := test.NewWindow(&fyne.Container{Layout: layout.NewVBoxLayout(), Objects: []fyne.CanvasObject{before, s, after}})
	defer w.Close()
	driver := &selectKeyboardDriver{Driver: app.Driver(), canvas: w.Canvas()}
	fyne.SetCurrentApp(&selectKeyboardApp{App: app, driver: driver})
	t.Cleanup(func() { fyne.SetCurrentApp(app) })
	for _, backwards := range []bool{false, true} {
		driver.modifiers = 0
		if backwards {
			driver.modifiers = fyne.KeyModifierShift
		}
		s.SetSelected("English")
		w.Canvas().Focus(s)
		s.AccessibilitySetExpanded(true)
		require.NotNil(t, s.popUp)
		s.popUp.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
		if runtime.GOOS == goos.Windows {
			require.True(t, s.popUp.AcceptsTab())
			s.popUp.TypedKey(&fyne.KeyEvent{Name: fyne.KeyTab})
		} else {
			s.popUp.commitSelectAndMoveFocus(backwards)
		}
		assert.Equal(t, "Russian", s.Selected)
		assert.False(t, s.AccessibilityExpanded())
		if backwards {
			assert.Same(t, before, w.Canvas().Focused())
		} else {
			assert.Same(t, after, w.Canvas().Focused())
		}
	}
	// A selection callback can intentionally redirect focus; Tab must respect it.
	s.SetSelected("English")
	s.OnChanged = func(string) { w.Canvas().Focus(before) }
	w.Canvas().Focus(s)
	s.AccessibilitySetExpanded(true)
	require.NotNil(t, s.popUp)
	s.popUp.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	s.popUp.commitSelectAndMoveFocus(false)
	assert.Same(t, before, w.Canvas().Focused())
}

func TestSelectDisclosureShortcuts(t *testing.T) {
	app := test.NewTempApp(t)
	s := NewSelect([]string{"English", "Russian"}, nil)
	w := test.NewWindow(s)
	defer w.Close()
	s.SetSelected("English")
	w.Canvas().Focus(s)
	for _, key := range []fyne.KeyName{fyne.KeyUp, fyne.KeyDown} {
		shortcut := &desktop.CustomShortcut{KeyName: key, Modifier: fyne.KeyModifierAlt}
		assert.True(t, isSelectDisclosureShortcut(shortcut))
		if runtime.GOOS == goos.Windows {
			s.TypedShortcut(shortcut)
			require.NotNil(t, s.popUp)
			assert.Equal(t, "English", s.popUp.activeItem.Item.Label)
			s.popUp.TypedShortcut(shortcut)
			assert.False(t, s.AccessibilityExpanded())
		}
		shortcut.Modifier |= fyne.KeyModifierControl
		assert.False(t, isSelectDisclosureShortcut(shortcut))
	}
	global := &desktop.CustomShortcut{KeyName: fyne.KeyK, Modifier: fyne.KeyModifierControl}
	called := 0
	// The test canvas wrapper hides TypedShortcut. Use the real software canvas
	// here, whose shortcut handler is also present on the desktop canvas.
	c := software.NewCanvasWithPainter(nil)
	c.SetContent(s)
	c.AddShortcut(global, func(fyne.Shortcut) { called++ })
	fyne.SetCurrentApp(&selectKeyboardApp{App: app, driver: &selectKeyboardDriver{Driver: app.Driver(), canvas: c}})
	t.Cleanup(func() { fyne.SetCurrentApp(app) })
	s.TypedShortcut(global)
	s.AccessibilitySetExpanded(true)
	require.NotNil(t, s.popUp)
	s.popUp.TypedShortcut(global)
	assert.Equal(t, 2, called, "adding Shortcutable must preserve canvas shortcuts")
	s.popUp.Dismiss()
	menu := NewPopUpMenu(fyne.NewMenu("", fyne.NewMenuItem("Ordinary", nil)), w.Canvas())
	assert.True(t, menu.AcceptsTab(), "Tab must close an ordinary popup menu")
}

type selectKeyboardApp struct {
	fyne.App
	driver fyne.Driver
}

func (a *selectKeyboardApp) Driver() fyne.Driver { return a.driver }

type selectKeyboardDriver struct {
	fyne.Driver
	canvas    fyne.Canvas
	modifiers fyne.KeyModifier
}

func (d *selectKeyboardDriver) CanvasForObject(fyne.CanvasObject) fyne.Canvas { return d.canvas }

func (d *selectKeyboardDriver) CurrentKeyModifiers() fyne.KeyModifier { return d.modifiers }

func (d *selectKeyboardDriver) CreateSplashWindow() fyne.Window { return d.CreateWindow("") }

func (*selectKeyboardDriver) HasSecondaryDisplay() bool { return false }
