package widget

import (
	"slices"
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
)

// AccessibilityLabel returns an empty fallback. Use SetAccessibilityInfo to name the list.
//
// Since: 2.9
func (*List) AccessibilityLabel() string { return "" }

// AccessibilityRole returns the list role.
//
// Since: 2.9
func (*List) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleList }

// AccessibilitySelection describes List's optional single selection.
//
// Since: 2.9
func (*List) AccessibilitySelection() (multiple, required bool) { return false, false }

// AccessibilityActiveElement returns the model key receiving keyboard commands.
//
// Since: 2.9
func (l *List) AccessibilityActiveElement() string {
	if l.Length == nil || l.currentHighlight < 0 || l.currentHighlight >= l.Length() {
		return ""
	}
	return l.keyForIndex(l.currentHighlight)
}

// AccessibilityCollection provides indexed topology and model-based semantics,
// independently of pooled renderer cells. Positional keys are decimal ListItemIDs
// unless ItemKey is set. Topology and geometry are cached until Refresh or a
// geometry change. Descriptions are requested only for materialized semantics.
//
// Since: 2.9
func (l *List) AccessibilityCollection() fyne.AccessibilityCollection {
	pad := l.Theme().Size(theme.SizeNamePadding)
	if cached := l.accessibilityCache; cached != nil && cached.height == l.itemMin.Height && cached.padding == pad {
		return cached
	}
	keys, indices := l.modelKeys()
	if l.itemKeys == nil && l.ItemKey != nil {
		// The first semantic query can precede rendering. Establish the model
		// baseline without changing selection, highlight or creating cells.
		l.itemKeys, l.keyed = keys, true
	}
	l.lifetimes.update(keys)
	l.accessibilityRevision++
	source := &listAccessibilitySource{
		owner: l, keys: keys, indices: indices,
		height: l.itemMin.Height, padding: pad,
		revision:    l.accessibilityRevision,
		generations: l.lifetimes.generations, positions: make([]float32, len(keys)), heights: make([]float32, len(keys)),
	}
	y := float32(0)
	for index := range keys {
		height := l.itemMin.Height
		if h, ok := l.itemHeights[index]; ok {
			height = h
		}
		source.positions[index], source.heights[index] = y, height
		y += height + pad
	}
	l.accessibilityCache = source
	return source
}

type listAccessibilitySource struct {
	owner              *List
	keys               []string
	indices            map[string]int
	generations        map[string]uint64
	positions, heights []float32
	height, padding    float32
	revision           uint64
}

func (s *listAccessibilitySource) Revision() uint64 { return s.revision }

func (s *listAccessibilitySource) ViewportKeys() []string {
	top, bottom := s.owner.offsetY, s.owner.offsetY+s.owner.Size().Height
	first := sort.Search(len(s.keys), func(i int) bool { return s.positions[i]+s.heights[i] >= top })
	last := sort.Search(len(s.keys), func(i int) bool { return s.positions[i] > bottom })
	return s.keys[first:last]
}

func (s *listAccessibilitySource) SelectedKeys() []string {
	var keys []string
	for _, index := range s.owner.selected {
		if index >= 0 && index < len(s.keys) {
			keys = append(keys, s.keys[index])
		}
	}
	return keys
}

func (s *listAccessibilitySource) Index(key string) (string, int, bool) {
	index, ok := s.indices[key]
	return "", index, ok && index >= 0
}

func (s *listAccessibilitySource) ChildCount(parent string) int {
	if parent != "" {
		return 0
	}
	return len(s.keys)
}

func (s *listAccessibilitySource) ChildKey(parent string, index int) string {
	if parent != "" || index < 0 || index >= len(s.keys) {
		return ""
	}
	return s.keys[index]
}

func (s *listAccessibilitySource) Element(key string) (fyne.AccessibilityElement, bool) {
	index, ok := s.indices[key]
	if !ok || index < 0 {
		return fyne.AccessibilityElement{}, false
	}
	generation := s.generations[key]
	item := &listAccessibilityItem{
		owner: s.owner, key: key, index: index, count: len(s.keys), generation: generation,
		position: fyne.NewPos(0, s.positions[index]-s.owner.offsetY), size: fyne.NewSize(s.owner.Size().Width, s.heights[index]),
	}
	return fyne.AccessibilityElement{Key: key, Object: item, Generation: generation}, true
}

type listAccessibilityItem struct {
	owner        *List
	key          string
	generation   uint64
	index, count int
	position     fyne.Position
	size         fyne.Size
}

func (i *listAccessibilityItem) AccessibilityLabel() string {
	return lang.X("accessibility.list.item", "Item {{.Number}}", map[string]any{"Number": i.index + 1})
}

func (*listAccessibilityItem) AccessibilityRole() fyne.AccessibleRole {
	return fyne.AccessibleRoleListItem
}

func (i *listAccessibilityItem) AccessibilityInfo() fyne.AccessibilityInfo {
	if f := i.owner.DescribeItem; f != nil {
		return f(i.index)
	}
	return fyne.AccessibilityInfo{}
}

func (i *listAccessibilityItem) AccessibilitySelectionItem() (owner fyne.CanvasObject, selected bool, position, count int) {
	return i.owner.super(), slices.Contains(i.owner.selected, i.index), i.index + 1, i.count
}

func (i *listAccessibilityItem) resolve() int {
	source, ok := i.owner.AccessibilityCollection().(*listAccessibilitySource)
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

func (i *listAccessibilityItem) AccessibilitySelect(mode fyne.AccessibilitySelectionMode) bool {
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
func (*listAccessibilityItem) AccessibilityFocusable() bool { return true }
func (i *listAccessibilityItem) AccessibilityFocus() bool {
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

func (i *listAccessibilityItem) AccessibilityScrollIntoView() bool {
	index := i.resolve()
	if index < 0 || i.owner.scroller == nil {
		return false
	}
	i.owner.ScrollTo(index)
	return true
}
func (i *listAccessibilityItem) Position() fyne.Position { return i.position }
func (i *listAccessibilityItem) Size() fyne.Size         { return i.size }
func (i *listAccessibilityItem) MinSize() fyne.Size      { return i.size }
func (*listAccessibilityItem) Move(fyne.Position)        {}
func (*listAccessibilityItem) Resize(fyne.Size)          {}
func (*listAccessibilityItem) Show()                     {}
func (*listAccessibilityItem) Hide()                     {}
func (*listAccessibilityItem) Refresh()                  {}
func (*listAccessibilityItem) Visible() bool             { return true }
