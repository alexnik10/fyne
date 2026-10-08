package widget_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPasswordAccessorySemanticsAndKeyboard(t *testing.T) {
	test.NewTempApp(t)
	entry := widget.NewPasswordEntry()
	entry.SetText("secret")
	next := widget.NewButton("Next", nil)
	w := test.NewWindow(container.NewVBox(entry, next))
	defer w.Close()
	inspector := test.NewAccessibilityTree(w.Canvas())
	field, ok := inspector.Node(entry)
	require.True(t, ok)
	require.True(t, field.Protected)
	assert.Empty(t, field.Text)
	assert.NotContains(t, field.Document.Text, "secret")
	revealer := entry.ActionItem.(fyne.Focusable)
	button, ok := inspector.Node(entry.ActionItem)
	require.True(t, ok)
	assert.Equal(t, "Show password", button.Name)
	assert.True(t, button.Invoke)
	assert.True(t, button.Focusable)
	w.Canvas().Focus(entry)
	w.Canvas().FocusNext()
	assert.Same(t, revealer, w.Canvas().Focused())
	revealer.TypedKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	assert.False(t, entry.Password)
	assert.Same(t, entry, w.Canvas().Focused(), "activation restores editing focus")
	button2, _ := inspector.Node(entry.ActionItem)
	assert.Equal(t, button.ID, button2.ID)
	assert.Equal(t, "Hide password", button2.Name)
	require.True(t, inspector.Perform(button.ID, test.AccessibilityActivate, "", 0))
	assert.True(t, entry.Password)
	entry.Disable()
	button2, _ = inspector.Node(entry.ActionItem)
	assert.True(t, button2.Disabled)
	assert.False(t, inspector.Perform(button.ID, test.AccessibilityActivate, "", 0))
	w.Canvas().Unfocus()
	w.Canvas().FocusNext()
	assert.Same(t, next, w.Canvas().Focused())
}

func TestPasswordAccessoryFocusOrder(t *testing.T) {
	test.NewTempApp(t)
	before := widget.NewButton("Before", nil)
	entry := widget.NewPasswordEntry()
	after := widget.NewButton("After", nil)
	w := test.NewWindow(container.NewVBox(before, entry, after))
	defer w.Close()
	c := w.Canvas()
	revealer := entry.ActionItem.(fyne.Focusable)
	order := []fyne.Focusable{before, entry, revealer, after}

	for _, expected := range order {
		c.FocusNext()
		require.Same(t, expected, c.Focused())
	}
	for i := len(order) - 2; i >= 0; i-- {
		c.FocusPrevious()
		require.Same(t, order[i], c.Focused())
	}
	c.FocusPrevious()
	require.Same(t, after, c.Focused(), "reverse traversal wraps")

	c.FocusPrevious()
	require.Same(t, revealer, c.Focused())
	c.FocusNext()
	require.Same(t, after, c.Focused(), "changing direction returns to the next control")

	entry.Disable()
	c.FocusPrevious()
	require.Same(t, before, c.Focused(), "disabled field and revealer are both skipped")
	entry.Enable()
	entry.Hide()
	c.FocusNext()
	require.Same(t, after, c.Focused())
	c.FocusPrevious()
	require.Same(t, before, c.Focused(), "hidden field and revealer are both skipped")
}
