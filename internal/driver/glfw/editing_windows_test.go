//go:build accessibility && windows

package glfw

import (
	"context"
	"errors"
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

func TestMainLoopNativeEditing(t *testing.T) {
	previousApp := fyne.CurrentApp()
	runOnMain(func() { fyne.SetCurrentApp(&nativeDriverApp{App: previousApp}) })
	defer runOnMain(func() { fyne.SetCurrentApp(previousApp) })
	w := createWindow("UIA editing regression")
	defer w.Close()
	var hwnd uintptr
	runOnMain(func() {
		values := make([]string, 120)
		for i := range values {
			values[i] = fmt.Sprintf("Value %d", i)
		}
		table := widget.NewTable(func() (int, int) { return len(values), 2 }, func() fyne.CanvasObject { return widget.NewLabel("Cell template") }, func(id widget.TableCellID, obj fyne.CanvasObject) { obj.(*widget.Label).SetText(values[id.Row]) })
		table.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Editable native table"})
		table.DescribeCell = func(id widget.TableCellID) fyne.AccessibilityInfo {
			return fyne.AccessibilityInfo{Name: fmt.Sprintf("Cell %d %d", id.Row, id.Col)}
		}
		table.CellValue = func(id widget.TableCellID) (string, bool) { return values[id.Row], id.Col == 0 }
		table.OnCellChanged = func(id widget.TableCellID, value string) error {
			if value == "" {
				return errors.New("required value")
			}
			values[id.Row] = value
			return nil
		}
		rich := widget.NewRichTextEntryFromMarkdown("Plain **bold** and *italic*")
		rich.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Native rich editor"})
		preview := widget.NewRichTextFromMarkdown("# Heading\n\nPlain **bold**")
		preview.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Native document"})
		w.window.SetContent(container.NewGridWithColumns(3, table, rich, preview))
		w.window.Resize(fyne.NewSize(850, 350))
		w.window.Show()
		w.window.RequestFocus()
		w.window.updateAccessibility()
		hwnd = uintptr(unsafe.Pointer(w.view().GetWin32Window()))
	})
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-File", "testdata/accessibility/editing_windows.ps1", "-WindowHandle", strconv.FormatUint(uint64(hwnd), 10))
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
}
