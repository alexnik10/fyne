package widget

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/internal/cache"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/require"
)

func TestTextGridSelectable(t *testing.T) {
	test.NewTempApp(t)
	grid := NewTextGridFromString("A\tB\nРусский 😀\n")
	w := test.NewWindow(grid)
	defer w.Close()
	w.Resize(fyne.NewSize(350, 150))
	require.False(t, grid.TabStop())
	require.False(t, grid.AccessibilityFocus())
	grid.AccessibilitySelectText(0, 2)
	require.True(t, grid.AccessibilityText().SelectionDisabled)
	require.Equal(t, grid.Size(), grid.AccessibilityText().ViewportSize)
	grid.Selectable, grid.ShowLineNumbers = true, true
	grid.Refresh()
	require.True(t, grid.AccessibilityFocus())
	grid.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	grid.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnd})
	end := grid.AccessibilityText().Caret
	grid.KeyDown(&fyne.KeyEvent{Name: desktop.KeyShiftLeft})
	grid.TypedKey(&fyne.KeyEvent{Name: fyne.KeyLeft})
	grid.KeyUp(&fyne.KeyEvent{Name: desktop.KeyShiftLeft})
	require.Equal(t, "😀", grid.SelectedText())
	require.Equal(t, end-1, grid.AccessibilityText().Caret, "caret uses rune offsets")
	grid.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	r := cache.Renderer(grid).(*textGridRenderer)
	require.True(t, r.caret.Visible())
	require.Equal(t, end, grid.AccessibilityText().Caret)
	grid.TypedShortcut(&fyne.ShortcutSelectAll{})
	grid.TypedShortcut(&fyne.ShortcutCopy{})
	require.Equal(t, "A\tB\nРусский 😀\n", fyne.CurrentApp().Clipboard().Content())
	require.True(t, grid.cellSelected(0, grid.content.lineNumberWidth()+1))
	require.False(t, grid.cellSelected(0, 0), "line number is not selected")
	grid.SetText("x")
	require.Equal(t, 1, grid.AccessibilityText().Caret)
	grid.TypedKey(&fyne.KeyEvent{Name: fyne.KeyHome})
	grid.TypedKey(&fyne.KeyEvent{Name: fyne.KeyUp})
	require.Zero(t, grid.AccessibilityText().Caret)
	grid.SetText("")
	grid.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnd})
	require.Zero(t, grid.AccessibilityText().Caret)
	grid.Selectable = false
	grid.Refresh()
	require.Empty(t, grid.SelectedText())
}

func TestTextGridReadingScroll(t *testing.T) {
	test.NewTempApp(t)
	grid := NewTextGridFromString(strings.Repeat("Long line\twith text\n", 40))
	grid.Selectable, grid.Scroll = true, fyne.ScrollBoth
	w := test.NewWindow(grid)
	defer w.Close()
	w.Resize(fyne.NewSize(130, 90))
	require.True(t, grid.AccessibilityFocus())
	grid.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyEnd, Modifier: fyne.KeyModifierControl})
	require.Greater(t, grid.scroller.Offset.Y, float32(0))
	require.True(t, cache.Renderer(grid).(*textGridRenderer).caret.Visible())
	require.Same(t, grid, w.Canvas().Focused())
	grid.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyHome, Modifier: fyne.KeyModifierControl})
	require.Zero(t, grid.scroller.Offset.Y)
	require.Zero(t, grid.AccessibilityText().Caret)
}

func TestRichTextReaderKeepsPositionAcrossWrap(t *testing.T) {
	test.NewTempApp(t)
	rich := NewRichTextFromMarkdown("# Heading\n\nA longer paragraph with Русский текст 😀 and several more words.")
	rich.Selectable, rich.Wrapping, rich.Scroll = true, fyne.TextWrapWord, fyne.ScrollVerticalOnly
	w := test.NewWindow(rich)
	defer w.Close()
	w.Resize(fyne.NewSize(420, 180))
	require.True(t, rich.AccessibilityFocus())
	rich.AccessibilitySelectText(15, 25)
	before := rich.AccessibilityText()
	for _, width := range []float32{90, 320} {
		w.Resize(fyne.NewSize(width, 180))
		current := rich.AccessibilityText()
		require.Equal(t, before.Caret, current.Caret)
		require.Equal(t, before.SelectionStart, current.SelectionStart)
		require.Equal(t, before.SelectionEnd, current.SelectionEnd)
		require.Equal(t, before.Text, current.Text)
	}
	rich.selection.TypedRune('x')
	rich.selection.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDelete})
	rich.selection.TypedShortcut(&fyne.ShortcutPaste{})
	rich.selection.TypedShortcut(&fyne.ShortcutCut{})
	require.Equal(t, before.Text, rich.AccessibilityText().Text)
}
