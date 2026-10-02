//go:build accessibility && windows && !no_glfw && !mobile

package glfw

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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
