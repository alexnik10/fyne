package widget

import (
	"slices"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/internal/cache"
)

const calendarDateKeyFormat = "2006-01-02"

// AccessibilityLabel identifies the displayed month.
// Since: 2.9
func (c *Calendar) AccessibilityLabel() string { return c.monthYear() }

// AccessibilityRole identifies a calendar.
// Since: 2.9
func (*Calendar) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleCalendar }

// AccessibilitySelection describes single-date selection.
// Since: 2.9
func (*Calendar) AccessibilitySelection() (multiple, required bool) { return false, true }

// AccessibilityGrid describes the displayed weeks and days.
// Since: 2.9
func (c *Calendar) AccessibilityGrid() (rows, columns int) {
	if c.dates == nil {
		return 0, daysPerWeek
	}
	return (len(c.dates.Objects) - 1) / daysPerWeek, daysPerWeek
}

// AccessibilityTable provides weekday headers.
// Since: 2.9
func (*Calendar) AccessibilityTable() (rowHeaders, columnHeaders bool) { return false, true }

// AccessibilityCellKey locates a day or weekday header in the visible month.
// Since: 2.9
func (c *Calendar) AccessibilityCellKey(row, column int) string {
	if c.dates == nil || row < -1 || column < 0 || column >= daysPerWeek {
		return ""
	}
	index := (row+1)*daysPerWeek + column
	rows, _ := c.AccessibilityGrid()
	if index >= (rows+1)*daysPerWeek {
		return ""
	}
	return c.calendarKey(c.dates.Objects, index)
}

// AccessibilityCollection exposes persistent controls with logical month/date keys.
// Since: 2.9
func (c *Calendar) AccessibilityCollection() fyne.AccessibilityCollection {
	cache.Renderer(c)
	s := &calendarAccessibilitySource{owner: c}
	if c.dates == nil {
		return s
	}
	s.objects = append([]fyne.CanvasObject{c.monthPrevious, c.monthNext, c.monthLabel}, c.dates.Objects...)
	s.keys = []string{"previous", "next", "title"}
	rows, _ := c.AccessibilityGrid()
	for i := 0; i < (rows+1)*daysPerWeek; i++ {
		s.keys = append(s.keys, c.calendarKey(c.dates.Objects, i))
	}
	selected := c.selectedDate.Format(calendarDateKeyFormat)
	if !slices.Contains(s.keys, selected) {
		s.keys = append(s.keys, selected)
		s.offscreenSelected = selected
	}
	return s
}

func (c *Calendar) calendarKey(objects []fyne.CanvasObject, index int) string {
	if index < len(objects) {
		if day, ok := objects[index].(*calendarDay); ok {
			return day.date.Format(calendarDateKeyFormat)
		}
	}
	return c.currentTime.Format("2006-01/") + strconv.Itoa(index)
}

type calendarAccessibilitySource struct {
	owner             *Calendar
	objects           []fyne.CanvasObject
	keys              []string
	offscreenSelected string
}

func (s *calendarAccessibilitySource) Revision() uint64       { return s.owner.accessibilityRevision }
func (s *calendarAccessibilitySource) ViewportKeys() []string { return s.keys }
func (s *calendarAccessibilitySource) SelectedKeys() []string {
	key := s.owner.selectedDate.Format(calendarDateKeyFormat)
	if _, _, ok := s.Index(key); ok {
		return []string{key}
	}
	return nil
}

func (s *calendarAccessibilitySource) ChildCount(parent string) int {
	if parent != "" {
		return 0
	}
	return len(s.keys)
}

func (s *calendarAccessibilitySource) ChildKey(parent string, index int) string {
	if parent != "" || index < 0 || index >= len(s.keys) {
		return ""
	}
	return s.keys[index]
}

func (s *calendarAccessibilitySource) Index(key string) (string, int, bool) {
	for i, k := range s.keys {
		if k == key {
			return "", i, true
		}
	}
	return "", 0, false
}

func (s *calendarAccessibilitySource) Element(key string) (fyne.AccessibilityElement, bool) {
	_, index, ok := s.Index(key)
	if !ok {
		return fyne.AccessibilityElement{}, false
	}
	if key == s.offscreenSelected {
		day := &calendarDay{owner: s.owner, date: s.owner.selectedDate}
		day.ExtendBaseWidget(day)
		day.OnTapped = day.choose
		return fyne.AccessibilityElement{Key: key, Object: day, Generation: s.owner.accessibilityLifetimes.generations[key]}, true
	}
	driver := fyne.CurrentApp().Driver()
	origin := driver.AbsolutePositionForObject(s.owner)
	var object fyne.CanvasObject
	var position fyne.Position
	if index < len(s.objects) {
		object = s.objects[index]
		position = driver.AbsolutePositionForObject(object).Subtract(origin)
	}
	gridIndex := index - 3
	if gridIndex >= 0 {
		_, day := object.(*calendarDay)
		if !day {
			size := s.owner.dates.Layout.(*calendarLayout).cellSize
			position = driver.AbsolutePositionForObject(s.owner.dates).Subtract(origin).AddXY(float32(gridIndex%daysPerWeek)*size.Width, float32(gridIndex/daysPerWeek)*size.Height)
			cell := &calendarEmpty{owner: s.owner, index: gridIndex, position: position, size: size}
			if label, ok := object.(*Label); ok {
				cell.label = label.Text
			}
			object = cell
		}
	}
	focus, _ := object.(fyne.Focusable)
	return fyne.AccessibilityElement{Key: key, Object: object, Position: &position, FocusTarget: focus, Generation: s.owner.accessibilityLifetimes.generations[key]}, true
}

type calendarEmpty struct {
	owner    *Calendar
	index    int
	label    string
	position fyne.Position
	size     fyne.Size
}

func (e *calendarEmpty) AccessibilityLabel() string { return e.label }
func (e *calendarEmpty) AccessibilityRole() fyne.AccessibleRole {
	if e.index < daysPerWeek {
		return fyne.AccessibleRoleHeader
	}
	return fyne.AccessibleRoleCell
}

func (e *calendarEmpty) AccessibilityGridItem() (owner fyne.CanvasObject, row, column, rowSpan, columnSpan int) {
	return e.owner, e.index/daysPerWeek - 1, e.index % daysPerWeek, 1, 1
}
func (e *calendarEmpty) Position() fyne.Position { return e.position }
func (e *calendarEmpty) Size() fyne.Size         { return e.size }
func (e *calendarEmpty) MinSize() fyne.Size      { return e.size }
func (*calendarEmpty) Move(fyne.Position)        {}
func (*calendarEmpty) Resize(fyne.Size)          {}
func (*calendarEmpty) Show()                     {}
func (*calendarEmpty) Hide()                     {}
func (*calendarEmpty) Refresh()                  {}
func (*calendarEmpty) Visible() bool             { return true }

type calendarNavigation struct {
	Button
	owner *Calendar
}

func (b *calendarNavigation) TypedKey(event *fyne.KeyEvent) {
	if event.Name == fyne.KeyEscape && b.owner.dismiss != nil {
		b.owner.dismiss()
		return
	}
	b.Button.TypedKey(event)
}

type calendarDay struct {
	Button
	owner *Calendar
	date  time.Time
}

func (d *calendarDay) AccessibilityLabel() string           { return d.date.Format("Monday, 2 January 2006") }
func (*calendarDay) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleCell }
func (d *calendarDay) AccessibilityGridItem() (owner fyne.CanvasObject, row, column, rowSpan, columnSpan int) {
	for i, object := range d.owner.dates.Objects {
		if object == d {
			return d.owner, i/daysPerWeek - 1, i % daysPerWeek, 1, 1
		}
	}
	return d.owner, -1, -1, 1, 1
}

func (d *calendarDay) AccessibilitySelectionItem() (owner fyne.CanvasObject, selected bool, position, count int) {
	return d.owner, d.owner.selectedDate.Format(calendarDateKeyFormat) == d.date.Format(calendarDateKeyFormat), d.date.Day(), time.Date(d.date.Year(), d.date.Month()+1, 1, 0, 0, 0, 0, d.date.Location()).AddDate(0, 0, -1).Day()
}

func (d *calendarDay) AccessibilitySelect(mode fyne.AccessibilitySelectionMode) bool {
	_, selected, _, _ := d.AccessibilitySelectionItem()
	if mode == fyne.AccessibilitySelectRemove {
		return !selected
	}
	if mode == fyne.AccessibilitySelectAdd && !selected {
		return false
	}
	d.choose()
	return true
}

func (d *calendarDay) choose() {
	d.owner.selectedDate = d.date
	d.owner.Refresh()
	if d.owner.OnChanged != nil {
		d.owner.OnChanged(d.date)
	}
}

func (d *calendarDay) TypedKey(event *fyne.KeyEvent) {
	delta := 0
	switch event.Name {
	case fyne.KeyEscape:
		if d.owner.dismiss != nil {
			d.owner.dismiss()
		}
		return
	case fyne.KeyLeft:
		delta = -1
	case fyne.KeyRight:
		delta = 1
	case fyne.KeyUp:
		delta = -daysPerWeek
	case fyne.KeyDown:
		delta = daysPerWeek
	case fyne.KeyHome:
		delta = 1 - d.date.Day()
	case fyne.KeyEnd:
		delta = time.Date(d.date.Year(), d.date.Month()+1, 1, 0, 0, 0, 0, d.date.Location()).AddDate(0, 0, -1).Day() - d.date.Day()
	default:
		d.Button.TypedKey(event)
		return
	}
	target := d.date.AddDate(0, 0, delta)
	c := fyne.CurrentApp().Driver().CanvasForObject(d.owner)
	if c == nil {
		return
	}
	if target.Month() != d.owner.currentTime.Month() || target.Year() != d.owner.currentTime.Year() {
		d.owner.currentTime = target
		d.owner.monthLabel.SetText(d.owner.monthYear())
		d.owner.dates.Objects = d.owner.calendarObjects()
		d.owner.dates.Refresh()
	}
	for _, object := range d.owner.dates.Objects {
		if day, ok := object.(*calendarDay); ok && day.date.Equal(target) {
			c.Focus(day)
			return
		}
	}
}

func (*calendarDay) AccessibilityFocusable() bool { return true }
func (d *calendarDay) AccessibilityFocus() bool {
	if !d.AccessibilityScrollIntoView() {
		return false
	}
	c := fyne.CurrentApp().Driver().CanvasForObject(d.owner)
	if c == nil {
		return false
	}
	for _, object := range d.owner.dates.Objects {
		if day, ok := object.(*calendarDay); ok && day.date.Format(calendarDateKeyFormat) == d.date.Format(calendarDateKeyFormat) {
			c.Focus(day)
			return c.Focused() == day
		}
	}
	return false
}

func (d *calendarDay) AccessibilityScrollIntoView() bool {
	if d.date.Year() != d.owner.currentTime.Year() || d.date.Month() != d.owner.currentTime.Month() {
		d.owner.currentTime = d.date
		d.owner.monthLabel.SetText(d.owner.monthYear())
		d.owner.dates.Objects = d.owner.calendarObjects()
		d.owner.dates.Refresh()
	}
	return true
}
