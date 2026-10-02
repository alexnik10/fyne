//go:build accessibility && windows

package glfw

/*
#include "accessibility_windows.h"
*/
import "C"

import "time"

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
			// posted input or advancing animations/rendering. Tick wakes also bound
			// delivery if funcQueue's internal relay has not forwarded a wake's work.
			if C.WinAccessibilityWaitForMessage(^C.uint32_t(0)) == 0 {
				return waitMainLoopChannels(done, work, ticks)
			}
		}
	}
}

func wakeMainLoop() {
	C.WinAccessibilityWake()
}
