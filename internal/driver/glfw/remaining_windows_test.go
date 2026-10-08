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
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/internal/accessibilitydemo"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/go-gl/glfw/v3.4/glfw"
	"github.com/stretchr/testify/require"
)

func TestMainLoopNativeRemainingControls(t *testing.T) {
	previous := fyne.CurrentApp()
	runOnMain(func() { fyne.SetCurrentApp(&nativeDriverApp{App: previous}) })
	defer runOnMain(func() { fyne.SetCurrentApp(previous) })
	w := createWindow("UIA remaining controls")
	defer w.Close()
	var hwnd uintptr
	runOnMain(func() {
		progress := widget.NewProgressBar()
		progress.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Native progress"})
		progress.SetValue(0.25)
		calendar := widget.NewCalendar(time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC), nil)
		calendar.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Native calendar"})
		combo := widget.NewSelectEntry([]string{"First", "Second"})
		combo.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Native editable choice"})
		text := widget.NewTextGridFromString("Read only\nРусский 😀")
		text.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Native text grid"})
		gridStatus := widget.NewLabel("Grid ready")
		grid := container.NewBorder(nil, gridStatus, nil, nil, accessibilitydemo.NewGrid(gridStatus))
		accordion := widget.NewAccordion(widget.NewAccordionItem("Native section", widget.NewLabel("Section content")))
		status := widget.NewLabel("Native ready")
		status.SetAccessibilityLiveSetting(fyne.AccessibilityLivePolite)
		save := widget.NewButton("Native save", func() { status.SetText("Native saved") })
		w.window.SetContent(container.NewGridWithColumns(3, container.NewVBox(progress, combo, accordion, text, save, status), calendar, grid))
		w.window.Resize(fyne.NewSize(1000, 550))
		w.window.Show()
		w.window.RequestFocus()
		w.window.updateAccessibility()
		hwnd = uintptr(unsafe.Pointer(w.view().GetWin32Window()))
	})
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-File", "testdata/accessibility/remaining_windows.ps1", "-WindowHandle", strconv.FormatUint(uint64(hwnd), 10))
	output, err := cmd.CombinedOutput()
	if err != nil {
		runOnMain(func() {
			t.Logf("Canvas focus after UIA failure: %T", w.canvas.Focused())
			for _, node := range test.NewAccessibilityTree(w.canvas).Snapshot() {
				if node.Focused || node.Name == "Enable Record 499" || node.Name == "Keyed wrapping grid" {
					t.Logf("Semantic state: %+v", node)
				}
			}
		})
	}
	require.NoError(t, err, "%s", output)
	checkNativeGridKeyboard(t, w)
	client := exec.CommandContext(ctx, "testdata/accessibility/live-client.exe", strconv.FormatUint(uint64(hwnd), 10))
	output, err = client.CombinedOutput()
	require.NoError(t, err, "%s", output)
}

func checkNativeGridKeyboard(t *testing.T, w *safeWindow) {
	t.Helper()
	grid := w.Canvas().Focused()
	require.NotNil(t, grid)
	press := func(key glfw.Key, mods glfw.ModifierKey) {
		w.keyPressed(nil, key, 0, glfw.Press, mods)
		w.keyPressed(nil, key, 0, glfw.Release, mods)
	}
	press(glfw.KeyTab, 0)
	button, ok := w.Canvas().Focused().(*widget.Button)
	require.True(t, ok, "Tab must leave the grid without focusing pooled checks")
	require.Equal(t, "Swap first and last", button.Text)
	press(glfw.KeyTab, glfw.ModShift)
	require.Same(t, grid, w.Canvas().Focused())
	for _, checked := range []bool{true, false} {
		press(glfw.KeySpace, 0)
		var focused []test.AccessibilityNode
		runOnMain(func() {
			for _, node := range test.NewAccessibilityTree(w.canvas).Snapshot() {
				if node.Focused {
					focused = append(focused, node)
				}
			}
		})
		require.Len(t, focused, 1)
		require.Equal(t, "Enable Record 499", focused[0].Name)
		require.Equal(t, fyne.AccessibleRoleCheck, focused[0].Role)
		require.Equal(t, checked, focused[0].Checked)
		require.Same(t, grid, w.Canvas().Focused())
	}
}
