// Package accessibilitydemo provides shared scenes for the accessibility demo
// and native integration tests.
package accessibilitydemo

import (
	"fmt"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

const gridRecords = 500

// NewGrid demonstrates keyboard-accessible model children in a virtualized grid.
// The caller displays status beside the returned content.
func NewGrid(status *widget.Label) fyne.CanvasObject {
	status.SetAccessibilityLiveSetting(fyne.AccessibilityLivePolite)
	g := &checkGrid{keys: make([]string, gridRecords), enabled: make(map[string]bool), status: status}
	g.ExtendBaseWidget(g)
	for i := range g.keys {
		g.keys[i] = fmt.Sprintf("Record %03d", i)
	}
	g.Length = func() int { return len(g.keys) }
	g.CreateItem = func() fyne.CanvasObject {
		check := &gridCheck{owner: g}
		check.ExtendBaseWidget(check)
		check.Text = "Enable Record 000"
		return check
	}
	g.UpdateItem = func(i int, object fyne.CanvasObject) {
		check, ok := object.(*gridCheck)
		if !ok {
			return
		}
		check.key = g.keys[i]
		check.Text, check.Checked = "Enable "+check.key, g.enabled[check.key]
		check.Refresh()
	}
	g.ItemKey = func(i int) string { return g.keys[i] }
	g.DescribeItem = func(i int) fyne.AccessibilityInfo { return fyne.AccessibilityInfo{Name: g.keys[i]} }
	g.ItemElements = func(i int) []fyne.AccessibilityElement {
		key := g.keys[i]
		check := &modelGridCheck{owner: g, key: key}
		check.ExtendBaseWidget(check)
		check.Text, check.Checked = "Enable "+key, g.enabled[key]
		check.OnChanged = func(value bool) { g.change(key, value) }
		check.Resize(check.MinSize())
		return []fyne.AccessibilityElement{{Key: "enabled", Object: check}}
	}
	const help = "Arrow keys move between checkboxes. Space changes the active checkbox. Home and End go to the first and last records. Tab leaves the grid."
	g.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Keyed wrapping grid", Description: help})
	instructions := widget.NewLabel(help)
	instructions.Wrapping = fyne.TextWrapWord
	controls := container.NewHBox(widget.NewButton("Swap first and last", g.swap))
	return container.New(layout.NewBorderLayout(instructions, controls, nil, nil), instructions, g, controls)
}

// Canvas focus stays on the grid; its model key identifies the focused checkbox.
// Renderer cells are recycled and must never become independent Tab stops.
type checkGrid struct {
	widget.GridWrap
	keys    []string
	enabled map[string]bool
	status  *widget.Label
}

func (g *checkGrid) AccessibilityActiveElement() string {
	key := g.GridWrap.AccessibilityActiveElement()
	if key == "" {
		return ""
	}
	return widget.CollectionControlKey(key, "enabled")
}

func (g *checkGrid) TypedKey(event *fyne.KeyEvent) {
	switch event.Name {
	case fyne.KeySpace, fyne.KeyReturn, fyne.KeyEnter:
		g.toggle(g.GridWrap.AccessibilityActiveElement())
	case fyne.KeyHome:
		g.Highlight(0)
	case fyne.KeyEnd:
		g.Highlight(len(g.keys) - 1)
	default:
		g.GridWrap.TypedKey(event)
	}
}

func (g *checkGrid) change(key string, value bool) {
	if !slices.Contains(g.keys, key) {
		return
	}
	g.enabled[key] = value
	g.status.SetText(fmt.Sprintf("%s enabled: %t", key, value))
	g.Refresh()
}

func (g *checkGrid) toggle(key string) { g.change(key, !g.enabled[key]) }

func (g *checkGrid) focus(key string) bool {
	index := slices.Index(g.keys, key)
	canvas := fyne.CurrentApp().Driver().CanvasForObject(g)
	if index < 0 || canvas == nil {
		return false
	}
	g.Highlight(index)
	canvas.Focus(g)
	return canvas.Focused() == g
}

func (g *checkGrid) swap() {
	key := g.GridWrap.AccessibilityActiveElement()
	g.keys[0], g.keys[len(g.keys)-1] = g.keys[len(g.keys)-1], g.keys[0]
	g.Refresh()
	if index := slices.Index(g.keys, key); index >= 0 {
		g.ScrollTo(index)
	}
}

type gridCheck struct {
	widget.Check
	owner *checkGrid
	key   string
}

func (*gridCheck) TabStop() bool { return false }

func (c *gridCheck) Tapped(_ *fyne.PointEvent) {
	key := c.key
	if c.Disabled() || !c.owner.focus(key) {
		return
	}
	c.owner.toggle(key)
}

type modelGridCheck struct {
	widget.Check
	owner *checkGrid
	key   string
}

func (c *modelGridCheck) AccessibilityFocusable() bool {
	return !c.Disabled() && slices.Contains(c.owner.keys, c.key)
}

func (c *modelGridCheck) AccessibilityFocus() bool { return c.owner.focus(c.key) }

func (c *modelGridCheck) AccessibilityScrollIntoView() bool {
	index := slices.Index(c.owner.keys, c.key)
	if index < 0 {
		return false
	}
	c.owner.ScrollTo(index)
	return true
}
