package widget

import (
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// AccessibilityLabel returns the fallback name; use SetAccessibilityInfo to name a viewport.
//
// Since: 2.9
func (*Scroll) AccessibilityLabel() string { return "" }

// AccessibilityRole identifies a scroll viewport as a container.
//
// Since: 2.9
func (*Scroll) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleContainer }

// AccessibilityInfo returns explicit metadata for the scroll viewport.
//
// Since: 2.9
func (s *Scroll) AccessibilityInfo() fyne.AccessibilityInfo { return s.accessibilityInfo }

// SetAccessibilityInfo names and describes the viewport without changing its content.
//
// Since: 2.9
func (s *Scroll) SetAccessibilityInfo(info fyne.AccessibilityInfo) {
	s.accessibilityInfo = info
	Repaint(s)
}

// AccessibilityChildren exposes the content without scrollbar or shadow objects.
//
// Since: 2.9
func (s *Scroll) AccessibilityChildren() []fyne.CanvasObject {
	if s.Content == nil {
		return nil
	}
	return []fyne.CanvasObject{s.Content}
}

// AccessibilityScroll describes the current viewport without changing its offset.
//
// Since: 2.9
func (s *Scroll) AccessibilityScroll() fyne.AccessibilityScrollInfo {
	if s == nil || s.Content == nil {
		return fyne.AccessibilityScrollInfo{Direction: fyne.ScrollNone}
	}
	th := theme.CurrentForWidget(s)
	step := th.Size(theme.SizeNameInlineIcon) + th.Size(theme.SizeNamePadding)
	return fyne.AccessibilityScrollInfo{
		Offset: s.Offset, ContentSize: s.Content.Size(), ViewportSize: s.Size(),
		Direction: s.Direction, SmallStep: fyne.NewSize(step, step),
	}
}

// AccessibilityScrollTo changes the viewport without changing selection or focus.
//
// Since: 2.9
func (s *Scroll) AccessibilityScrollTo(offset fyne.Position) bool {
	if s == nil || s.Content == nil || !s.Visible() ||
		math.IsNaN(float64(offset.X)) || math.IsInf(float64(offset.X), 0) ||
		math.IsNaN(float64(offset.Y)) || math.IsInf(float64(offset.Y), 0) {
		return false
	}
	info := s.AccessibilityScroll()
	if info.Direction != fyne.ScrollBoth && info.Direction != fyne.ScrollHorizontalOnly {
		offset.X = s.Offset.X
	}
	if info.Direction != fyne.ScrollBoth && info.Direction != fyne.ScrollVerticalOnly {
		offset.Y = s.Offset.Y
	}
	offset.X = min(max(offset.X, 0), max(0, info.ContentSize.Width-info.ViewportSize.Width))
	offset.Y = min(max(offset.Y, 0), max(0, info.ContentSize.Height-info.ViewportSize.Height))
	if offset == s.Offset {
		return true
	}
	s.Offset = offset
	if s.OnScrolled != nil {
		s.OnScrolled(offset)
	}
	s.refreshWithoutOffsetUpdate()
	return true
}
