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
	return accessibility.Roots(w.canvas, w.canvas.menu)
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
		x.level = C.int(n.Level)
		x.grid_owner = C.uint32_t(n.GridOwner)
		x.rows, x.columns = C.int(n.Rows), C.int(n.Columns)
		x.row, x.column = C.int(n.Row), C.int(n.Column)
		x.row_span, x.column_span = C.int(n.RowSpan), C.int(n.ColumnSpan)
		x.shortcut = C.CString(n.Shortcut)
		defer C.free(unsafe.Pointer(x.shortcut))
		x.name, x.description, x.value = C.CString(n.Name), C.CString(n.Description), C.CString(n.Text)
		defer C.free(unsafe.Pointer(x.name))
		defer C.free(unsafe.Pointer(x.description))
		defer C.free(unsafe.Pointer(x.value))
		x.x, x.y = C.double(scale.ToScreenCoordinate(w.canvas, n.BoundsPosition.X)), C.double(scale.ToScreenCoordinate(w.canvas, n.BoundsPosition.Y))
		x.width, x.height = C.double(scale.ToScreenCoordinate(w.canvas, n.BoundsSize.Width)), C.double(scale.ToScreenCoordinate(w.canvas, n.BoundsSize.Height))
		x.horizontal_percent, x.vertical_percent = C.double(n.HorizontalScrollPercent), C.double(n.VerticalScrollPercent)
		x.horizontal_view, x.vertical_view = C.double(n.HorizontalViewSize), C.double(n.VerticalViewSize)
		x.number, x.minimum, x.maximum, x.step = C.double(n.Number), C.double(n.Min), C.double(n.Max), C.double(n.Step)
		if doc := n.Document; doc != nil {
			originX := C.double(scale.ToScreenCoordinate(w.canvas, n.Position.X))
			originY := C.double(scale.ToScreenCoordinate(w.canvas, n.Position.Y))
			x.text = C.CString(doc.Text)
			defer C.free(unsafe.Pointer(x.text))
			x.caret, x.selection_start, x.selection_end = C.int(doc.Caret), C.int(doc.SelectionStart), C.int(doc.SelectionEnd)
			x.text_revision = C.uint64_t(doc.Revision)
			if doc.SelectionDisabled {
				x.selection_disabled = 1
			}
			if len(doc.Runs) != 0 {
				runs := (*C.WinAccessibilityTextRun)(C.calloc(C.size_t(len(doc.Runs)), C.size_t(C.sizeof_WinAccessibilityTextRun)))
				if runs == nil {
					return
				}
				defer C.free(unsafe.Pointer(runs))
				for j, run := range doc.Runs {
					marshalTextRun(&unsafe.Slice(runs, len(doc.Runs))[j], run)
				}
				x.runs, x.run_count = runs, C.int(len(doc.Runs))
			}
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
			x.viewport_x = originX + C.double(scale.ToScreenCoordinate(w.canvas, doc.ViewportPosition.X))
			x.viewport_y = originY + C.double(scale.ToScreenCoordinate(w.canvas, doc.ViewportPosition.Y))
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
					point.x = originX + C.double(scale.ToScreenCoordinate(w.canvas, p.Position.X))
					point.y = originY + C.double(scale.ToScreenCoordinate(w.canvas, p.Position.Y))
					point.height = C.double(scale.ToScreenCoordinate(w.canvas, p.Height))
					point.line = C.int(p.Line)
				}
				x.positions, x.position_count = positions, C.int(len(doc.Positions))
			}
		}
		flags := []bool{n.Disabled, n.Focusable, n.Focused, n.Required, n.Invalid, n.Invoke, n.Toggle, n.Value, n.Range, n.Checked, n.ReadOnly, n.Protected, n.Document != nil, n.Selection, n.Multiple, n.SelectionRequired, n.Selectable, n.Selected, n.Expandable, n.Expanded, n.Role == fyne.AccessibleRoleTreeItem && !n.Expandable, n.ScrollItem, n.ItemContainer, n.VirtualizedItem, n.Grid, n.GridItem, n.Table, n.RowHeaders, n.ColumnHeaders, n.Scroll}
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
			diagnostic.Add(diagnostic.Sample{
				Kind: "uia_slowest_event", Window: windowID, DurationUS: int64(stats.slowest_ms * microsecondsPerMillisecond),
				Node: uint32(stats.slowest_node), EventID: int(stats.slowest_id), EventKind: int(stats.slowest_kind),
			})
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
	case fyne.AccessibleRoleTree:
		return 11
	case fyne.AccessibleRoleTreeItem:
		return 12
	case fyne.AccessibleRoleList:
		return 13
	case fyne.AccessibleRoleTab:
		return 14
	case fyne.AccessibleRoleTabItem:
		return 15
	case fyne.AccessibleRoleMenu:
		return 16
	case fyne.AccessibleRoleMenuBar:
		return 17
	case fyne.AccessibleRoleMenuItem:
		return 18
	case fyne.AccessibleRoleTable:
		return 19
	case fyne.AccessibleRoleCell:
		return 20
	case fyne.AccessibleRoleHeader:
		return 21
	case fyne.AccessibleRoleDocument:
		return 22
	case fyne.AccessibleRoleProgressBar:
		return 23
	case fyne.AccessibleRoleToolBar:
		return 24
	case fyne.AccessibleRoleImage:
		return 25
	case fyne.AccessibleRoleCalendar:
		return 26
	case fyne.AccessibleRoleSeparator:
		return 27
	default:
		return 0
	}
}

func marshalTextRun(out *C.WinAccessibilityTextRun, run fyne.AccessibilityTextRun) {
	out.start, out.end = C.int(run.Start), C.int(run.End)
	// Fyne text sizes use logical 96-DPI units. UIA requires typographic points.
	const pointsPerLogicalUnit = 72.0 / 96.0
	out.size = C.double(float64(run.Size) * pointsPerLogicalUnit)
	out.weight = 400
	if run.Style.Bold {
		out.weight = 700
	}
	if run.Style.Italic {
		out.italic = 1
	}
	if run.Style.Underline {
		out.underline = 1
	}
	if run.Style.Strikethrough {
		out.strike = 1
	}
	if run.Style.Monospace {
		out.monospace = 1
	}
	out.alignment, out.heading = C.int(run.Alignment), C.int(run.HeadingLevel)
	out.foreground = C.uint32_t(uint32(run.Foreground.R) | uint32(run.Foreground.G)<<8 | uint32(run.Foreground.B)<<16)
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
	if action == accessibility.Realize {
		accepted = b.tree.Realize(id)
	} else if !accepted {
		accepted = b.tree.Perform(id, action, text, number, b.window.canvas)
	}
	if accepted && action == accessibility.Focus {
		b.window.view().Focus()
	}
	b.window.updateAccessibility()
	return accepted
}

//export goFyneAccessibilityScroll
func goFyneAccessibilityScroll(handle C.uintptr_t, id C.uint32_t, relative C.int, horizontal, vertical C.double) C.int {
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
	if relative != 0 {
		accepted = b.tree.Scroll(uint32(id), fyne.AccessibilityScrollAmount(horizontal), fyne.AccessibilityScrollAmount(vertical))
	} else {
		accepted = b.tree.SetScrollPercent(uint32(id), float64(horizontal), float64(vertical))
	}
	b.window.updateAccessibility()
	if accepted {
		return 1
	}
	return 0
}

//export goFyneAccessibilityFindItem
func goFyneAccessibilityFindItem(handle C.uintptr_t, container, start C.uint32_t, property C.int, value *C.char, result *C.uint32_t) C.int {
	stored, ok := accessibilityHandles.Load(uintptr(handle))
	if !ok {
		return 0
	}
	b := stored.(*accessibilityBridge)
	if b.window.closing || b.window.view() == nil {
		return 0
	}
	// Called by the HWND procedure, with no native provider lock held.
	b.tree.Build(b.window.accessibilityRoots(), b.window.canvas.Focused())
	owner, key, valid := b.tree.FindItem(uint32(container), uint32(start), accessibility.FindProperty(property), C.GoString(value))
	if !valid {
		return 0
	}
	if key == "" {
		return 1
	}
	if !b.tree.RequestElement(owner, key) {
		return 0
	}
	b.window.updateAccessibility()
	if node, exists := b.tree.NodeForElement(owner, key); exists {
		*result = C.uint32_t(node.ID)
		return 1
	}
	return 0
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
