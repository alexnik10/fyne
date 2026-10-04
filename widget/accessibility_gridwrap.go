package widget

import (
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
)

// AccessibilityLabel returns an empty fallback. Use SetAccessibilityInfo to name the wrapping grid.
//
// Since: 2.9
func (*GridWrap) AccessibilityLabel() string { return "" }

// AccessibilityRole returns the list role for a wrapping grid.
//
// Since: 2.9
func (*GridWrap) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleList }

// AccessibilitySelection describes GridWrap's optional single selection.
//
// Since: 2.9
func (*GridWrap) AccessibilitySelection() (multiple, required bool) { return false, false }

// AccessibilityActiveElement returns the model key receiving keyboard commands.
//
// Since: 2.9
func (l *GridWrap) AccessibilityActiveElement() string {
	if l.Length == nil || l.currentHighlight < 0 || l.currentHighlight >= l.Length() {
		return ""
	}
	return l.keyForIndex(l.currentHighlight)
}

// AccessibilityCollection provides indexed topology and model-based semantics,
// independently of pooled renderer cells. Positional keys are decimal GridWrapItemIDs
// unless ItemKey is set. Topology and geometry are cached until Refresh or a
// geometry change. Descriptions are requested only for materialized semantics.
//
// Since: 2.9
func (l *GridWrap) AccessibilityCollection() fyne.AccessibilityCollection {
	base := l.accessibilityBaseCollection()
	return l.accessibilityWithChildren(base)
}

func (l *GridWrap) accessibilityBaseCollection() fyne.AccessibilityCollectionView {
	pad := l.Theme().Size(theme.SizeNamePadding)
	columns := max(1, l.ColumnCount())
	if cached := l.accessibilityCache; cached != nil && cached.height == l.itemMin.Height && cached.width == l.itemMin.Width && cached.columns == columns && cached.padding == pad {
		return cached
	}
	keys, indices := l.modelKeys()
	if l.itemKeys == nil && l.ItemKey != nil {
		l.itemKeys, l.keyed = keys, true
	}
	l.lifetimes.update(keys)
	l.accessibilityRevision++
	source := &gridWrapAccessibilitySource{owner: l, keys: keys, indices: indices, generations: l.lifetimes.generations, height: l.itemMin.Height, width: l.itemMin.Width, padding: pad, columns: columns, revision: l.accessibilityRevision}
	l.accessibilityCache = source
	return source
}

type gridWrapAccessibilitySource struct {
	owner                  *GridWrap
	keys                   []string
	indices                map[string]int
	generations            map[string]uint64
	height, width, padding float32
	columns                int
	revision               uint64
}

func (s *gridWrapAccessibilitySource) Revision() uint64 { return s.revision }
func (s *gridWrapAccessibilitySource) ViewportKeys() []string {
	stride := s.height + s.padding
	if stride <= 0 {
		return nil
	}
	first := max(0, int(s.owner.offsetY/stride)*s.columns)
	last := min(len(s.keys), (int((s.owner.offsetY+s.owner.Size().Height)/stride)+1)*s.columns)
	return s.keys[min(first, last):last]
}

func (s *gridWrapAccessibilitySource) SelectedKeys() []string {
	var keys []string
	for _, index := range s.owner.selected {
		if index >= 0 && index < len(s.keys) {
			keys = append(keys, s.keys[index])
		}
	}
	return keys
}

func (s *gridWrapAccessibilitySource) Index(key string) (string, int, bool) {
	index, ok := s.indices[key]
	return "", index, ok && index >= 0
}

func (s *gridWrapAccessibilitySource) ChildCount(parent string) int {
	if parent != "" {
		return 0
	}
	return len(s.keys)
}

func (s *gridWrapAccessibilitySource) ChildKey(parent string, index int) string {
	if parent != "" || index < 0 || index >= len(s.keys) {
		return ""
	}
	return s.keys[index]
}

func (s *gridWrapAccessibilitySource) Element(key string) (fyne.AccessibilityElement, bool) {
	index, ok := s.indices[key]
	if !ok || index < 0 {
		return fyne.AccessibilityElement{}, false
	}
	generation := s.generations[key]
	item := &gridWrapAccessibilityItem{
		owner: s.owner, key: key, index: index, count: len(s.keys), generation: generation,
		position: fyne.NewPos(float32(index%s.columns)*(s.width+s.padding), float32(index/s.columns)*(s.height+s.padding)-s.owner.offsetY), size: fyne.NewSize(s.width, s.height),
	}
	return fyne.AccessibilityElement{Key: key, Object: item, Generation: generation}, true
}

type gridWrapAccessibilityItem struct {
	owner        *GridWrap
	key          string
	generation   uint64
	index, count int
	position     fyne.Position
	size         fyne.Size
}

func (i *gridWrapAccessibilityItem) AccessibilityLabel() string {
	return lang.X("accessibility.list.item", "Item {{.Number}}", map[string]any{"Number": i.index + 1})
}

func (*gridWrapAccessibilityItem) AccessibilityRole() fyne.AccessibleRole {
	return fyne.AccessibleRoleListItem
}

func (i *gridWrapAccessibilityItem) AccessibilityInfo() fyne.AccessibilityInfo {
	if f := i.owner.DescribeItem; f != nil {
		return f(i.index)
	}
	return fyne.AccessibilityInfo{}
}

func (i *gridWrapAccessibilityItem) AccessibilitySelectionItem() (owner fyne.CanvasObject, selected bool, position, count int) {
	return i.owner.super(), slices.Contains(i.owner.selected, i.index), i.index + 1, i.count
}

func (i *gridWrapAccessibilityItem) resolve() int {
	source, ok := i.owner.accessibilityBaseCollection().(*gridWrapAccessibilitySource)
	if !ok {
		return -1
	}
	if i.owner.lifetimes.generations[i.key] != i.generation {
		return -1
	}
	if index, ok := source.indices[i.key]; ok {
		return index
	}
	return -1
}

func (i *gridWrapAccessibilityItem) AccessibilitySelect(mode fyne.AccessibilitySelectionMode) bool {
	index := i.resolve()
	if index < 0 {
		return false
	}
	l := i.owner
	switch mode {
	case fyne.AccessibilitySelectRemove:
		l.Unselect(index)
	case fyne.AccessibilitySelectAdd:
		if len(l.selected) > 0 && !slices.Contains(l.selected, index) {
			return false
		}
		l.Select(index)
	case fyne.AccessibilitySelectReplace:
		l.Select(index)
	default:
		return false
	}
	return true
}
func (*gridWrapAccessibilityItem) AccessibilityFocusable() bool { return true }
func (i *gridWrapAccessibilityItem) AccessibilityFocus() bool {
	index, l := i.resolve(), i.owner
	if index < 0 {
		return false
	}
	c := fyne.CurrentApp().Driver().CanvasForObject(l.super())
	f, ok := l.super().(fyne.Focusable)
	if c == nil || !ok {
		return false
	}
	previous := l.currentHighlight
	l.currentHighlight = index
	l.scrollWithoutItemCheckTo(index)
	l.RefreshItem(previous)
	l.RefreshItem(index)
	if c.Focused() == f {
		if previous != index && l.OnHighlighted != nil {
			l.OnHighlighted(index)
		}
	} else {
		c.Focus(f)
	}
	return c.Focused() == f
}

func (i *gridWrapAccessibilityItem) AccessibilityScrollIntoView() bool {
	index := i.resolve()
	if index < 0 || i.owner.scroller == nil {
		return false
	}
	i.owner.ScrollTo(index)
	return true
}
func (i *gridWrapAccessibilityItem) Position() fyne.Position { return i.position }
func (i *gridWrapAccessibilityItem) Size() fyne.Size         { return i.size }
func (i *gridWrapAccessibilityItem) MinSize() fyne.Size      { return i.size }
func (*gridWrapAccessibilityItem) Move(fyne.Position)        {}
func (*gridWrapAccessibilityItem) Resize(fyne.Size)          {}
func (*gridWrapAccessibilityItem) Show()                     {}
func (*gridWrapAccessibilityItem) Hide()                     {}
func (*gridWrapAccessibilityItem) Refresh()                  {}
func (*gridWrapAccessibilityItem) Visible() bool             { return true }

// AccessibilityChildren suppresses pooled renderer cells in favor of the model.
// Since: 2.9
func (*GridWrap) AccessibilityChildren() []fyne.CanvasObject { return nil }

// AccessibilityScroll describes the wrapping grid viewport.
// Since: 2.9
func (l *GridWrap) AccessibilityScroll() fyne.AccessibilityScrollInfo {
	return l.scroller.AccessibilityScroll()
}

// AccessibilityScrollTo changes the viewport without selecting or focusing.
// Since: 2.9
func (l *GridWrap) AccessibilityScrollTo(offset fyne.Position) bool {
	return l.scroller.AccessibilityScrollTo(offset)
}
