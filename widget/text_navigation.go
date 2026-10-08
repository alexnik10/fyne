package widget

import (
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

// Read-only documents share caret movement with their accessibility selection.
// Tab remains owned by the canvas; editing shortcuts are deliberately absent.
type textNavigation struct {
	shift bool
}

type readOnlyDocument interface {
	fyne.AccessibleText
	fyne.AccessibleTextScroller
}

func (n *textNavigation) keyDown(key *fyne.KeyEvent) {
	if key.Name == desktop.KeyShiftLeft || key.Name == desktop.KeyShiftRight {
		n.shift = true
	}
}

func (n *textNavigation) keyUp(key *fyne.KeyEvent) {
	if key.Name == desktop.KeyShiftLeft || key.Name == desktop.KeyShiftRight {
		n.shift = false
	}
}

func (n *textNavigation) move(doc readOnlyDocument, key fyne.KeyName, modifiers fyne.KeyModifier) {
	info := doc.AccessibilityText()
	if info.SelectionDisabled || len(info.Positions) == 0 {
		return
	}
	last := len(info.Positions) - 1
	caret := min(max(info.Caret, 0), last)
	next := caret
	control := modifiers&fyne.KeyModifierControl != 0
	shift := n.shift || modifiers&fyne.KeyModifierShift != 0
	line := info.Positions[caret].Line
	switch key {
	case fyne.KeyLeft, fyne.KeyRight:
		direction := 1
		if key == fyne.KeyLeft {
			direction = -1
		}
		next += direction
		if !shift && info.SelectionStart != info.SelectionEnd {
			next = info.SelectionEnd
			if direction < 0 {
				next = info.SelectionStart
			}
		} else if control {
			next = readOnlyWordBoundary(info, caret, direction)
		}
	case fyne.KeyHome, fyne.KeyEnd:
		if control {
			next = 0
			if key == fyne.KeyEnd {
				next = last
			}
		} else {
			for i, p := range info.Positions {
				if p.Line == line {
					next = i
					if key == fyne.KeyHome {
						break
					}
				}
			}
		}
	case fyne.KeyUp, fyne.KeyDown, fyne.KeyPageUp, fyne.KeyPageDown:
		step := 1
		if key == fyne.KeyPageUp || key == fyne.KeyPageDown {
			step = max(1, int(info.ViewportSize.Height/max(1, info.Positions[caret].Height)))
		}
		if key == fyne.KeyUp || key == fyne.KeyPageUp {
			step = -step
		}
		target := min(max(0, line+step), info.Positions[last].Line)
		distance := float32(math.MaxFloat32)
		for i, p := range info.Positions {
			if p.Line != target {
				continue
			}
			delta := float32(math.Abs(float64(p.Position.X - info.Positions[caret].Position.X)))
			if delta < distance {
				next, distance = i, delta
			}
		}
	default:
		return
	}
	next = min(max(0, next), last)
	anchor := next
	if shift {
		anchor = info.SelectionStart
		if caret == info.SelectionStart {
			anchor = info.SelectionEnd
		}
	}
	doc.AccessibilitySelectText(anchor, next)
	revealTextCaret(doc)
}

func readOnlyWordBoundary(info fyne.AccessibilityTextInfo, caret, direction int) int {
	next := 0
	if direction > 0 {
		next = len(info.Positions) - 1
	}
	for _, boundary := range info.WordBoundaries {
		if direction < 0 && boundary < caret {
			next = boundary
		} else if direction > 0 && boundary > caret {
			return boundary
		}
	}
	return next
}

func (n *textNavigation) shortcut(doc readOnlyDocument, shortcut fyne.Shortcut) {
	info := doc.AccessibilityText()
	if info.SelectionDisabled {
		return
	}
	switch s := shortcut.(type) {
	case *fyne.ShortcutCopy:
		clipboard := s.Clipboard
		if clipboard == nil {
			clipboard = fyne.CurrentApp().Clipboard()
		}
		if info.SelectionStart != info.SelectionEnd {
			clipboard.SetContent(string([]rune(info.Text)[info.SelectionStart:info.SelectionEnd]))
		}
	case *fyne.ShortcutSelectAll:
		doc.AccessibilitySelectText(0, len([]rune(info.Text)))
	case fyne.KeyboardShortcut:
		if s.Mod()&^(fyne.KeyModifierControl|fyne.KeyModifierShift) == 0 {
			n.move(doc, s.Key(), s.Mod())
		}
	}
}

func revealTextCaret(doc readOnlyDocument) {
	info := doc.AccessibilityText()
	if len(info.Positions) == 0 {
		return
	}
	p := info.Positions[min(max(info.Caret, 0), len(info.Positions)-1)]
	v, size := info.ViewportPosition, info.ViewportSize
	if p.Position.X < v.X || p.Position.X >= v.X+size.Width || p.Position.Y < v.Y || p.Position.Y+p.Height > v.Y+size.Height {
		doc.AccessibilityScrollText(info.Caret, info.Caret, p.Position.Y < v.Y)
	}
}
