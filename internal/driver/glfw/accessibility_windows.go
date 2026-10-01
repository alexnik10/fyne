//go:build accessibility && windows

package glfw

/*
#cgo LDFLAGS: -lole32 -loleaut32 -luuid
#include <stdlib.h>
#include "accessibility_windows.h"
*/
import "C"

import (
	"sync"
	"sync/atomic"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/internal/accessibility"
	"fyne.io/fyne/v2/internal/scale"
)

type accessibilityBridge struct {
	window *window
	tree   accessibility.Tree
	native *C.WinAccessibility
	handle uintptr
}

// The window map is event-thread-only. COM callbacks use only the handle registry
// and enqueue work; no Go pointer is retained by a C provider.
var accessibilityWindows = make(map[*window]*accessibilityBridge)
var accessibilityHandles sync.Map
var nextAccessibilityHandle atomic.Uint64

func (w *window) accessibilityRoots() []accessibility.Root {
	overlays := w.canvas.Overlays().List()
	roots := []accessibility.Root{
		{Object: w.canvas.Content(), Suppressed: len(overlays) != 0},
		{Object: w.canvas.menu, Suppressed: len(overlays) != 0},
	}
	for i, overlay := range overlays {
		roots = append(roots, accessibility.Root{Object: overlay, Suppressed: i != len(overlays)-1})
	}
	return roots
}

func (w *window) updateAccessibility() {
	if w.view() == nil || w.closing {
		return
	}
	b := accessibilityWindows[w]
	if b == nil {
		b = &accessibilityBridge{window: w, handle: uintptr(nextAccessibilityHandle.Add(1))}
		b.native = C.WinAccessibilityCreate(unsafe.Pointer(w.view().GetWin32Window()), C.uintptr_t(b.handle))
		if b.native == nil {
			return
		}
		accessibilityWindows[w] = b
		accessibilityHandles.Store(b.handle, b)
	}
	nodes := b.tree.Build(w.accessibilityRoots(), w.canvas.Focused())
	// Allocate the array and every string in C memory: cgo never receives an
	// array containing pointers into the Go heap.
	var data *C.WinAccessibilityNode
	if len(nodes) != 0 {
		data = (*C.WinAccessibilityNode)(C.calloc(C.size_t(len(nodes)), C.size_t(C.sizeof_WinAccessibilityNode)))
		if data == nil {
			return
		}
		defer C.free(unsafe.Pointer(data))
	}
	native := unsafe.Slice(data, len(nodes))
	for i, n := range nodes {
		x := &native[i]
		x.id, x.parent = C.uint32_t(n.ID), C.uint32_t(n.Parent)
		x.role = roleToCWin(n.Role)
		x.name, x.description, x.value = C.CString(n.Name), C.CString(n.Description), C.CString(n.Text)
		defer C.free(unsafe.Pointer(x.name))
		defer C.free(unsafe.Pointer(x.description))
		defer C.free(unsafe.Pointer(x.value))
		x.x, x.y = C.double(scale.ToScreenCoordinate(w.canvas, n.Position.X)), C.double(scale.ToScreenCoordinate(w.canvas, n.Position.Y))
		x.width, x.height = C.double(scale.ToScreenCoordinate(w.canvas, n.Size.Width)), C.double(scale.ToScreenCoordinate(w.canvas, n.Size.Height))
		x.number, x.minimum, x.maximum, x.step = C.double(n.Number), C.double(n.Min), C.double(n.Max), C.double(n.Step)
		flags := []bool{n.Disabled, n.Focusable, n.Focused, n.Required, n.Invalid, n.Invoke, n.Toggle, n.Value, n.Range, n.Checked, n.ReadOnly, n.Protected}
		for bit, set := range flags {
			if set {
				x.flags |= 1 << bit
			}
		}
	}
	C.WinAccessibilityUpdate(b.native, data, C.int(len(nodes)))
}

func roleToCWin(role fyne.AccessibleRole) C.int {
	switch role {
	case fyne.AccessibleRoleButton:
		return 1
	case fyne.AccessibleRoleText:
		return 2
	case fyne.AccessibleRoleLink:
		return 3
	case fyne.AccessibleRoleCheck:
		return 4
	case fyne.AccessibleRoleEntry:
		return 5
	case fyne.AccessibleRoleSlider:
		return 6
	case fyne.AccessibleRoleDialog:
		return 7
	default:
		return 0
	}
}

//export goFyneAccessibilityAction
func goFyneAccessibilityAction(handle C.uintptr_t, id C.uint32_t, action C.int, value *C.char, number C.double) {
	text := ""
	if value != nil {
		text = C.GoString(value)
	}
	// Invoke is asynchronous, so application callbacks may open dialogs safely.
	runOnMainWithWait(func() {
		performAccessibilityAction(uintptr(handle), uint32(id), accessibility.Action(action), text, float64(number))
	}, false)
}

//export goFyneAccessibilityPerform
func goFyneAccessibilityPerform(handle C.uintptr_t, id C.uint32_t, action C.int, value *C.char, number C.double) C.int {
	text := ""
	if value != nil {
		text = C.GoString(value)
	}
	// Called only by the window procedure on Fyne's locked native event thread.
	// No provider lock is held. Setters and focus complete before UIA returns.
	if performAccessibilityAction(uintptr(handle), uint32(id), accessibility.Action(action), text, float64(number)) {
		return 1
	}
	return 0
}

func performAccessibilityAction(handle uintptr, id uint32, action accessibility.Action, text string, number float64) bool {
	stored, ok := accessibilityHandles.Load(handle)
	if !ok {
		return false
	}
	b := stored.(*accessibilityBridge)
	if b.window.closing || b.window.view() == nil {
		return false
	}
	b.tree.Build(b.window.accessibilityRoots(), b.window.canvas.Focused())
	accepted := id == 0 && action == accessibility.Focus
	if !accepted {
		accepted = b.tree.Perform(id, action, text, number, b.window.canvas)
	}
	if accepted && action == accessibility.Focus {
		b.window.view().Focus()
	}
	b.window.updateAccessibility()
	return accepted
}

func (w *window) pollAccessibility() {
	if b := accessibilityWindows[w]; b != nil {
		C.WinAccessibilityFocus(b.native, C.uint32_t(b.tree.FocusedID(w.canvas.Focused())))
	}
}

func (w *window) initAccessibilityForWindow() {}

func (w *window) cleanupAccessibilityForWindow() {
	if b := accessibilityWindows[w]; b != nil {
		accessibilityHandles.Delete(b.handle)
		delete(accessibilityWindows, w)
		C.WinAccessibilityCleanup(b.native)
	}
}
