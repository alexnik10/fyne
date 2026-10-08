//go:build accessibility && windows

package glfw

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/internal/accessibilitydemo"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/go-gl/glfw/v3.4/glfw"
	"github.com/stretchr/testify/require"
)

func TestMainLoopNativeDialogs(t *testing.T) {
	previous := fyne.CurrentApp()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "report.txt"), []byte("example"), 0600))
	runOnMain(func() { fyne.SetCurrentApp(&nativeDriverApp{App: previous}) })
	defer runOnMain(func() { fyne.SetCurrentApp(previous) })
	w := createWindow("UIA dialogs")
	defer w.Close()
	var hwnd uintptr
	var oldFolder string
	var oldView int
	runOnMain(func() {
		oldFolder = previous.Preferences().String("fyne:fileDialogLastFolder")
		oldView = previous.Preferences().Int("fyne:fileDialogViewLayout")
		previous.Preferences().SetString("fyne:fileDialogLastFolder", storage.NewFileURI(dir).String())
		previous.Preferences().SetInt("fyne:fileDialogViewLayout", 2)
		status := widget.NewLabel("Ready")
		status.SetAccessibilityLiveSetting(fyne.AccessibilityLivePolite)
		w.window.SetContent(container.NewBorder(nil, status, nil, nil, accessibilitydemo.NewDialogs(w.window, status)))
		w.window.Resize(fyne.NewSize(850, 650))
		w.window.Show()
		w.window.RequestFocus()
		w.window.updateAccessibility()
		hwnd = uintptr(unsafe.Pointer(w.view().GetWin32Window()))
	})
	defer runOnMain(func() {
		previous.Preferences().SetString("fyne:fileDialogLastFolder", oldFolder)
		previous.Preferences().SetInt("fyne:fileDialogViewLayout", oldView)
	})
	press := func(key glfw.Key, mods glfw.ModifierKey) {
		w.keyPressed(nil, key, 0, glfw.Press, mods)
		w.keyPressed(nil, key, 0, glfw.Release, mods)
	}
	named := func(name string, focus bool) test.AccessibilityNode {
		t.Helper()
		var found test.AccessibilityNode
		runOnMain(func() {
			tree := test.NewAccessibilityTree(w.canvas)
			for _, n := range tree.Snapshot() {
				if n.Name == name {
					found = n
					if focus {
						require.True(t, tree.Perform(n.ID, test.AccessibilityFocus, "", 0))
					}
					break
				}
			}
		})
		require.NotZero(t, found.ID, "missing %q", name)
		return found
	}
	checkFocus := func(name string) { t.Helper(); require.True(t, named(name, false).Focused, "%q is not focused", name) }
	press(glfw.KeyTab, 0)
	checkFocus("Choose colour")
	press(glfw.KeyTab, 0)
	checkFocus("Open file")
	press(glfw.KeyEnter, 0)
	named("File options", true)
	press(glfw.KeyEnter, 0)
	checkFocus("Show Hidden Files")
	press(glfw.KeyEscape, 0)
	checkFocus("File options")
	press(glfw.KeyEnter, 0)
	press(glfw.KeyTab, glfw.ModShift)
	checkFocus("List view")
	press(glfw.KeyTab, 0)
	checkFocus("File options")
	press(glfw.KeyEnter, 0)
	press(glfw.KeyTab, 0)
	require.False(t, named("File options", false).Focused)
	named("File options", true)
	press(glfw.KeyEnter, 0)
	press(glfw.KeyKPEnter, 0)
	checkFocus("File options")
	named("List view", true)
	require.False(t, named("List view", false).Checked)
	press(glfw.KeyEnter, 0)
	require.True(t, named("List view", false).Checked)
	press(glfw.KeyKPEnter, 0)
	require.False(t, named("List view", false).Checked)
	runOnMain(func() { w.window.updateAccessibility() })
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-File", "testdata/accessibility/dialogs_windows.ps1", "-WindowHandle", strconv.FormatUint(uint64(hwnd), 10))
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
	checkFocus("Open file")
	press(glfw.KeyEnter, 0)
	named("Cancel", true)
	press(glfw.KeyKPEnter, 0)
	checkFocus("Open file")
	require.Equal(t, "Selected file: report.txt", named("Open file", false).Description)
	press(glfw.KeyTab, 0)
	checkFocus("Selected file")
	press(glfw.KeyTab, 0)
	checkFocus("Example file icon: report.txt")
	named("Choose colour", true)
	press(glfw.KeyEnter, 0)
	named("Red, #f44336", true)
	press(glfw.KeyKPEnter, 0)
	checkFocus("Choose colour")
}
