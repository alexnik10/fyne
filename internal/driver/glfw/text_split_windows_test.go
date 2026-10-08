//go:build accessibility && windows

package glfw

import (
	"context"
	"os/exec"
	"strconv"
	"testing"
	"time"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/internal/accessibilitydemo"
	"fyne.io/fyne/v2/test"
	"github.com/go-gl/glfw/v3.4/glfw"
	"github.com/stretchr/testify/require"
)

func TestMainLoopNativeTextAndSplit(t *testing.T) {
	previous := fyne.CurrentApp()
	runOnMain(func() { fyne.SetCurrentApp(&nativeDriverApp{App: previous}) })
	defer runOnMain(func() { fyne.SetCurrentApp(previous) })
	w := createWindow("UIA text and split")
	defer w.Close()
	var hwnd uintptr
	runOnMain(func() {
		w.window.SetContent(accessibilitydemo.NewTextSplit())
		w.window.Resize(fyne.NewSize(850, 500))
		w.window.Show()
		w.window.RequestFocus()
		w.window.updateAccessibility()
		hwnd = uintptr(unsafe.Pointer(w.view().GetWin32Window()))
	})
	press := func(key glfw.Key, mods glfw.ModifierKey) {
		w.keyPressed(nil, key, 0, glfw.Press, mods)
		w.keyPressed(nil, key, 0, glfw.Release, mods)
	}
	assertFocus := func(name string) test.AccessibilityNode {
		t.Helper()
		var nodes []test.AccessibilityNode
		runOnMain(func() {
			for _, n := range test.NewAccessibilityTree(w.canvas).Snapshot() {
				if n.Focused {
					nodes = append(nodes, n)
				}
			}
		})
		require.Len(t, nodes, 1)
		require.Equal(t, name, nodes[0].Name)
		return nodes[0]
	}
	press(glfw.KeyTab, 0)
	assertFocus("Text grid document")
	press(glfw.KeyTab, 0)
	assertFocus("Resize panes")
	press(glfw.KeyHome, 0)
	press(glfw.KeyTab, 0)
	assertFocus("Pane two document")
	press(glfw.KeyDown, 0)
	require.Greater(t, assertFocus("Pane two document").Document.Caret, 0)
	press(glfw.KeyTab, glfw.ModShift)
	assertFocus("Resize panes")
	press(glfw.KeyEnd, 0)
	press(glfw.KeyTab, glfw.ModShift)
	assertFocus("Text grid document")
	press(glfw.KeyDown, 0)
	press(glfw.KeyDown, 0)
	press(glfw.KeyHome, 0)
	w.keyPressed(nil, glfw.KeyLeftShift, 0, glfw.Press, glfw.ModShift)
	press(glfw.KeyEnd, glfw.ModShift)
	w.keyPressed(nil, glfw.KeyLeftShift, 0, glfw.Release, 0)
	info := assertFocus("Text grid document").Document
	require.Equal(t, "Русский текст 😀", string([]rune(info.Text)[info.SelectionStart:info.SelectionEnd]))
	runOnMain(func() { w.window.updateAccessibility() })
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-File", "testdata/accessibility/text_split_windows.ps1", "-WindowHandle", strconv.FormatUint(uint64(hwnd), 10))
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
}
