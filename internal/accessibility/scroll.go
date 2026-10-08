package accessibility

import (
	"math"

	"fyne.io/fyne/v2"
)

const (
	noScroll           = -1
	scrollPageFraction = 0.95
	scrollSmallStep    = 32
)

func populateScroll(n *Node, obj fyne.CanvasObject) {
	_, n.ScrollItem = obj.(fyne.AccessibleScrollItem)
	if s, ok := obj.(fyne.AccessibleScroll); ok {
		n.Scroll = true
		info := s.AccessibilityScroll()
		x, y := scrollLimits(info)
		n.HorizontalScrollPercent, n.HorizontalViewSize = scrollPercent(info.Offset.X, info.ViewportSize.Width, x)
		n.VerticalScrollPercent, n.VerticalViewSize = scrollPercent(info.Offset.Y, info.ViewportSize.Height, y)
	}
}

func scrollLimits(info fyne.AccessibilityScrollInfo) (x, y float32) {
	if info.Direction == fyne.ScrollBoth || info.Direction == fyne.ScrollHorizontalOnly {
		x = scrollLimit(info.Offset.X, info.ViewportSize.Width, info.ContentSize.Width)
	}
	if info.Direction == fyne.ScrollBoth || info.Direction == fyne.ScrollVerticalOnly {
		y = scrollLimit(info.Offset.Y, info.ViewportSize.Height, info.ContentSize.Height)
	}
	return x, y
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func scrollLimit(offset, view, content float32) float32 {
	if !finite(float64(offset)) || !finite(float64(view)) || !finite(float64(content)) || view <= 0 || content <= view {
		return 0
	}
	return content - view
}

func scrollPercent(offset, view, limit float32) (percent, viewSize float64) {
	if limit <= 0 {
		return noScroll, 100
	}
	return float64(min(max(offset, 0), limit)) / float64(limit) * 100, float64(view) / (float64(view) + float64(limit)) * 100
}

func (t *Tree) scrollTarget(id uint32) fyne.AccessibleScroll {
	obj := t.objects[id]
	n := t.nodes[id]
	if obj == nil || !obj.Visible() || n.Disabled || !n.Scroll {
		return nil
	}
	if d, ok := obj.(fyne.Disableable); ok && d.Disabled() {
		return nil
	}
	s, _ := obj.(fyne.AccessibleScroll)
	return s
}

// SetScrollPercent revalidates both axes before changing either. -1 leaves an
// axis unchanged. Rebuild immediately before calling, as with Perform.
func (t *Tree) SetScrollPercent(id uint32, horizontal, vertical float64) bool {
	if !validScrollPercent(horizontal) || !validScrollPercent(vertical) {
		return false
	}
	s := t.scrollTarget(id)
	if s == nil {
		return false
	}
	info := s.AccessibilityScroll()
	x, y := scrollLimits(info)
	if (horizontal != noScroll && x <= 0) || (vertical != noScroll && y <= 0) {
		return false
	}
	offset := info.Offset
	if horizontal != noScroll {
		offset.X = float32(horizontal / 100 * float64(x))
	}
	if vertical != noScroll {
		offset.Y = float32(vertical / 100 * float64(y))
	}
	if horizontal == noScroll && vertical == noScroll {
		return true
	}
	return s.AccessibilityScrollTo(offset)
}

func validScrollPercent(value float64) bool {
	return finite(value) && (value == noScroll || (value >= 0 && value <= 100))
}

// Scroll makes small or page-sized changes without changing focus or selection.
// Unsupported requested axes reject the entire command; edges are successful.
func (t *Tree) Scroll(id uint32, horizontal, vertical fyne.AccessibilityScrollAmount) bool {
	if horizontal > fyne.AccessibilityScrollLargeIncrement || vertical > fyne.AccessibilityScrollLargeIncrement {
		return false
	}
	s := t.scrollTarget(id)
	if s == nil {
		return false
	}
	info := s.AccessibilityScroll()
	x, y := scrollLimits(info)
	if (horizontal != fyne.AccessibilityScrollNone && x <= 0) || (vertical != fyne.AccessibilityScrollNone && y <= 0) {
		return false
	}
	if horizontal == fyne.AccessibilityScrollNone && vertical == fyne.AccessibilityScrollNone {
		return true
	}
	offset := info.Offset
	if horizontal != fyne.AccessibilityScrollNone {
		offset.X = min(max(offset.X+scrollDelta(horizontal, info.ViewportSize.Width, info.SmallStep.Width), 0), x)
	}
	if vertical != fyne.AccessibilityScrollNone {
		offset.Y = min(max(offset.Y+scrollDelta(vertical, info.ViewportSize.Height, info.SmallStep.Height), 0), y)
	}
	return s.AccessibilityScrollTo(offset)
}

func scrollDelta(amount fyne.AccessibilityScrollAmount, view, small float32) float32 {
	if !finite(float64(small)) || small <= 0 {
		small = scrollSmallStep
	}
	switch amount {
	case fyne.AccessibilityScrollSmallDecrement:
		return -min(small, view)
	case fyne.AccessibilityScrollSmallIncrement:
		return min(small, view)
	case fyne.AccessibilityScrollLargeDecrement:
		return -view * scrollPageFraction
	case fyne.AccessibilityScrollLargeIncrement:
		return view * scrollPageFraction
	default:
		return 0
	}
}

func (t *Tree) scrollAncestor(parent uint32) uint32 {
	for parent != 0 {
		n := t.nodes[parent]
		if n.Scroll {
			return parent
		}
		if isSemanticPopup(t.objects[parent]) {
			break
		}
		parent = n.Parent
	}
	return 0
}

func (t *Tree) scrollIntoView(n Node, obj fyne.CanvasObject) bool {
	if s, ok := obj.(fyne.AccessibleScrollItem); ok {
		return s.AccessibilityScrollIntoView()
	}
	parent := t.scrollAncestor(n.Parent)
	if parent == 0 {
		return false
	}
	for ancestor := parent; ancestor != 0; ancestor = t.scrollAncestor(t.nodes[ancestor].Parent) {
		if t.scrollTarget(ancestor) == nil {
			return false
		}
	}
	position := n.Position
	for parent != 0 {
		s := t.scrollTarget(parent)
		if s == nil {
			return false
		}
		info := s.AccessibilityScroll()
		owner := t.nodes[parent]
		x, y := scrollLimits(info)
		left := position.X - owner.Position.X - info.ViewportPosition.X
		top := position.Y - owner.Position.Y - info.ViewportPosition.Y
		offset := fyne.NewPos(
			min(max(info.Offset.X+revealDelta(left, n.Size.Width, info.ViewportSize.Width), 0), x),
			min(max(info.Offset.Y+revealDelta(top, n.Size.Height, info.ViewportSize.Height), 0), y))
		if !s.AccessibilityScrollTo(offset) {
			return false
		}
		actual := s.AccessibilityScroll().Offset
		position.X -= actual.X - info.Offset.X
		position.Y -= actual.Y - info.Offset.Y
		parent = t.scrollAncestor(owner.Parent)
	}
	return true
}

func revealDelta(start, size, view float32) float32 {
	if start < 0 {
		return start
	}
	if start+size > view {
		return min(start, start+size-view)
	}
	return 0
}
