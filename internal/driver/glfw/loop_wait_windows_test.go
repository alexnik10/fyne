//go:build accessibility && windows && !no_glfw && !mobile

package glfw

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/stretchr/testify/require"
)

func TestMainLoopNativeWorkWithoutFrames(t *testing.T) {
	if os.Getenv("FYNE_TEST_NATIVE_WAIT_ONLY") != "1" {
		executable, err := os.Executable()
		require.NoError(t, err)
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, executable, "-test.run=^TestMainLoopNativeWorkWithoutFrames$", "-test.timeout=15s")
		cmd.Env = append(os.Environ(), "FYNE_TEST_NATIVE_WAIT_ONLY=1")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "%s", output)
		return
	}

	// This process never starts mainLoopTickEvents. Use the production queue,
	// producer and native wait, with no frame or shutdown wake as a fallback.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	runtime.GOMAXPROCS(1) // force the consumer to exhaust Out before the relay runs
	running.Store(true)
	const count = 128 // exceeds both bounded channels inside funcQueue
	var order []int
	queued := make(chan struct{})
	completed := make(chan struct{})
	go func() {
		for i := 0; i < count; i++ {
			value := i
			runOnMainWithWait(func() { order = append(order, value) }, false)
		}
		close(queued)
		runOnMain(func() { order = append(order, count) })
		close(completed)
	}()
	<-queued
	for i := 0; i <= count; i++ {
		event, work := nextMainLoopEvent(nil, funcQueue.Out(), nil)
		require.Equal(t, mainLoopWork, event)
		work.f()
		if work.done != nil {
			work.done <- struct{}{}
		}
	}
	<-completed
	for i, value := range order {
		require.Equal(t, i, value)
	}
	require.Len(t, order, count+1)
	funcQueue.Close()
}

type nativeSemanticComposite struct {
	widget.BaseWidget
	content fyne.CanvasObject
}

func newNativeSemanticComposite(content fyne.CanvasObject) *nativeSemanticComposite {
	c := &nativeSemanticComposite{content: content}
	c.ExtendBaseWidget(c)
	return c
}

func (c *nativeSemanticComposite) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.content)
}

func TestMainLoopNativeInvoke(t *testing.T) {
	w := createWindow("UIA Invoke queue regression")
	defer w.Close()
	events := make(chan string, 4)
	var hwnd uintptr
	runOnMain(func() {
		var popup *widget.PopUp
		closeButton := widget.NewButton("Close queue test dialog", func() {
			popup.Hide()
			events <- "close"
		})
		popup = widget.NewModalPopUp(newNativeSemanticComposite(closeButton), w.canvas)
		button := widget.NewButton("Open queue test dialog", func() {
			popup.Show()
			events <- "open"
			// Exercise another asynchronous enqueue from an Invoke callback.
			runOnMainWithWait(func() { events <- "queued" }, false)
		})
		link := widget.NewHyperlink("Queue test link", nil)
		link.OnTapped = func() { events <- "link" }
		w.window.SetContent(container.NewAccessibilityGroup("Queue test actions",
			newNativeSemanticComposite(container.NewVBox(button, link))))
		w.window.Show()
		w.window.updateAccessibility()
		hwnd = uintptr(unsafe.Pointer(w.view().GetWin32Window()))
	})
	for _, step := range []struct {
		name string
		want []string
	}{
		{"Open queue test dialog", []string{"open", "queued"}},
		{"Close queue test dialog", []string{"close"}},
		{"Queue test link", []string{"link"}},
	} {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-File",
			"testdata/accessibility/invoke_windows.ps1", "-WindowHandle", strconv.FormatUint(uint64(hwnd), 10), "-Name", step.name)
		output, err := cmd.CombinedOutput()
		cancel()
		require.NoError(t, err, "%s", output)
		for _, want := range step.want {
			select {
			case got := <-events:
				require.Equal(t, want, got)
			case <-time.After(5 * time.Second):
				t.Fatalf("UIA Invoke did not complete %q", want)
			}
		}
	}
}

type nativeDriverApp struct{ fyne.App }

func (*nativeDriverApp) Driver() fyne.Driver { return d }

func TestMainLoopNativeTree(t *testing.T) {
	// Importing fyne/test installs a dummy app. Logical focus handlers must find
	// the real canvas through the production driver, just as an application does.
	previousApp := fyne.CurrentApp()
	runOnMain(func() { fyne.SetCurrentApp(&nativeDriverApp{App: previousApp}) })
	defer runOnMain(func() { fyne.SetCurrentApp(previousApp) })
	w := createWindow("UIA keyed Tree regression")
	defer w.Close()
	var hwnd uintptr
	var tree *widget.Tree
	selected := make(chan string, 8)
	runOnMain(func() {
		children := make([]string, 120)
		for i := range children {
			children[i] = fmt.Sprintf("item-%03d", i)
		}
		data := map[string][]string{"": {"folder"}, "folder": children}
		tree = widget.NewTreeWithStrings(data)
		tree.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Native tree"})
		tree.OnSelected = func(id string) { selected <- id }
		reorder := widget.NewButton("Reverse tree", func() {
			for i, j := 0, len(children)-1; i < j; i, j = i+1, j-1 {
				children[i], children[j] = children[j], children[i]
			}
			tree.Refresh()
		})
		remove := widget.NewButton("Remove target", func() { data["folder"] = children[1:]; tree.Refresh() })
		restore := widget.NewButton("Restore target", func() { data["folder"] = children; tree.Refresh() })
		w.window.SetContent(container.NewBorder(nil, container.NewVBox(reorder, remove, restore), nil, nil, tree))
		w.window.Resize(fyne.NewSize(350, 280))
		w.window.Show()
		w.window.RequestFocus()
		w.window.updateAccessibility()
		hwnd = uintptr(unsafe.Pointer(w.view().GetWin32Window()))
	})
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-File",
		"testdata/accessibility/tree_windows.ps1", "-WindowHandle", strconv.FormatUint(uint64(hwnd), 10))
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
	select {
	case id := <-selected:
		require.Equal(t, "item-119", id)
	default:
		t.Fatal("UIA selection did not reach the Tree model")
	}
	var active string
	var focused fyne.Focusable
	runOnMain(func() { active, focused = tree.AccessibilityActiveElement(), w.canvas.Focused() })
	require.Equal(t, "item-119", active)
	require.Same(t, tree, focused)
}

// TestMain runs the actual GLFW loop on its locked native thread. Exercise work
// arriving while that loop waits for Windows messages, not a mocked Go select.
func TestMainLoopNativeWork(t *testing.T) {
	var workers sync.WaitGroup
	completed := 0
	for i := 0; i < 64; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			runOnMain(func() { completed++ })
		}()
	}
	done := make(chan struct{})
	go func() { workers.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("main loop did not wake for queued work")
	}
	var got int
	runOnMain(func() { got = completed })
	require.Equal(t, 64, got)
}

func TestMainLoopNativeWorkOrder(t *testing.T) {
	var order []int
	for i := 0; i < 100; i++ {
		value := i
		runOnMainWithWait(func() { order = append(order, value) }, false)
	}
	var result []int
	runOnMain(func() { result = append([]int(nil), order...) })
	require.Len(t, result, 100)
	for i, value := range result {
		require.Equal(t, i, value)
	}
}

func TestMainLoopNativeList(t *testing.T) {
	previousApp := fyne.CurrentApp()
	runOnMain(func() { fyne.SetCurrentApp(&nativeDriverApp{App: previousApp}) })
	defer runOnMain(func() { fyne.SetCurrentApp(previousApp) })
	w := createWindow("UIA keyed List regression")
	defer w.Close()
	var hwnd uintptr
	var list *widget.List
	selected := make(chan string, 8)
	runOnMain(func() {
		items := make([]string, 120)
		for i := range items {
			items[i] = fmt.Sprintf("item-%03d", i)
		}
		list = widget.NewList(func() int { return len(items) },
			func() fyne.CanvasObject { return widget.NewLabel("Template") },
			func(id int, obj fyne.CanvasObject) { obj.(*widget.Label).SetText(items[id]) })
		list.ItemKey = func(id int) string { return items[id] }
		list.DescribeItem = func(id int) fyne.AccessibilityInfo { return fyne.AccessibilityInfo{Name: items[id]} }
		list.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Native list"})
		list.OnSelected = func(id int) { selected <- items[id] }
		reorder := widget.NewButton("Reverse list", func() {
			for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
				items[i], items[j] = items[j], items[i]
			}
			list.Refresh()
		})
		replace := widget.NewButton("Replace target", func() {
			items = items[1:]
			list.Refresh()
			items = append(items, "item-119")
			list.Refresh()
		})
		w.window.SetContent(container.NewBorder(nil, container.NewVBox(reorder, replace), nil, nil, list))
		w.window.Resize(fyne.NewSize(350, 280))
		w.window.Show()
		w.window.RequestFocus()
		w.window.updateAccessibility()
		hwnd = uintptr(unsafe.Pointer(w.view().GetWin32Window()))
	})
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-File",
		"testdata/accessibility/list_windows.ps1", "-WindowHandle", strconv.FormatUint(uint64(hwnd), 10))
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
	select {
	case id := <-selected:
		require.Equal(t, "item-119", id)
	default:
		t.Fatal("UIA selection did not reach the List model")
	}
	var active string
	var focused fyne.Focusable
	runOnMain(func() { active, focused = list.AccessibilityActiveElement(), w.canvas.Focused() })
	require.Equal(t, "item-119", active)
	require.Same(t, list, focused)
}
