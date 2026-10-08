package dialog

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/internal/cache"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
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

func TestAccessibilityDialogKeyboard(t *testing.T) {
	test.NewTempApp(t)
	if fyne.CurrentDevice().IsMobile() {
		t.Skip("desktop keyboard menus take focus automatically; mobile popups do not")
	}
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "folder"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "report.txt"), []byte("example"), 0o600))
	location, err := storage.ListerForURI(storage.NewFileURI(dir))
	require.NoError(t, err)
	launch := widget.NewButton("Open file", nil)
	w := test.NewWindow(launch)
	defer w.Close()
	w.Resize(fyne.NewSize(800, 600))
	var selected string
	d := NewFileOpen(func(r fyne.URIReadCloser, err error) {
		require.NoError(t, err)
		if r != nil {
			selected = r.URI().Name()
			require.NoError(t, r.Close())
		}
	}, w)
	d.SetLocation(location)
	launch.OnTapped = d.Show
	c := w.Canvas()
	c.Focus(launch)
	launch.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	require.NotNil(t, d.dialog)
	f := d.dialog
	tree := test.NewAccessibilityTree(c)
	focus := func(name string) {
		t.Helper()
		for _, n := range tree.Snapshot() {
			if n.Name == name && n.Focusable {
				require.True(t, tree.Perform(n.ID, test.AccessibilityFocus, "", 0))
				return
			}
		}
		t.Fatalf("focus target %q missing", name)
	}
	key := func(k fyne.KeyName) { c.Focused().TypedKey(&fyne.KeyEvent{Name: k}) }
	focus("File options")
	owner := c.Focused()
	key(fyne.KeyReturn)
	menu, ok := c.Focused().(*widget.PopUpMenu)
	require.True(t, ok)
	item := menu.AccessibilityActiveDescendant()
	n, ok := tree.Node(item)
	require.True(t, ok)
	require.Equal(t, "Show Hidden Files", n.Name)
	require.True(t, n.Toggle)
	require.False(t, n.Checked)
	key(fyne.KeyEscape)
	require.Same(t, owner, c.Focused())
	require.Len(t, c.Overlays().List(), 1)
	for _, k := range []fyne.KeyName{fyne.KeyEnter, fyne.KeySpace} {
		key(fyne.KeyReturn)
		key(k)
		require.Same(t, owner, c.Focused())
		require.Len(t, c.Overlays().List(), 1)
	}
	require.False(t, f.showHidden)
	key(fyne.KeyReturn)
	key(fyne.KeyTab)
	require.Len(t, c.Overlays().List(), 1)
	require.NotSame(t, owner, c.Focused())
	c.Focus(f.toggleViewButton)
	for _, k := range []fyne.KeyName{fyne.KeyReturn, fyne.KeyEnter} {
		before, _ := tree.Node(f.toggleViewButton)
		key(k)
		after, ok := tree.Node(f.toggleViewButton)
		require.True(t, ok)
		require.Equal(t, before.ID, after.ID)
		require.Equal(t, "List view", after.Name)
		require.True(t, after.Toggle)
		require.NotEqual(t, before.Checked, after.Checked)
		require.Same(t, f.toggleViewButton, c.Focused())
	}
	for _, view := range []ViewLayout{GridView, ListView} {
		f.setView(view)
		focus("folder")
		key(fyne.KeyReturn)
		require.Equal(t, "folder", f.dir.Name())
		f.setLocation(location)
		focus("report.txt")
		key(fyne.KeyEnter)
		require.Equal(t, "report.txt", f.selected.Name())
	}
	focus("Open")
	key(fyne.KeyReturn)
	require.Equal(t, "report.txt", selected)
	require.Empty(t, c.Overlays().List())
	require.Same(t, launch, c.Focused())
}

func TestAccessibilityPaletteNames(t *testing.T) {
	test.NewTempApp(t)
	for hex, name := range map[string]string{
		"#f44336": "Red", "#ff9800": "Orange", "#ffeb3b": "Yellow", "#8bc34a": "Green",
		"#296ff6": "Blue", "#9c27b0": "Purple", "#795548": "Brown", "#ffffff": "White",
		"#cccccc": "Very light grey", "#aaaaaa": "Light grey", "#808080": "Grey",
		"#555555": "Dark grey", "#333333": "Very dark grey", "#000000": "Black",
	} {
		swatch := newColorButton(stringsToColors(hex)[0], nil)
		require.Equal(t, name+", "+hex, swatch.AccessibilityLabel())
	}
	custom := newColorButton(color.NRGBA{R: 12, G: 34, B: 56, A: 128}, nil)
	require.Contains(t, custom.AccessibilityLabel(), "Red 12, green 34, blue 56, opacity 50%")
	preview := newColorPreview(color.NRGBA{R: 12, G: 34, B: 56, A: 128})
	value, readonly, protected := preview.AccessibilityValue()
	require.Contains(t, value, custom.AccessibilityLabel())
	require.True(t, readonly)
	require.False(t, protected)
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
