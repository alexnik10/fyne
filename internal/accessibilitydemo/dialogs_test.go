package accessibilitydemo

import (
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"
)

func TestDialogsSelectedFileAndCancellation(t *testing.T) {
	a := test.NewTempApp(t)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "report.txt"), []byte("example"), 0600))
	a.Preferences().SetString("fyne:fileDialogLastFolder", storage.NewFileURI(dir).String())
	w := test.NewWindow(nil)
	defer w.Close()
	status := widget.NewLabel("Ready")
	status.SetAccessibilityLiveSetting(fyne.AccessibilityLivePolite)
	w.SetContent(NewDialogs(w, status))
	w.Resize(fyne.NewSize(850, 650))
	c := w.Canvas()
	tree := test.NewAccessibilityTree(c)
	named := func(name string) test.AccessibilityNode {
		t.Helper()
		for _, n := range tree.Snapshot() {
			if n.Name == name {
				return n
			}
		}
		t.Fatalf("missing %q", name)
		return test.AccessibilityNode{}
	}
	focus := func(name string) {
		n := named(name)
		require.True(t, tree.Perform(n.ID, test.AccessibilityFocus, "", 0))
	}
	key := func(k fyne.KeyName) { c.Focused().TypedKey(&fyne.KeyEvent{Name: k}) }
	c.FocusNext()
	require.Equal(t, "Choose colour", c.Focused().(fyne.Accessible).AccessibilityLabel())
	c.FocusNext()
	open := c.Focused()
	require.Equal(t, "No file selected", named("Open file").Description)
	key(fyne.KeyReturn)
	focus("report.txt")
	key(fyne.KeyReturn)
	focus("Open")
	key(fyne.KeyEnter)
	require.Same(t, open, c.Focused())
	require.Equal(t, "Selected file: report.txt", named("Open file").Description)
	require.Equal(t, "Selected file: report.txt", status.Text)
	c.FocusNext()
	n := named("Selected file")
	require.True(t, n.Focused && n.ReadOnly)
	require.Contains(t, n.Document.Text, "report.txt")
	key(fyne.KeyEnd)
	c.Focused().(fyne.Shortcutable).TypedShortcut(&fyne.ShortcutSelectAll{})
	c.Focused().(fyne.Shortcutable).TypedShortcut(&fyne.ShortcutCopy{})
	require.Equal(t, "Selected file: report.txt", a.Clipboard().Content())
	c.FocusPrevious()
	key(fyne.KeySpace)
	focus("Cancel")
	key(fyne.KeyReturn)
	require.Same(t, open, c.Focused())
	require.Equal(t, "Selected file: report.txt", named("Open file").Description)
	require.Equal(t, "Selected file: report.txt", named("Selected file").Document.Text)
	c.FocusNext()
	c.FocusNext()
	require.True(t, named("Example file icon: report.txt").Focused)
	require.Equal(t, "Example file icon", named("report.txt").Description)
}
