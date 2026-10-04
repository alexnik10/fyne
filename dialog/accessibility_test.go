package dialog

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/internal/cache"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccessibilityColorValueUpdatesModel(t *testing.T) {
	test.NewTempApp(t)
	var changed int
	channel := newColorChannel("R", 0, 255, 10, func(v int) { changed = v })
	w := test.NewWindow(channel)
	defer w.Close()
	r := cache.Renderer(channel).(*colorChannelRenderer)
	tree := test.NewAccessibilityTree(w.Canvas())
	entry, ok := tree.Node(r.entry)
	require.True(t, ok)
	assert.Equal(t, "Red", entry.Name)
	require.True(t, tree.Perform(entry.ID, test.AccessibilitySetValue, "42", 0))
	assert.Equal(t, 42, changed)
	assert.Equal(t, 42, channel.value)
	assert.False(t, tree.Perform(entry.ID, test.AccessibilitySetValue, "999", 0))
	assert.Equal(t, 42, channel.value)
}

func TestAccessibilityColorSwatchKeyboard(t *testing.T) {
	test.NewTempApp(t)
	calls := 0
	swatch := newColorButton(color.NRGBA{R: 255, A: 255}, func(color.Color) { calls++ })
	w := test.NewWindow(swatch)
	defer w.Close()
	tree := test.NewAccessibilityTree(w.Canvas())
	node, ok := tree.Node(swatch)
	require.True(t, ok)
	assert.NotEmpty(t, node.Name)
	require.True(t, tree.Perform(node.ID, test.AccessibilityFocus, "", 0))
	w.Canvas().Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	assert.Equal(t, 1, calls)
	require.True(t, tree.Perform(node.ID, test.AccessibilityActivate, "", 0))
	assert.Equal(t, 2, calls)
}
