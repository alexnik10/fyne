// Package components demonstrates custom widgets using only Fyne's public API.
package components

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// FilePicker composes existing controls without duplicating their semantics.
type FilePicker struct {
	widget.BaseWidget
	Path   *widget.Entry
	Browse *widget.Button
}

func NewFilePicker(browse func()) *FilePicker {
	p := &FilePicker{Path: widget.NewEntry(), Browse: widget.NewButton("Browse", browse)}
	p.Path.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "File path"})
	p.ExtendBaseWidget(p)
	return p
}

func (p *FilePicker) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewBorder(nil, nil, nil, p.Browse, p.Path))
}

// Toggle owns its behavior and semantic state; its label is renderer detail.
type Toggle struct {
	widget.BaseWidget
	Name    string
	Checked bool
	focused bool
}

func NewToggle(name string) *Toggle {
	t := &Toggle{Name: name}
	t.ExtendBaseWidget(t)
	return t
}

func (t *Toggle) AccessibilityLabel() string           { return t.Name }
func (*Toggle) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleCheck }
func (t *Toggle) AccessibilityChecked() bool           { return t.Checked }
func (t *Toggle) AccessibilityToggle()                 { t.Checked = !t.Checked; t.Refresh() }
func (t *Toggle) Tapped(*fyne.PointEvent)              { t.AccessibilityToggle() }
func (t *Toggle) FocusGained()                         { t.focused = true; t.Refresh() }
func (t *Toggle) FocusLost()                           { t.focused = false; t.Refresh() }
func (*Toggle) TypedRune(rune)                         {}
func (t *Toggle) TypedKey(key *fyne.KeyEvent) {
	if key.Name == fyne.KeySpace {
		t.AccessibilityToggle()
	}
}

func (t *Toggle) CreateRenderer() fyne.WidgetRenderer {
	r := &toggleRenderer{owner: t, label: widget.NewLabel("")}
	r.WidgetRenderer = widget.NewSimpleRenderer(r.label)
	r.Refresh()
	return r
}

type toggleRenderer struct {
	fyne.WidgetRenderer
	owner *Toggle
	label *widget.Label
}

func (r *toggleRenderer) Refresh() {
	state := "off"
	if r.owner.Checked {
		state = "on"
	}
	if r.owner.focused {
		state = "> " + state
	}
	r.label.SetText(r.owner.Name + ": " + state)
}
