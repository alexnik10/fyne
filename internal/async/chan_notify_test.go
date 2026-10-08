package async_test

import (
	"testing"
	"time"

	"fyne.io/fyne/v2/internal/async"
)

func TestUnboundedChanNotifyReady(t *testing.T) {
	const count = 128
	observed := make(chan int, count)
	var queue *async.UnboundedChan[int]
	queue = async.NewUnboundedChanWithNotify[int](func() {
		// No other receiver: every notification must follow publication, not
		// merely acceptance at In. Never wait here for a future publication.
		select {
		case value := <-queue.Out():
			observed <- value
		default:
			observed <- -1
		}
	})
	defer queue.Close()
	for i := 0; i < count; i++ {
		queue.In() <- i
	}
	for i := 0; i < count; i++ {
		select {
		case value := <-observed:
			if value != i {
				t.Fatalf("notification %d: got %d; value must already be readable in FIFO order", i, value)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("publication did not notify the receiver")
		}
	}
}

func TestUnboundedChanNotifyBacklog(t *testing.T) {
	for _, closing := range []bool{false, true} {
		name := "running"
		if closing {
			name = "closing"
		}
		t.Run(name, func(t *testing.T) {
			const count = 128
			wake := make(chan struct{}, 1) // coalesces like an auto-reset event
			published := make(chan struct{}, count)
			queue := async.NewUnboundedChanWithNotify[int](func() {
				select {
				case wake <- struct{}{}:
				default:
				}
				published <- struct{}{}
			})
			for i := 0; i < count; i++ {
				queue.In() <- i
			}
			// Fill Out before consuming so further publications must come from
			// the relay's backlog, including its separate Close drain path.
			for i := 0; i < cap(queue.Out()); i++ {
				select {
				case <-published:
				case <-time.After(5 * time.Second):
					t.Fatal("output buffer did not fill")
				}
			}
			if closing {
				queue.Close()
			} else {
				defer queue.Close()
			}
			deadline := time.After(5 * time.Second)
			for want := 0; want < count; {
				select {
				case got := <-queue.Out():
					if got != want {
						t.Fatalf("got %d, want %d", got, want)
					}
					want++
				default:
					// No timer/ticker retries and no producer-side wake: a missed
					// notification strands the backlog until the test fails.
					select {
					case <-wake:
					case <-deadline:
						t.Fatalf("stalled after %d values", want)
					}
				}
			}
			if closing {
				select {
				case _, ok := <-queue.Out():
					if ok {
						t.Fatal("unexpected value after draining the queue")
					}
				case <-deadline:
					t.Fatal("output did not close")
				}
			}
		})
	}
}
