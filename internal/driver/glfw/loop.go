package glfw

import (
	"runtime"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/internal/accessibility/diagnostic"
	"fyne.io/fyne/v2/internal/app"
	"fyne.io/fyne/v2/internal/async"
	"fyne.io/fyne/v2/internal/cache"
	"fyne.io/fyne/v2/internal/driver/common"
	"fyne.io/fyne/v2/internal/goos"
	"fyne.io/fyne/v2/internal/painter"
	"fyne.io/fyne/v2/internal/scale"
)

type funcData struct {
	f    func()
	done chan struct{} // Zero allocation signalling channel
}

type mainLoopEvent uint8

const (
	mainLoopStop mainLoopEvent = iota
	mainLoopWork
	mainLoopFrame
)

func waitMainLoopChannels(done <-chan struct{}, work <-chan funcData, ticks <-chan time.Time) (mainLoopEvent, funcData) {
	select {
	case <-done:
		return mainLoopStop, funcData{}
	case f := <-work:
		return mainLoopWork, f
	case <-ticks:
		return mainLoopFrame, funcData{}
	}
}

// channel for queuing functions on the main thread
var (
	funcQueue        = newMainLoopQueue()
	running, drained atomic.Bool
)

// Arrange that main.main runs on main thread.
func init() {
	runtime.LockOSThread()
	async.SetMainGoroutine()
}

// force a function f to run on the main thread
func runOnMain(f func()) {
	runOnMainWithWait(f, true)
}

// force a function f to run on the main thread and specify if we should wait for it to return
func runOnMainWithWait(f func(), wait bool) {
	// If we are on main before app run just execute - otherwise add it to the main queue and wait.
	// We also need to run it as-is if the app is in the process of shutting down as the queue will be stopped.
	if (!running.Load() && async.IsMainGoroutine()) || drained.Load() {
		f()
		return
	}

	if wait {
		done := common.DonePool.Get()
		defer common.DonePool.Put(done)

		funcQueue.In() <- funcData{f: f, done: done}
		<-done
	} else {
		funcQueue.In() <- funcData{f: f}
	}
}

func decideRepaint(visible, ready bool, checkDirtyAndClear func() bool) bool {
	return visible && ready && checkDirtyAndClear()
}

func (d *gLDriver) drawSingleFrame() {
	refreshed := false
	for _, win := range d.AllWindows() {
		w, _ := win.(*window)
		if w.closing {
			continue
		}

		if decideRepaint(w.visible, w.frame.ready(), w.canvas.CheckDirtyAndClear) {
			paintStart := diagnostic.Start()
			w.RunWithContext(func() {
				if w.driver.repaintWindow(w) {
					refreshed = true
				}
			})
			diagnostic.Duration("render", diagnostic.WindowID(w), "", paintStart)
			w.updateAccessibility()
		} else {
			w.markCacheAlive()
		}
		w.pollAccessibility()
	}
	cache.Clean(refreshed)
}

func (w *window) markCacheAlive() {
	threshold := time.Now().Add(10*time.Second - cache.ValidDuration)
	if w.lastWalkedTime.Before(threshold) {
		w.canvas.WalkTrees(nil, func(node *common.RenderCacheNode, _ fyne.Position) {
			_ = cache.GetCanvasForObject(node.Obj())
			if wid, ok := node.Obj().(fyne.Widget); ok {
				_, _ = cache.CachedRenderer(wid)
			}
		})
		w.lastWalkedTime = time.Now()
	}
}

func (*gLDriver) applyThemeToWindow(w fyne.Window) {
	if win, ok := w.(*window); ok {
		win.setDarkMode()
	}
}

func (d *gLDriver) runGL() {
	if !running.CompareAndSwap(false, true) {
		return // Run was called twice.
	}

	d.init()
	if d.trayStart != nil {
		d.trayStart()
	}

	fyne.CurrentApp().Settings().AddListener(func(set fyne.Settings) {
		painter.ClearFontCache()
		cache.ResetThemeCaches()
		app.ApplySettingsWithCallback(set, fyne.CurrentApp(), func(w fyne.Window) {
			d.applyThemeToWindow(w)
			c, ok := w.Canvas().(*glCanvas)
			if !ok {
				return
			}
			c.applyThemeOutOfTreeObjects()
			c.reloadScale()
		})
	})

	if f := fyne.CurrentApp().Lifecycle().(*app.Lifecycle).OnStarted(); f != nil {
		f()
	}

	eventTick := time.NewTicker(time.Second / 60)
	ticks := mainLoopTickEvents(eventTick.C, d.done)
	for {
		event, f := nextMainLoopEvent(d.done, funcQueue.Out(), ticks)
		switch event {
		case mainLoopStop:
			eventTick.Stop()
			d.Terminate()
			l, _ := fyne.CurrentApp().Lifecycle().(*app.Lifecycle)
			if f := l.OnStopped(); f != nil {
				l.QueueEvent(f)
			}

			// as we are shutting down make sure we drain the pending funcQueue and close it out.
			for len(funcQueue.Out()) > 0 {
				f := <-funcQueue.Out()
				if f.done != nil {
					f.done <- struct{}{}
				}
			}
			drained.Store(true)
			funcQueue.Close()
			return
		case mainLoopWork:
			f.f()
			if f.done != nil {
				f.done <- struct{}{}
			}
		case mainLoopFrame:
			d.pollEvents()
			for i := 0; i < len(d.windows); i++ {
				w, _ := d.windows[i].(*window)
				w.ensurePositionProcessed()

				if w.viewport == nil {
					continue
				}

				if w.viewport.ShouldClose() {
					d.destroyWindow(w, i)
					i-- // Trailing windows are moved forward one step.
					continue
				}

				expand := w.shouldExpand
				fullScreen := w.fullScreen

				if expand && !fullScreen {
					w.fitContent()
					shouldExpand := w.shouldExpand
					w.shouldExpand = false
					view := w.viewport

					if shouldExpand && runtime.GOOS != goos.JavaScript {
						view.SetSize(w.shouldWidth, w.shouldHeight)
					}
				}
			}

			d.animation.TickAnimations()
			d.drawSingleFrame()
		}
	}
}

func (d *gLDriver) destroyWindow(w *window, index int) {
	w.visible = false
	w.viewport.Destroy()
	w.destroy(d)

	if index < len(d.windows)-1 {
		copy(d.windows[index:], d.windows[index+1:])
	}
	d.windows[len(d.windows)-1] = nil
	d.windows = d.windows[:len(d.windows)-1]

	if len(d.windows) == 0 {
		d.Quit()
	}
}

func (*gLDriver) repaintWindow(w *window) bool {
	canvas := w.canvas
	freed := false
	if canvas.EnsureMinSize() {
		w.shouldExpand = true
	}
	freed = canvas.FreeDirtyTextures() > 0

	updateGLContext(w)
	canvas.paint(canvas.Size())

	view := w.viewport
	visible := w.visible

	if view != nil && visible {
		w.frame.requestFrame()
		view.SwapBuffers()
	}

	// mark that we have walked the window and don't
	// need to walk it again to mark caches alive
	w.lastWalkedTime = time.Now()
	return freed
}

// refreshWindow requests that the specified window be redrawn
func refreshWindow(w *window) {
	w.canvas.SetDirty()
}

func updateGLContext(w *window) {
	canvas := w.canvas
	size := canvas.Size()

	// w.width and w.height are not correct if we are maximised, so figure from canvas
	winWidth := float32(scale.ToScreenCoordinate(canvas, size.Width)) * canvas.texScale
	winHeight := float32(scale.ToScreenCoordinate(canvas, size.Height)) * canvas.texScale

	canvas.Painter().SetFrameBufferScale(canvas.texScale)
	canvas.Painter().SetOutputSize(int(winWidth), int(winHeight))
}
