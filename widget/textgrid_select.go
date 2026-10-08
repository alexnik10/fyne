package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
)

// TabStop includes selectable documents in sequential keyboard navigation.
// Since: 2.9
func (t *TextGrid) TabStop() bool { return t.Selectable }

// FocusGained displays the read-only caret.
// Since: 2.9
func (t *TextGrid) FocusGained() { t.focused = true; t.Refresh() }

// FocusLost preserves the reading position and clears held modifiers.
// Since: 2.9
func (t *TextGrid) FocusLost() { t.focused = false; t.navigation.shift = false; t.Refresh() }

// TypedRune does not edit the read-only document.
// Since: 2.9
func (*TextGrid) TypedRune(rune) {}

// TypedKey moves the reading caret or extends a selection.
// Since: 2.9
func (t *TextGrid) TypedKey(key *fyne.KeyEvent) { t.navigation.move(t, key.Name, 0) }

// KeyDown tracks selection modifiers.
// Since: 2.9
func (t *TextGrid) KeyDown(key *fyne.KeyEvent) { t.navigation.keyDown(key) }

// KeyUp releases selection modifiers.
// Since: 2.9
func (t *TextGrid) KeyUp(key *fyne.KeyEvent) { t.navigation.keyUp(key) }

// TypedShortcut supports Copy, Select All and modified caret movement.
// Since: 2.9
func (t *TextGrid) TypedShortcut(shortcut fyne.Shortcut) { t.navigation.shortcut(t, shortcut) }

// SelectedText returns the selected model text, retaining tabs and Unicode.
// Since: 2.9
func (t *TextGrid) SelectedText() string {
	if !t.Selectable {
		return ""
	}
	info := t.AccessibilityText()
	return string([]rune(info.Text)[info.SelectionStart:info.SelectionEnd])
}

// Tapped places the reading caret without changing the document.
// Since: 2.9
func (t *TextGrid) Tapped(event *fyne.PointEvent) {
	if !t.AccessibilityFocus() {
		return
	}
	info := t.AccessibilityText()
	best, distance := 0, float32(1e30)
	for i, p := range info.Positions {
		x, y := p.Position.X-event.Position.X, p.Position.Y+p.Height/2-event.Position.Y
		d := x*x + y*y*100
		if d < distance {
			best, distance = i, d
		}
	}
	t.AccessibilitySelectText(best, best)
}

func (t *TextGrid) updateSelectionGeometry() {
	if !t.Selectable {
		return
	}
	info := t.AccessibilityText()
	t.caret, t.anchor = info.Caret, min(max(t.anchor, 0), len(info.Positions)-1)
	t.selectionStart = info.Positions[info.SelectionStart]
	t.selectionEnd = info.Positions[info.SelectionEnd]
	if t.Scroll != fyne.ScrollNone && t.scroller != nil {
		origin := t.scroller.Offset.Subtract(t.scroller.Position())
		t.selectionStart.Position = t.selectionStart.Position.Add(origin)
		t.selectionEnd.Position = t.selectionEnd.Position.Add(origin)
	}
}

func (t *TextGrid) cellSelected(row, column int) bool {
	if !t.Selectable || !t.focused || t.anchor == t.caret || t.content == nil {
		return false
	}
	start, end := t.selectionStart, t.selectionEnd
	x := float32(column) * t.content.cellSize.Width
	return row >= start.Line && row <= end.Line && (row != start.Line || x >= start.Position.X) && (row != end.Line || x < end.Position.X)
}

func (r *textGridRenderer) refreshCaret() {
	t := r.text.text
	if !t.Selectable {
		return
	}
	if r.caret == nil {
		r.caret = canvas.NewRectangle(t.Theme().Color(theme.ColorNameForeground, fyne.CurrentApp().Settings().ThemeVariant()))
	}
	info := t.AccessibilityText()
	p := info.Positions[info.Caret]
	v, size := info.ViewportPosition, info.ViewportSize
	if !t.focused || t.anchor != t.caret || p.Position.X < v.X || p.Position.X > v.X+size.Width || p.Position.Y < v.Y || p.Position.Y >= v.Y+size.Height {
		r.caret.Hide()
		return
	}
	r.caret.FillColor = t.Theme().Color(theme.ColorNameForeground, fyne.CurrentApp().Settings().ThemeVariant())
	r.caret.Move(fyne.NewPos(min(p.Position.X, v.X+size.Width-1), p.Position.Y))
	r.caret.Resize(fyne.NewSize(1, min(p.Height, v.Y+size.Height-p.Position.Y)))
	r.caret.Show()
	r.caret.Refresh()
}
