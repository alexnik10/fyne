package widget

import (
	"strings"
	"testing"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccessibilityRichTextDocument(t *testing.T) {
	rich := NewRichTextFromMarkdown("# Title\n\nPlain **bold** and *italic*.\n\n* First\n* Second\n\nРусский 😀")
	rich.Selectable = true
	rich.Wrapping = fyne.TextWrapWord
	w := test.NewWindow(rich)
	defer w.Close()
	w.Resize(fyne.NewSize(220, 300))
	info := rich.AccessibilityText()
	assert.Equal(t, "Title\nPlain bold and italic.\n• First\n• Second\nРусский 😀", info.Text)
	assert.True(t, info.ReadOnly)
	assert.False(t, info.SelectionDisabled)
	assert.Len(t, info.Positions, utf8.RuneCountInString(info.Text)+1)
	for i, r := range []rune(info.Text) {
		if r == '\n' && i > 0 {
			assert.Equal(t, info.Positions[i-1].Line, info.Positions[i].Line, "paragraph separator belongs to the preceding line")
			assert.Greater(t, info.Positions[i+1].Line, info.Positions[i].Line)
		}
	}
	require.NotEmpty(t, info.Runs)
	assert.Equal(t, 1, info.Runs[0].HeadingLevel)
	previous := 0
	for _, run := range info.Runs {
		assert.Equal(t, previous, run.Start)
		previous = run.End
		if string([]rune(info.Text)[run.Start:run.End]) == "bold" {
			assert.True(t, run.Style.Bold)
		}
	}
	assert.Equal(t, utf8.RuneCountInString(info.Text), previous)
	start := strings.Index(info.Text, "bold")
	rich.AccessibilitySelectText(start, start+4)
	assert.Equal(t, "bold", rich.SelectedText())
	selected := rich.AccessibilityText()
	assert.Equal(t, start, selected.SelectionStart)
	assert.Equal(t, start+4, selected.SelectionEnd)
	rich.selection.TypedShortcut(&fyne.ShortcutCopy{})
	assert.Equal(t, "bold", fyne.CurrentApp().Clipboard().Content())
	tree := test.NewAccessibilityTree(w.Canvas())
	node, ok := tree.Node(rich)
	require.True(t, ok)
	require.True(t, tree.Perform(node.ID, test.AccessibilityFocus, "", 0))
	node, _ = tree.Node(rich)
	assert.True(t, node.Focused)
	assert.Empty(t, tree.Issues())
	rich.Selectable = false
	rich.Refresh()
	node, _ = tree.Node(rich)
	assert.True(t, node.Document.SelectionDisabled)
	assert.False(t, tree.SelectText(node.ID, 0, 2))
}

func TestAccessibilityRichTextEntryFormattingAndReplacement(t *testing.T) {
	e := NewRichTextEntryFromMarkdown("Plain **bold** 😀")
	w := test.NewWindow(e)
	defer w.Close()
	w.Resize(fyne.NewSize(250, 120))
	info := e.AccessibilityText()
	assert.Equal(t, e.Text, info.Text)
	require.Len(t, info.Runs, 3)
	assert.True(t, info.Runs[1].Style.Bold)
	e.AccessibilitySelectText(6, 10)
	assert.Equal(t, "bold", e.SelectedText())
	before := e.Text
	e.SetStyleForSelection(RichTextStyleEmphasis)
	assert.Equal(t, before, e.Text)
	assert.True(t, e.AccessibilityText().Runs[1].Style.Italic)
	assert.False(t, info.Runs[1].Style.Italic, "published snapshots must not alias the model")
	changes := 0
	e.OnChanged = func(string) { changes++ }
	e.AccessibilitySetValue("Replacement 😀")
	assert.Equal(t, "Replacement 😀", e.Text)
	assert.Equal(t, e.Text, e.richProvider().String())
	assert.Equal(t, 1, changes)
	require.Len(t, e.AccessibilityText().Runs, 1)
	assert.False(t, e.AccessibilityText().Runs[0].Style.Bold)
	e.Disable()
	e.AccessibilitySetValue("must not apply")
	assert.Equal(t, "Replacement 😀", e.Text)
	e.Password = true
	protected := e.AccessibilityText()
	assert.NotContains(t, protected.Text, "Replacement")
	assert.Empty(t, protected.Runs)
}

func TestAccessibilityRichTextWrappedParagraphsAndEmpty(t *testing.T) {
	rich := NewRichTextWithText("Hello world and a longer line\nLast\n")
	rich.Wrapping = fyne.TextWrapWord
	rich.Resize(fyne.NewSize(80, 250))
	assert.Equal(t, rich.String(), rich.AccessibilityText().Text, "soft wraps must not become paragraphs")
	empty := NewRichText()
	empty.Resize(fyne.NewSize(100, 50))
	info := empty.AccessibilityText()
	assert.Empty(t, info.Text)
	assert.Len(t, info.Positions, 1)
	assert.Len(t, info.Runs, 1)
}
