//go:build !accessibility || !windows

package glfw

import "time"

func mainLoopTickEvents(ticks <-chan time.Time, _ <-chan struct{}) <-chan time.Time {
	return ticks
}

func nextMainLoopEvent(done <-chan struct{}, work <-chan funcData, ticks <-chan time.Time) (mainLoopEvent, funcData) {
	return waitMainLoopChannels(done, work, ticks)
}

func wakeMainLoop() {}
