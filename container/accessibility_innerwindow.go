package container

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	innerWindowKeyboardStep = 10
	innerWindowPairFormat   = "%g,%g"
)

// AccessibilityLabel names an inner window.
// Since: 2.9
func (w *InnerWindow) AccessibilityLabel() string { return w.Title }

// AccessibilityRole identifies a named content group inside the platform window.
// Since: 2.9
func (*InnerWindow) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleContainer }

// AccessibilityMode retains the title bar, window commands and content.
// Since: 2.9
func (w *InnerWindow) AccessibilityMode() fyne.AccessibilityMode {
	if mode := w.BaseWidget.AccessibilityMode(); mode != fyne.AccessibilityAuto {
		return mode
	}
	return fyne.AccessibilityGroup
}

func (d *draggableLabel) AccessibilityLabel() string {
	return lang.X("accessibility.window.move", "Move {{.Title}}", map[string]any{"Title": d.win.Title})
}
func (*draggableLabel) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleButton }
func (d *draggableLabel) AccessibilityActivate()               { d.Tapped(nil) }
func (d *draggableLabel) AccessibilityValue() (value string, readOnly, protected bool) {
	return fmt.Sprintf(innerWindowPairFormat, d.win.Position().X, d.win.Position().Y), d.win.OnDragged == nil, false
}
func (d *draggableLabel) AccessibilitySetValue(value string) { d.AccessibilitySetValueChecked(value) }
func (d *draggableLabel) AccessibilitySetValueChecked(value string) bool {
	x, y, ok := windowPair(value)
	if !ok || d.win.OnDragged == nil {
		return false
	}
	target := fyne.NewPos(x, y)
	delta := target.Subtract(d.win.Position())
	absolute := fyne.CurrentApp().Driver().AbsolutePositionForObject(d.win).Add(delta)
	d.win.OnDragged(&fyne.DragEvent{PointEvent: fyne.PointEvent{AbsolutePosition: absolute}, Dragged: fyne.Delta{DX: delta.X, DY: delta.Y}})
	return d.win.Position() == target
}
func (d *draggableLabel) FocusGained() { d.focused = true; d.Tapped(nil); d.Refresh() }
func (d *draggableLabel) FocusLost()   { d.focused = false; d.Refresh() }
func (*draggableLabel) TypedRune(rune) {}
func (d *draggableLabel) TypedKey(event *fyne.KeyEvent) {
	if event.Name == fyne.KeyEnter || event.Name == fyne.KeyReturn || event.Name == fyne.KeySpace {
		d.Tapped(nil)
		return
	}
	x, y := windowKeyDelta(event.Name)
	p := d.win.Position()
	d.AccessibilitySetValue(fmt.Sprintf(innerWindowPairFormat, p.X+x, p.Y+y))
}

func (*draggableCorner) AccessibilityLabel() string {
	return lang.X("accessibility.window.resize", "Resize window")
}
func (*draggableCorner) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleSeparator }
func (d *draggableCorner) AccessibilityValue() (value string, readOnly, protected bool) {
	return fmt.Sprintf(innerWindowPairFormat, d.win.Size().Width, d.win.Size().Height), d.win.OnResized == nil, false
}
func (d *draggableCorner) AccessibilitySetValue(value string) { d.AccessibilitySetValueChecked(value) }
func (d *draggableCorner) AccessibilitySetValueChecked(value string) bool {
	x, y, ok := windowPair(value)
	minimum := d.win.MinSize()
	if !ok || d.win.OnResized == nil || x < minimum.Width || y < minimum.Height {
		return false
	}
	d.win.OnResized(&fyne.DragEvent{Dragged: fyne.Delta{DX: x - d.win.Size().Width, DY: y - d.win.Size().Height}})
	return d.win.Size() == fyne.NewSize(x, y)
}
func (d *draggableCorner) FocusGained() { d.focused = true; d.Refresh() }
func (d *draggableCorner) FocusLost()   { d.focused = false; d.Refresh() }
func (*draggableCorner) TypedRune(rune) {}
func (d *draggableCorner) TypedKey(event *fyne.KeyEvent) {
	x, y := windowKeyDelta(event.Name)
	s := d.win.Size()
	d.AccessibilitySetValue(fmt.Sprintf(innerWindowPairFormat, s.Width+x, s.Height+y))
}

func windowKeyDelta(key fyne.KeyName) (x, y float32) {
	switch key {
	case fyne.KeyLeft:
		return -innerWindowKeyboardStep, 0
	case fyne.KeyRight:
		return innerWindowKeyboardStep, 0
	case fyne.KeyUp:
		return 0, -innerWindowKeyboardStep
	case fyne.KeyDown:
		return 0, innerWindowKeyboardStep
	}
	return 0, 0
}

func windowPair(value string) (x, y float32, valid bool) {
	parts := strings.Split(value, ",")
	if len(parts) != 2 {
		return 0, 0, false
	}
	px, e1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 32)
	py, e2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 32)
	return float32(px), float32(py), e1 == nil && e2 == nil && !math.IsNaN(px) && !math.IsNaN(py) && !math.IsInf(px, 0) && !math.IsInf(py, 0)
}

func (d *draggableLabel) CreateRenderer() fyne.WidgetRenderer {
	return newWindowFocusRenderer(d.Label.CreateRenderer(), &d.BaseWidget, &d.focused)
}

type windowFocusRenderer struct {
	fyne.WidgetRenderer
	widget  *widget.BaseWidget
	focused *bool
	outline *canvas.Rectangle
}

func newWindowFocusRenderer(base fyne.WidgetRenderer, owner *widget.BaseWidget, focused *bool) fyne.WidgetRenderer {
	return &windowFocusRenderer{WidgetRenderer: base, widget: owner, focused: focused, outline: canvas.NewRectangle(color.Transparent)}
}

func (r *windowFocusRenderer) Objects() []fyne.CanvasObject {
	return append(append([]fyne.CanvasObject(nil), r.WidgetRenderer.Objects()...), r.outline)
}

func (r *windowFocusRenderer) Layout(size fyne.Size) {
	r.WidgetRenderer.Layout(size)
	r.outline.Resize(size)
}

func (r *windowFocusRenderer) Refresh() {
	r.WidgetRenderer.Refresh()
	r.outline.StrokeWidth = 0
	if *r.focused {
		r.outline.StrokeWidth = r.widget.Theme().Size(theme.SizeNameInputBorder)
	}
	r.outline.StrokeColor = r.widget.Theme().Color(theme.ColorNameFocus, fyne.CurrentApp().Settings().ThemeVariant())
	r.outline.Refresh()
}
