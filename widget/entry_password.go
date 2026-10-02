package widget

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
)

var (
	_ desktop.Cursorable = (*passwordRevealer)(nil)
	_ fyne.Tappable      = (*passwordRevealer)(nil)
	_ fyne.Widget        = (*passwordRevealer)(nil)
)

type passwordRevealer struct {
	BaseWidget

	icon    *canvas.Image
	entry   *Entry
	focused bool
}

func (r *passwordRevealer) AccessibilityLabel() string {
	if r.entry.Password {
		return lang.L("Show password")
	}
	return lang.L("Hide password")
}

func (*passwordRevealer) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleButton }
func (r *passwordRevealer) AccessibilityActivate()               { r.Tapped(nil) }
func (r *passwordRevealer) Disabled() bool                       { return r.entry.Disabled() }
func (r *passwordRevealer) Enable()                              { r.entry.Enable() }
func (r *passwordRevealer) Disable()                             { r.entry.Disable() }
func (r *passwordRevealer) FocusGained()                         { r.focused = true; r.Refresh() }
func (r *passwordRevealer) FocusLost()                           { r.focused = false; r.Refresh() }
func (*passwordRevealer) TypedRune(rune)                         {}
func (r *passwordRevealer) TypedKey(event *fyne.KeyEvent) {
	if event.Name == fyne.KeySpace || event.Name == fyne.KeyReturn || event.Name == fyne.KeyEnter {
		r.AccessibilityActivate()
	}
}

func newPasswordRevealer(e *Entry) *passwordRevealer {
	th := e.Theme()
	pr := &passwordRevealer{
		icon:  canvas.NewImageFromResource(th.Icon(theme.IconNameVisibilityOff)),
		entry: e,
	}
	pr.ExtendBaseWidget(pr)
	return pr
}

func (r *passwordRevealer) CreateRenderer() fyne.WidgetRenderer {
	focus := canvas.NewRectangle(color.Transparent)
	focus.StrokeWidth = 2
	focus.Hide()
	return &passwordRevealerRenderer{
		WidgetRenderer: NewSimpleRenderer(r.icon),
		icon:           r.icon,
		entry:          r.entry,
		owner:          r,
		focus:          focus,
	}
}

func (*passwordRevealer) Cursor() desktop.Cursor {
	return desktop.DefaultCursor
}

func (r *passwordRevealer) Tapped(*fyne.PointEvent) {
	if r.entry.Disabled() {
		return
	}

	r.entry.setFieldsAndRefresh(func() {
		r.entry.Password = !r.entry.Password
	})
	if c := fyne.CurrentApp().Driver().CanvasForObject(r); c != nil {
		c.Focus(r.entry.super().(fyne.Focusable))
	}
}

var _ fyne.WidgetRenderer = (*passwordRevealerRenderer)(nil)

type passwordRevealerRenderer struct {
	fyne.WidgetRenderer
	entry *Entry
	icon  *canvas.Image
	owner *passwordRevealer
	focus *canvas.Rectangle
}

func (r *passwordRevealerRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.focus, r.icon}
}

func (r *passwordRevealerRenderer) Layout(size fyne.Size) {
	r.focus.Move(fyne.NewPos(1, 1))
	r.focus.Resize(fyne.NewSize(max(0, size.Width-2), max(0, size.Height-2)))
	iconSize := r.entry.Theme().Size(theme.SizeNameInlineIcon)
	r.icon.Resize(fyne.NewSquareSize(iconSize))
	r.icon.Move(fyne.NewPos((size.Width-iconSize)/2, (size.Height-iconSize)/2))
}

func (r *passwordRevealerRenderer) MinSize() fyne.Size {
	iconSize := r.entry.Theme().Size(theme.SizeNameInlineIcon)
	return fyne.NewSquareSize(iconSize + r.entry.Theme().Size(theme.SizeNameInnerPadding)*2)
}

func (r *passwordRevealerRenderer) Refresh() {
	th := r.entry.Theme()
	r.focus.StrokeColor = th.Color(theme.ColorNameFocus, fyne.CurrentApp().Settings().ThemeVariant())
	if r.owner.focused && !r.entry.Disabled() {
		r.focus.Show()
	} else {
		r.focus.Hide()
	}
	r.focus.Refresh()
	if !r.entry.Password {
		r.icon.Resource = th.Icon(theme.IconNameVisibility)
	} else {
		r.icon.Resource = th.Icon(theme.IconNameVisibilityOff)
	}

	if r.entry.Disabled() {
		r.icon.Resource = theme.NewDisabledResource(r.icon.Resource)
	}
	r.icon.Refresh()
}
