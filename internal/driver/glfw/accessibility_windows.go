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
	"time"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/internal/accessibility"
	"fyne.io/fyne/v2/internal/accessibility/diagnostic"
	"fyne.io/fyne/v2/internal/scale"
)

type accessibilityBridge struct {
	window *window
	tree   accessibility.Tree
	scope  accessibility.FocusScope
	native *C.WinAccessibility
	handle uintptr
}

// The window map is event-thread-only. COM callbacks use the handle registry
// and marshal work to the window thread; C providers never retain Go pointers.
var (
	accessibilityWindows    = make(map[*window]*accessibilityBridge)
	accessibilityHandles    sync.Map
	nextAccessibilityHandle atomic.Uint64
)

func (w *window) accessibilityRoots() []accessibility.Root {
	overlays := w.canvas.Overlays().List()
	roots := []accessibility.Root{
		{Object: w.canvas.Content(), Suppressed: len(overlays) != 0},
		{Object: w.canvas.menu, Suppressed: len(overlays) != 0},
	}
	for i, overlay := range overlays {
		roots = append(roots, accessibility.Root{Object: overlay, Suppressed: i != len(overlays)-1})
	}
	if owner, ok := w.canvas.Overlays().Top().(fyne.AccessibleOverlayOwner); ok {
		if scope := owner.AccessibilityOverlayOwner(); scope != nil {
			for i := range roots {
				roots[i].Suppressed, roots[i].Scope = false, scope
			}
		}
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
	buildStart := diagnostic.Start()
	b.scope.Update(w.canvas)
	nodes := b.tree.Build(w.accessibilityRoots(), w.canvas.Focused())
	diagnostic.Duration("semantic_tree", diagnostic.WindowID(w), "", buildStart)
	marshalStart := diagnostic.Start()
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
		x.selection_owner = C.uint32_t(n.SelectionOwner)
		x.set_position, x.set_size = C.int(n.SetPosition), C.int(n.SetSize)
		x.name, x.description, x.value = C.CString(n.Name), C.CString(n.Description), C.CString(n.Text)
		defer C.free(unsafe.Pointer(x.name))
		defer C.free(unsafe.Pointer(x.description))
		defer C.free(unsafe.Pointer(x.value))
		x.x, x.y = C.double(scale.ToScreenCoordinate(w.canvas, n.Position.X)), C.double(scale.ToScreenCoordinate(w.canvas, n.Position.Y))
		x.width, x.height = C.double(scale.ToScreenCoordinate(w.canvas, n.Size.Width)), C.double(scale.ToScreenCoordinate(w.canvas, n.Size.Height))
		x.number, x.minimum, x.maximum, x.step = C.double(n.Number), C.double(n.Min), C.double(n.Max), C.double(n.Step)
		if doc := n.Document; doc != nil {
			x.text = C.CString(doc.Text)
			defer C.free(unsafe.Pointer(x.text))
			x.caret, x.selection_start, x.selection_end = C.int(doc.Caret), C.int(doc.SelectionStart), C.int(doc.SelectionEnd)
			x.text_revision = C.uint64_t(doc.Revision)
			if len(doc.WordBoundaries) != 0 {
				words := (*C.int)(C.calloc(C.size_t(len(doc.WordBoundaries)), C.size_t(C.sizeof_int)))
				if words == nil {
					return
				}
				defer C.free(unsafe.Pointer(words))
				for j, boundary := range doc.WordBoundaries {
					unsafe.Slice(words, len(doc.WordBoundaries))[j] = C.int(boundary)
				}
				x.word_boundaries, x.word_boundary_count = words, C.int(len(doc.WordBoundaries))
			}
			x.viewport_x = x.x + C.double(scale.ToScreenCoordinate(w.canvas, doc.ViewportPosition.X))
			x.viewport_y = x.y + C.double(scale.ToScreenCoordinate(w.canvas, doc.ViewportPosition.Y))
			x.viewport_width = C.double(scale.ToScreenCoordinate(w.canvas, doc.ViewportSize.Width))
			x.viewport_height = C.double(scale.ToScreenCoordinate(w.canvas, doc.ViewportSize.Height))
			if len(doc.Positions) != 0 {
				positions := (*C.WinAccessibilityTextPosition)(C.calloc(C.size_t(len(doc.Positions)), C.size_t(C.sizeof_WinAccessibilityTextPosition)))
				if positions == nil {
					return
				}
				defer C.free(unsafe.Pointer(positions))
				for j, p := range doc.Positions {
					point := &unsafe.Slice(positions, len(doc.Positions))[j]
					point.x = x.x + C.double(scale.ToScreenCoordinate(w.canvas, p.Position.X))
					point.y = x.y + C.double(scale.ToScreenCoordinate(w.canvas, p.Position.Y))
					point.height = C.double(scale.ToScreenCoordinate(w.canvas, p.Height))
					point.line = C.int(p.Line)
				}
				x.positions, x.position_count = positions, C.int(len(doc.Positions))
			}
		}
		flags := []bool{n.Disabled, n.Focusable, n.Focused, n.Required, n.Invalid, n.Invoke, n.Toggle, n.Value, n.Range, n.Checked, n.ReadOnly, n.Protected, n.Document != nil, n.Selection, n.Multiple, n.SelectionRequired, n.Selectable, n.Selected, n.Expandable, n.Expanded}
		for bit, set := range flags {
			if set {
				x.flags |= 1 << bit
			}
		}
	}
	diagnostic.Duration("marshal_snapshot", diagnostic.WindowID(w), "", marshalStart)
	if diagnostic.Enabled {
		const microsecondsPerMillisecond = 1000
		var stats C.WinAccessibilityStats
		start := diagnostic.Start()
		C.WinAccessibilityUpdateWithStats(b.native, data, C.int(len(nodes)), &stats)
		windowID := diagnostic.WindowID(w)
		diagnostic.Duration("native_update_total", windowID, "", start)
		diagnostic.Add(diagnostic.Sample{Kind: "native_snapshot", Window: windowID, DurationUS: int64(stats.snapshot_ms * microsecondsPerMillisecond), Count: len(nodes)})
		diagnostic.Add(diagnostic.Sample{Kind: "uia_events", Window: windowID, DurationUS: int64(stats.events_ms * microsecondsPerMillisecond), Count: int(stats.event_count)})
		if stats.event_count > 0 {
			diagnostic.Add(diagnostic.Sample{Kind: "uia_slowest_event", Window: windowID, DurationUS: int64(stats.slowest_ms * microsecondsPerMillisecond),
				Node: uint32(stats.slowest_node), EventID: int(stats.slowest_id), EventKind: int(stats.slowest_kind)})
		}
	} else {
		C.WinAccessibilityUpdate(b.native, data, C.int(len(nodes)))
	}
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
	case fyne.AccessibleRoleComboBox:
		return 8
	case fyne.AccessibleRoleRadio:
		return 9
	case fyne.AccessibleRoleListItem:
		return 10
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
	if diagnostic.Enabled && action == accessibility.Toggle {
		diagnostic.Add(diagnostic.Sample{Kind: "uia_toggle_request", Window: diagnostic.WindowID(b.window), Node: id})
	}
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

//export goFyneAccessibilityTextAction
func goFyneAccessibilityTextAction(handle C.uintptr_t, id C.uint32_t, start, end C.int, scroll, alignTop C.int) C.int {
	stored, ok := accessibilityHandles.Load(uintptr(handle))
	if !ok {
		return 0
	}
	b := stored.(*accessibilityBridge)
	if b.window.closing || b.window.view() == nil {
		return 0
	}
	b.tree.Build(b.window.accessibilityRoots(), b.window.canvas.Focused())
	var accepted bool
	if scroll != 0 {
		accepted = b.tree.ScrollText(uint32(id), int(start), int(end), alignTop != 0)
	} else {
		accepted = b.tree.SelectText(uint32(id), int(start), int(end))
	}
	b.window.updateAccessibility()
	if accepted {
		return 1
	}
	return 0
}

func (w *window) pollAccessibility() {
	if b := accessibilityWindows[w]; b != nil {
		start := diagnostic.Start()
		C.WinAccessibilityFocus(b.native, C.uint32_t(b.tree.FocusedID(w.canvas.Focused())))
		if diagnostic.Enabled && time.Since(start) >= time.Millisecond {
			diagnostic.Duration("slow_focus_poll", diagnostic.WindowID(w), "", start)
		}
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
