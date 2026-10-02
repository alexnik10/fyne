package accessibility_test

import (
	"errors"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/internal/accessibility"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditableTextSnapshotAndCommands(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	entry := widget.NewEntry()
	entry.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Text"})
	win := test.NewWindow(entry)
	defer win.Close()
	entry.Resize(fyne.NewSize(240, 40))
	entry.SetText("A😀Бc")
	entry.AccessibilitySelectText(1, 3)
	var tree accessibility.Tree
	roots := []accessibility.Root{{Object: entry}}
	node := nodeNamed(t, tree.Build(roots, entry), "Text")
	require.NotNil(t, node.Document)
	assert.Equal(t, "A😀Бc", node.Document.Text)
	assert.Equal(t, 1, node.Document.SelectionStart)
	assert.Equal(t, 3, node.Document.SelectionEnd)
	assert.Equal(t, 3, node.Document.Caret)
	require.Len(t, node.Document.Positions, 5) // runes, not UTF-16 or bytes
	assert.Greater(t, node.Document.Positions[2].Position.X, node.Document.Positions[1].Position.X)
	assert.True(t, tree.SelectText(node.ID, 2, 2))
	assert.Equal(t, 2, entry.CursorTextOffset())
	assert.Empty(t, entry.SelectedText())
	assert.False(t, tree.SelectText(node.ID, 0, 5))
	assert.False(t, tree.SelectText(node.ID, -1, 2))
	assert.False(t, tree.SelectText(node.ID, 3, 1))
	entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyLeft})
	moved := nodeNamed(t, tree.Build(roots, entry), "Text")
	assert.Equal(t, 1, moved.Document.Caret)
	assert.Equal(t, node.Document.Revision, moved.Document.Revision) // arrows aren't edits
	entry.TypedRune('Ж')
	changed := nodeNamed(t, tree.Build(roots, entry), "Text")
	assert.Greater(t, changed.Document.Revision, moved.Document.Revision)
	assert.Equal(t, "AЖ😀Бc", changed.Document.Text)
	entry.Password = true
	entry.Refresh()
	masked := nodeNamed(t, tree.Build(roots, entry), "Text")
	assert.Empty(t, masked.Text)
	assert.Equal(t, "•••••", masked.Document.Text)
	assert.Equal(t, []int{0, 5}, masked.Document.WordBoundaries)
	assert.Equal(t, 2, masked.Document.Caret)
	assert.True(t, masked.Protected)
	entry.Disable()
	tree.Build(roots, nil)
	assert.False(t, tree.SelectText(node.ID, 0, 1))
	entry.Enable()
	tree.Build([]accessibility.Root{{Object: entry, Suppressed: true}}, nil)
	assert.False(t, tree.SelectText(node.ID, 0, 1))
}

func TestTextSelectionDoesNotEditAndScrollDoesNotMoveCaret(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	entry := widget.NewMultiLineEntry()
	win := test.NewWindow(entry)
	defer win.Close()
	entry.Resize(fyne.NewSize(100, 50))
	entry.SetText("first line\nsecond line\nthird line\nlast line")
	changes := 0
	entry.OnChanged = func(string) { changes++ }
	entry.AccessibilitySelectText(11, 17)
	assert.Equal(t, "second", entry.SelectedText())
	before := entry.AccessibilityText()
	entry.AccessibilityScrollText(0, 1, true)
	after := entry.AccessibilityText()
	assert.Equal(t, before.Caret, after.Caret)
	assert.Equal(t, before.SelectionStart, after.SelectionStart)
	assert.Equal(t, before.SelectionEnd, after.SelectionEnd)
	assert.Equal(t, 0, changes)
	entry.AccessibilitySelectText(-2, 1000)
	assert.Equal(t, entry.Text, entry.SelectedText())
}

func TestTextRevisionForIdenticalReplacement(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	entry := widget.NewEntry()
	entry.SetText("same")
	entry.AccessibilitySelectText(0, 4)
	before := entry.AccessibilityText()
	clipboard := test.NewClipboard()
	clipboard.SetContent("same")
	entry.TypedShortcut(&fyne.ShortcutPaste{Clipboard: clipboard})
	after := entry.AccessibilityText()
	assert.Equal(t, before.Text, after.Text)
	assert.Greater(t, after.Revision, before.Revision)
	assert.Equal(t, 4, after.Caret)
	assert.Equal(t, after.SelectionStart, after.SelectionEnd)
}

func TestSliderStringValueTracksKeyboardRange(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	slider := widget.NewSlider(0, 10)
	slider.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Volume"})
	var tree accessibility.Tree
	roots := []accessibility.Root{{Object: slider}}
	before := nodeNamed(t, tree.Build(roots, slider), "Volume")
	slider.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	after := nodeNamed(t, tree.Build(roots, slider), "Volume")
	assert.True(t, after.Value && after.Range)
	assert.Equal(t, "0", before.Text)
	assert.Equal(t, "1", after.Text)
	assert.Equal(t, float64(1), after.Number)
	slider.AccessibilitySetValue("3")
	assert.Equal(t, float64(3), slider.Value)
	for _, value := range []string{"bad", "NaN", "+Inf", "11"} {
		slider.AccessibilitySetValue(value)
		assert.Equal(t, float64(3), slider.Value)
	}
}

func TestFormHintSurvivesValidInput(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	entry := widget.NewEntry()
	entry.Validator = func(value string) error {
		if value != "user@example.org" {
			return errors.New("invalid address")
		}
		return nil
	}
	item := widget.NewFormItem("Email", entry)
	item.HintText = "For example user@example.org"
	form := widget.NewForm(item)
	win := test.NewWindow(form)
	defer win.Close()
	var tree accessibility.Tree
	roots := []accessibility.Root{{Object: form}}
	entry.SetText("bad")
	invalid := nodeNamed(t, tree.Build(roots, nil), "Email")
	assert.True(t, invalid.Invalid)
	assert.Equal(t, "invalid address", invalid.Description)
	entry.SetText("user@example.org")
	valid := nodeNamed(t, tree.Build(roots, nil), "Email")
	assert.False(t, valid.Invalid)
	assert.Equal(t, item.HintText, valid.Description)
}
