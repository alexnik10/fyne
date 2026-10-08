//go:build accessibility && windows

package glfw

import (
	"context"
	"fmt"
	"image/color"
	"os/exec"
	"strconv"
	"testing"
	"time"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"
)

func TestMainLoopNativeScroll(t *testing.T) {
	previousApp := fyne.CurrentApp()
	runOnMain(func() { fyne.SetCurrentApp(&nativeDriverApp{App: previousApp}) })
	defer runOnMain(func() { fyne.SetCurrentApp(previousApp) })
	w := createWindow("UIA scroll regression")
	defer w.Close()
	var hwnd uintptr
	runOnMain(func() {
		extent := canvas.NewRectangle(color.Transparent)
		extent.SetMinSize(fyne.NewSize(1400, 1800))
		target := widget.NewButton("Scroll target", nil)
		target.Move(fyne.NewPos(1100, 1500))
		target.Resize(target.MinSize())
		scroll := container.NewScroll(container.NewWithoutLayout(extent, target))
		scroll.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Native viewport"})
		list := widget.NewList(func() int { return 120 }, func() fyne.CanvasObject { return widget.NewLabel("Template") }, func(int, fyne.CanvasObject) {})
		list.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Scroll list"})
		list.DescribeItem = func(i int) fyne.AccessibilityInfo { return fyne.AccessibilityInfo{Name: fmt.Sprintf("List row %d", i)} }
		children := make([]string, 120)
		for i := range children {
			children[i] = fmt.Sprintf("Tree row %d", i)
		}
		tree := widget.NewTreeWithStrings(map[string][]string{"": children})
		tree.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Scroll tree"})
		tree.DescribeNode = func(id string) fyne.AccessibilityInfo { return fyne.AccessibilityInfo{Name: id} }
		table := widget.NewTableWithHeaders(func() (int, int) { return 120, 10 }, func() fyne.CanvasObject { return widget.NewLabel("Table template") }, func(widget.TableCellID, fyne.CanvasObject) {})
		table.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Scroll table"})
		table.DescribeCell = func(id widget.TableCellID) fyne.AccessibilityInfo {
			return fyne.AccessibilityInfo{Name: fmt.Sprintf("Cell %d %d", id.Row, id.Col)}
		}
		table.StickyRowCount, table.StickyColumnCount = 1, 1
		tabs := container.NewAppTabs(container.NewTabItem("Viewport", scroll), container.NewTabItem("List", list), container.NewTabItem("Tree", tree), container.NewTabItem("Table", table))
		fit := widget.NewButton("Fit viewport content", func() {
			scroll.Content = widget.NewLabel("Content fits")
			scroll.Refresh()
		})
		modal := widget.NewButton("Open scroll modal", func() {
			var popup *widget.PopUp
			popup = widget.NewModalPopUp(widget.NewButton("Close scroll modal", func() { popup.Hide() }), w.canvas)
			popup.Show()
		})
		w.window.SetContent(container.NewBorder(nil, container.NewHBox(fit, modal), nil, nil, tabs))
		w.window.Resize(fyne.NewSize(500, 350))
		w.window.Show()
		w.window.RequestFocus()
		w.window.updateAccessibility()
		hwnd = uintptr(unsafe.Pointer(w.view().GetWin32Window()))
	})
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-File", "testdata/accessibility/scroll_windows.ps1", "-WindowHandle", strconv.FormatUint(uint64(hwnd), 10))
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
}
