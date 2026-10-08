package main

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/stretchr/testify/require"
)

func TestRemainingDatesTabOrder(t *testing.T) {
	test.NewTempApp(t)
	content := remainingDates(widget.NewLabel("Ready")).(*fyne.Container)
	fields := content.Objects[0].(*fyne.Container)
	date := fields.Objects[0].(*widget.DateEntry)
	choice := fields.Objects[1].(*widget.SelectEntry)
	w := test.NewWindow(content)
	defer w.Close()
	canvas := w.Canvas()
	canvas.Unfocus()
	for _, expected := range []fyne.Focusable{date, date.ActionItem.(fyne.Focusable), choice, choice.ActionItem.(fyne.Focusable)} {
		canvas.FocusNext()
		require.Same(t, expected, canvas.Focused())
	}
	tree := test.NewAccessibilityTree(canvas)
	for _, name := range []string{"Previous month", "Next month", time.Now().Format("Monday, 2 January 2006")} {
		canvas.FocusNext()
		var focusedNames []string
		for _, node := range tree.Snapshot() {
			if node.Focused {
				focusedNames = append(focusedNames, node.Name)
			}
		}
		require.Equal(t, []string{name}, focusedNames)
	}
	canvas.FocusNext()
	require.Same(t, date, canvas.Focused(), "wrap after the single calendar date stop")
}
