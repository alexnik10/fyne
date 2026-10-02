//go:build accessibility && windows && !no_glfw && !mobile

package glfw

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"
	"unsafe"

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
		popup = widget.NewModalPopUp(closeButton, w.canvas)
		button := widget.NewButton("Open queue test dialog", func() {
			popup.Show()
			events <- "open"
			// Exercise another asynchronous enqueue from an Invoke callback.
			runOnMainWithWait(func() { events <- "queued" }, false)
		})
		link := widget.NewHyperlink("Queue test link", nil)
		link.OnTapped = func() { events <- "link" }
		w.window.SetContent(container.NewVBox(button, link))
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
