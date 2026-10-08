package accessibilitydemo

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"
)

func TestTextSplitKeyboardReading(t *testing.T) {
	test.NewTempApp(t)
	split := NewTextSplit()
	before, after := widget.NewButton("Before", nil), widget.NewButton("After", nil)
	w := test.NewWindow(container.New(layout.NewBorderLayout(before, after, nil, nil), before, split, after))
	defer w.Close()
	w.Resize(fyne.NewSize(700, 350))
	c := w.Canvas()
	tree := test.NewAccessibilityTree(c)
	grid := split.Leading.(*widget.TextGrid)
	rich := split.Trailing.(*widget.RichText)
	names := []string{"Before", "Text grid document", "Resize panes", "Pane two document", "After"}
	checkFocus := func(name string) test.AccessibilityNode {
		t.Helper()
		var focused []test.AccessibilityNode
		for _, n := range tree.Snapshot() {
			if n.Focused {
				focused = append(focused, n)
			}
		}
		require.Len(t, focused, 1)
		require.Equal(t, name, focused[0].Name)
		return focused[0]
	}
	c.Unfocus()
	for _, name := range names {
		c.FocusNext()
		checkFocus(name)
	}
	for i := len(names) - 2; i >= 0; i-- {
		c.FocusPrevious()
		checkFocus(names[i])
	}
	c.FocusNext()
	first := checkFocus("Text grid document")
	require.True(t, first.ReadOnly && first.Focusable)
	require.False(t, first.Document.SelectionDisabled)
	require.Contains(t, first.Document.Text, "Columns\tValue\nРусский текст 😀")
	original := grid.Text()
	grid.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	grid.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	info := grid.AccessibilityText()
	require.Equal(t, 2, info.Positions[info.Caret].Line)
	grid.TypedKey(&fyne.KeyEvent{Name: fyne.KeyHome})
	grid.KeyDown(&fyne.KeyEvent{Name: desktop.KeyShiftLeft})
	grid.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnd})
	grid.KeyUp(&fyne.KeyEvent{Name: desktop.KeyShiftLeft})
	require.Equal(t, "Русский текст 😀", grid.SelectedText())
	grid.TypedShortcut(&fyne.ShortcutCopy{})
	require.Equal(t, grid.SelectedText(), fyne.CurrentApp().Clipboard().Content())
	grid.TypedRune('x')
	grid.TypedKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
	grid.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDelete})
	grid.TypedShortcut(&fyne.ShortcutCut{})
	grid.TypedShortcut(&fyne.ShortcutPaste{})
	require.Equal(t, original, grid.Text())
	require.False(t, tree.Perform(first.ID, test.AccessibilitySetValue, "changed", 0))
	c.FocusNext()
	divider := checkFocus("Resize panes")
	for _, key := range []fyne.KeyName{fyne.KeyHome, fyne.KeyEnd} {
		c.Focused().TypedKey(&fyne.KeyEvent{Name: key})
		checkFocus("Resize panes")
		c.FocusPrevious()
		n := checkFocus("Text grid document")
		require.Equal(t, first.ID, n.ID)
		require.Equal(t, original, n.Document.Text)
		require.Greater(t, n.BoundsSize.Width, float32(0))
		c.FocusNext()
		require.Equal(t, divider.ID, checkFocus("Resize panes").ID)
		c.FocusNext()
		n = checkFocus("Pane two document")
		require.True(t, n.ReadOnly && n.Focusable)
		require.Greater(t, n.BoundsSize.Width, float32(0))
		c.Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
		require.Greater(t, rich.AccessibilityText().Caret, 0)
		c.Focused().(fyne.Shortcutable).TypedShortcut(&fyne.ShortcutSelectAll{})
		c.Focused().(fyne.Shortcutable).TypedShortcut(&fyne.ShortcutCopy{})
		require.True(t, strings.HasPrefix(fyne.CurrentApp().Clipboard().Content(), "Pane two\n"))
		c.FocusPrevious()
	}
	require.Empty(t, tree.Issues())
}
