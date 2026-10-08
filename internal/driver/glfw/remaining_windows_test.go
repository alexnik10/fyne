//go:build accessibility && windows

package glfw

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"testing"
	"time"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
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
		grid := widget.NewGridWrap(func() int { return 1000 }, func() fyne.CanvasObject { return widget.NewLabel("Record 0000") }, func(i int, o fyne.CanvasObject) { o.(*widget.Label).SetText(fmt.Sprintf("Record %d", i)) })
		grid.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Native wrapping grid"})
		grid.DescribeItem = func(i int) fyne.AccessibilityInfo { return fyne.AccessibilityInfo{Name: fmt.Sprintf("Record %d", i)} }
		checked := false
		grid.ItemElements = func(i int) []fyne.AccessibilityElement {
			check := widget.NewCheck(fmt.Sprintf("Flag %d", i), func(v bool) { checked = v; grid.Refresh() })
			check.Checked = checked
			return []fyne.AccessibilityElement{{Key: "flag", Object: check}}
		}
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
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-File", "testdata/accessibility/remaining_windows.ps1", "-WindowHandle", strconv.FormatUint(uint64(hwnd), 10))
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
}
