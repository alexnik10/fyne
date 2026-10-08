package widget

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/require"
)

func TestButtonKeyboardActivation(t *testing.T) {
	test.NewTempApp(t)
	calls := 0
	b := NewButton("Action", func() { calls++ })
	for i, key := range []fyne.KeyName{fyne.KeySpace, fyne.KeyReturn, fyne.KeyEnter} {
		b.TypedKey(&fyne.KeyEvent{Name: key})
		require.Equal(t, i+1, calls)
	}
	b.Disable()
	for _, key := range []fyne.KeyName{fyne.KeySpace, fyne.KeyReturn, fyne.KeyEnter} {
		b.TypedKey(&fyne.KeyEvent{Name: key})
	}
	require.Equal(t, 3, calls)
}

func TestPopUpMenuTabReturnsToUnderlyingDialog(t *testing.T) {
	test.NewTempApp(t)
	w := test.NewWindow(NewButton("Background", nil))
	defer w.Close()
	before, owner, after := NewButton("Before", nil), NewButton("Options", nil), NewButton("After", nil)
	popup := NewModalPopUp(fyne.NewContainerWithLayout(layout.NewVBoxLayout(), before, owner, after), w.Canvas())
	popup.Show()
	w.Canvas().Focus(owner)
	for _, backwards := range []bool{false, true} {
		menu := NewPopUpMenu(fyne.NewMenu("Options", fyne.NewMenuItem("Item", nil)), w.Canvas())
		menu.Show()
		menu.ActivateNext()
		require.True(t, menu.AcceptsTab())
		menu.dismissAndMoveFocus(backwards)
		require.Len(t, w.Canvas().Overlays().List(), 1)
		if backwards {
			require.Same(t, before, w.Canvas().Focused())
		} else {
			require.Same(t, after, w.Canvas().Focused())
		}
		w.Canvas().Focus(owner)
	}
}
