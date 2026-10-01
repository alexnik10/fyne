package widget

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

func TestEntryWordBoundaries(t *testing.T) {
	for _, tc := range []struct {
		text  string
		stops []int
	}{
		{"", []int{0}},
		{"user@example.org", []int{0, 4, 5, 12, 13, 16}},
		{"user@example.org/", []int{0, 4, 5, 12, 13, 16, 17}},
		{"\"user@example.org\"/.", []int{0, 1, 5, 6, 13, 14, 17, 20}},
		{"foo_bar", []int{0, 7}},
		{"one  two", []int{0, 5, 8}},
		{"  one\ttwo", []int{0, 2, 6, 9}},
		{"a\nbc", []int{0, 1, 2, 4}},
		{"тест@домен.рф", []int{0, 4, 5, 10, 11, 13}},
		{"😀a@b", []int{0, 2, 3, 4}},
		{"  ", []int{0, 2}},
	} {
		t.Run(tc.text, func(t *testing.T) {
			assert.Equal(t, tc.stops, entryWordBoundaries(tc.text))
		})
	}
}

func TestEntryWindowsWordNavigationMatchesAccessibleWords(t *testing.T) {
	entry := NewEntry()
	entry.SetText("user@example.org")
	entry.AccessibilitySelectText(0, 0)
	stops := []int{0, 4, 5, 12, 13, 16}
	assert.Equal(t, stops, entry.AccessibilityText().WordBoundaries)
	move := func(right bool) {
		if runtime.GOOS == "windows" {
			key := fyne.KeyLeft
			if right {
				key = fyne.KeyRight
			}
			entry.TypedShortcut(&desktop.CustomShortcut{KeyName: key, Modifier: fyne.KeyModifierControl})
		} else {
			// Exercise the same Windows implementation on non-Windows test hosts.
			entry.moveWordWindows(right)
		}
	}
	for _, stop := range stops[1:] {
		move(true)
		assert.Equal(t, stop, entry.CursorTextOffset())
	}
	move(true)
	assert.Equal(t, 16, entry.CursorTextOffset()) // end is stable
	for i := len(stops) - 2; i >= 0; i-- {
		move(false)
		assert.Equal(t, stops[i], entry.CursorTextOffset())
	}
	move(false)
	assert.Zero(t, entry.CursorTextOffset())

	// Starting at punctuation must reach the word that used to be skipped.
	entry.AccessibilitySelectText(4, 4)
	move(true)
	require.Equal(t, 5, entry.CursorTextOffset())
	info := entry.AccessibilityText()
	assert.Equal(t, "example", string([]rune(info.Text)[info.WordBoundaries[2]:info.WordBoundaries[3]]))

	entry.Password = true
	entry.Refresh()
	entry.AccessibilitySelectText(0, 0)
	assert.Equal(t, []int{0, 16}, entry.AccessibilityText().WordBoundaries)
	move(true)
	assert.Equal(t, 16, entry.CursorTextOffset())
	move(false)
	assert.Zero(t, entry.CursorTextOffset())
}

func TestEntryWindowsWordSelectionAndWrapping(t *testing.T) {
	entry := NewMultiLineEntry()
	entry.Wrapping = fyne.TextWrapBreak
	entry.Resize(fyne.NewSize(65, 100))
	entry.SetText("user@example.org\nnext")
	entry.AccessibilitySelectText(5, 5)
	if runtime.GOOS == "windows" {
		entry.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyRight, Modifier: fyne.KeyModifierControl | fyne.KeyModifierShift})
	} else {
		entry.sel.selecting = true
		entry.moveWordWindows(true)
	}
	assert.Equal(t, "example", entry.SelectedText())
	assert.Equal(t, 12, entry.CursorTextOffset())
	entry.AccessibilitySelectText(12, 12)
	entry.sel.selecting = true
	entry.moveWordWindows(false)
	assert.Equal(t, "example", entry.SelectedText())
	assert.Equal(t, 5, entry.CursorTextOffset())
	entry.AccessibilitySelectText(16, 16)
	entry.moveWordWindows(true)
	assert.Equal(t, 17, entry.CursorTextOffset()) // cross a hard line break
	entry.moveWordWindows(false)
	assert.Equal(t, 16, entry.CursorTextOffset())
}
