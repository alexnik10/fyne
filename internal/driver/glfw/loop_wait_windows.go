//go:build accessibility && windows

package glfw

/*
#include "accessibility_windows.h"
*/
import "C"

import (
	"time"

	"fyne.io/fyne/v2/internal/async"
)

func newMainLoopQueue() *async.UnboundedChan[funcData] {
	// Wake only after Out can be read. Signalling after a send to In can race
	// the queue's relay and leave ready work asleep until the next frame.
	return async.NewUnboundedChanWithNotify[funcData](wakeMainLoop)
}

// Keep frame pacing on the existing Go ticker, but wake the native wait when a
// tick or shutdown becomes ready. The relay does not consume application work.
func mainLoopTickEvents(ticks <-chan time.Time, done <-chan struct{}) <-chan time.Time {
	forwarded := make(chan time.Time, 1)
	go func() {
		for {
			select {
			case <-done:
				wakeMainLoop()
				return
			case tick := <-ticks:
				select {
				case forwarded <- tick:
				default:
				}
				wakeMainLoop()
			}
		}
	}()
	return forwarded
}

func nextMainLoopEvent(done <-chan struct{}, work <-chan funcData, ticks <-chan time.Time) (mainLoopEvent, funcData) {
	for {
		select {
		case <-done:
			return mainLoopStop, funcData{}
		case f := <-work:
			return mainLoopWork, f
		case <-ticks:
			return mainLoopFrame, funcData{}
		default:
			// Process synchronous window queries between frames, without consuming
			// posted input or advancing animations/rendering. The queue publishes
			// work before signalling, so delivery does not depend on frame ticks.
			if C.WinAccessibilityWaitForMessage(^C.uint32_t(0)) == 0 {
				return waitMainLoopChannels(done, work, ticks)
			}
		}
	}
}

func wakeMainLoop() {
	C.WinAccessibilityWake()
}
