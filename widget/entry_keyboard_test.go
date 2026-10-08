package widget

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
)

func TestMultilineEntryFocusEscape(t *testing.T) {
	test.NewTempApp(t)
	before, after := NewButton("Before", nil), NewButton("Close", nil)
	skipped := NewButton("Disabled", nil)
	skipped.Disable()
	e := NewMultiLineEntry()
	w := test.NewWindow(&fyne.Container{Layout: layout.NewVBoxLayout(), Objects: []fyne.CanvasObject{before, e, skipped, after}})
	defer w.Close()
	e.SetText("Notes")
	w.Canvas().Focus(e)
	e.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnd})
	e.TypedKey(&fyne.KeyEvent{Name: fyne.KeyTab})
	assert.Equal(t, "Notes\t", e.Text)
	assert.Same(t, e, w.Canvas().Focused())
	row, column := e.CursorRow, e.CursorColumn
	e.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyTab, Modifier: fyne.KeyModifierControl})
	assert.Same(t, after, w.Canvas().Focused(), "canvas traversal skips disabled controls")
	assert.Equal(t, "Notes\t", e.Text)
	assert.Equal(t, row, e.CursorRow)
	assert.Equal(t, column, e.CursorColumn)
	w.Canvas().Focus(e)
	e.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyTab, Modifier: fyne.KeyModifierControl | fyne.KeyModifierShift})
	assert.Same(t, before, w.Canvas().Focused())
	assert.Equal(t, "Notes\t", e.Text)
	// A retained, unfocused entry must not move focus elsewhere.
	e.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyTab, Modifier: fyne.KeyModifierControl})
	assert.Same(t, before, w.Canvas().Focused())
	// Other modifier combinations and single-line entries keep their policy.
	w.Canvas().Focus(e)
	e.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyTab, Modifier: fyne.KeyModifierControl | fyne.KeyModifierAlt})
	assert.Same(t, e, w.Canvas().Focused())
	e.MultiLine = false
	e.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyTab, Modifier: fyne.KeyModifierControl})
	assert.Same(t, e, w.Canvas().Focused())
}

func TestMultilineEntryFocusEscapeStaysInModal(t *testing.T) {
	test.NewTempApp(t)
	background := NewButton("Background", nil)
	w := test.NewWindow(background)
	defer w.Close()
	w.Canvas().Focus(background)
	e, closeButton := NewMultiLineEntry(), NewButton("Close", nil)
	popup := NewModalPopUp(&fyne.Container{Layout: layout.NewVBoxLayout(), Objects: []fyne.CanvasObject{e, closeButton}}, w.Canvas())
	popup.Show()
	w.Canvas().Focus(e)
	for _, modifiers := range []fyne.KeyModifier{fyne.KeyModifierControl, fyne.KeyModifierControl | fyne.KeyModifierShift} {
		e.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyTab, Modifier: modifiers})
		assert.Same(t, closeButton, w.Canvas().Focused(), "traversal must stay in the modal scope")
		w.Canvas().Focus(e)
	}
	popup.Hide()
}
