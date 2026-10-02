package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// AccessibilityLabel returns an empty fallback. Use SetAccessibilityInfo to name the tree.
//
// Since: 2.9
func (*Tree) AccessibilityLabel() string { return "" }

// AccessibilityRole returns the tree role.
//
// Since: 2.9
func (*Tree) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleTree }

// AccessibilitySelection describes Tree's optional single selection.
//
// Since: 2.9
func (*Tree) AccessibilitySelection() (multiple, required bool) { return false, false }

// AccessibilityActiveElement returns the item receiving keyboard commands.
//
// Since: 2.9
func (t *Tree) AccessibilityActiveElement() string { return t.currentHighlight }

// AccessibilityCollection separates tree topology from per-node semantics.
// Model keys include closed descendants, whose descriptions are not requested.
// No CreateNode or UpdateNode calls are made. Topology enumeration is O(N).
//
// Since: 2.9
func (t *Tree) AccessibilityCollection() fyne.AccessibilityCollection {
	source := &treeAccessibilitySource{children: make(map[string][]string), nodes: make(map[string]treeAccessibilityNode)}
	if t.IsBranch == nil {
		t.lifetimes.update(nil)
		return source
	}
	pad, y := t.Theme().Size(theme.SizeNamePadding), float32(0)
	var keys []string
	var visit func(TreeNodeID, TreeNodeID, int, int, int, bool)
	visit = func(id, parent TreeNodeID, level, position, count int, hidden bool) {
		if id == "" {
			return
		}
		source.children[parent] = append(source.children[parent], id)
		if _, seen := source.nodes[id]; seen {
			return // The adapter diagnoses duplicate keys and cycles.
		}
		branch := t.IsBranch(id)
		node := treeAccessibilityNode{
			parent: parent, hidden: hidden, branch: branch,
			item: treeAccessibilityItem{owner: t, id: id, level: level, index: position, count: count},
		}
		if !hidden {
			if len(keys) > 0 {
				y += pad
			}
			height := t.leafMinSize.Height
			if branch {
				height = t.branchMinSize.Height
			}
			node.item.position = fyne.NewPos(0, y-t.offset.Y)
			node.item.size = fyne.NewSize(t.Size().Width, height)
			y += height
		}
		source.nodes[id] = node
		keys = append(keys, id)
		if branch && t.ChildUIDs != nil {
			children := t.ChildUIDs(id)
			for i, child := range children {
				visit(child, id, level+1, i+1, len(children), hidden || !t.IsBranchOpen(id))
			}
		}
	}
	if t.Root != "" {
		visit(t.Root, "", 1, 1, 1, false)
	} else if t.ChildUIDs != nil {
		children := t.ChildUIDs(t.Root)
		for i, child := range children {
			visit(child, "", 1, i+1, len(children), false)
		}
	}
	t.lifetimes.update(keys)
	source.generations = t.lifetimes.generations
	return source
}

// AccessibilityElements returns a complete snapshot for callers of the original
// keyed API. Adapters use AccessibilityCollection in preference to this method.
//
// Since: 2.9
func (t *Tree) AccessibilityElements() []fyne.AccessibilityElement {
	source := t.AccessibilityCollection()
	var elements []fyne.AccessibilityElement
	seen := make(map[string]bool)
	var visit func(string)
	visit = func(parent string) {
		for index := 0; index < source.ChildCount(parent); index++ {
			key := source.ChildKey(parent, index)
			if seen[key] {
				continue
			}
			seen[key] = true
			if element, ok := source.Element(key); ok {
				elements = append(elements, element)
				visit(key)
			}
		}
	}
	visit("")
	return elements
}

type treeAccessibilityNode struct {
	parent         string
	hidden, branch bool
	item           treeAccessibilityItem
}

type treeAccessibilitySource struct {
	children    map[string][]string
	nodes       map[string]treeAccessibilityNode
	generations map[string]uint64
}

func (s *treeAccessibilitySource) ChildCount(parent string) int { return len(s.children[parent]) }
func (s *treeAccessibilitySource) ChildKey(parent string, index int) string {
	children := s.children[parent]
	if index < 0 || index >= len(children) {
		return ""
	}
	return children[index]
}

func (s *treeAccessibilitySource) Element(key string) (fyne.AccessibilityElement, bool) {
	node, ok := s.nodes[key]
	if !ok {
		return fyne.AccessibilityElement{}, false
	}
	element := fyne.AccessibilityElement{Key: key, Parent: node.parent, Hidden: node.hidden, Generation: s.generations[key]}
	if !node.hidden {
		node.item.generation = element.Generation
		element.Object = &node.item
		if node.branch {
			element.Object = &treeAccessibilityBranch{&node.item}
		}
	}
	return element, true
}

// This is an immutable geometry snapshot with commands routed by model key.
// It is not a Widget and cannot construct a renderer.
type treeAccessibilityItem struct {
	owner               *Tree
	id                  TreeNodeID
	generation          uint64
	level, index, count int
	position            fyne.Position
	size                fyne.Size
}

func (i *treeAccessibilityItem) attached() bool {
	if i.owner.lifetimes.generations[i.id] != i.generation {
		return false
	}
	_, found := i.owner.accessibilityPath(i.id)
	return found
}

func (i *treeAccessibilityItem) AccessibilityLabel() string { return i.id }
func (*treeAccessibilityItem) AccessibilityRole() fyne.AccessibleRole {
	return fyne.AccessibleRoleTreeItem
}

func (i *treeAccessibilityItem) AccessibilityInfo() fyne.AccessibilityInfo {
	if f := i.owner.DescribeNode; f != nil {
		return f(i.id)
	}
	return fyne.AccessibilityInfo{}
}

func (i *treeAccessibilityItem) AccessibilityHierarchy() (level, position, count int) {
	return i.level, i.index, i.count
}

func (i *treeAccessibilityItem) AccessibilitySelectionItem() (owner fyne.CanvasObject, selected bool, position, count int) {
	return i.owner.super(), contains(i.owner.selected, i.id), i.index, i.count
}

func (i *treeAccessibilityItem) AccessibilitySelect(mode fyne.AccessibilitySelectionMode) bool {
	t := i.owner
	if !i.attached() {
		return false
	}
	switch mode {
	case fyne.AccessibilitySelectRemove:
		t.Unselect(i.id)
	case fyne.AccessibilitySelectAdd:
		if len(t.selected) > 0 && !contains(t.selected, i.id) {
			return false
		}
		t.Select(i.id)
	case fyne.AccessibilitySelectReplace:
		t.Select(i.id)
	default:
		return false
	}
	return true
}
func (*treeAccessibilityItem) AccessibilityFocusable() bool { return true }
func (i *treeAccessibilityItem) AccessibilityFocus() bool {
	t := i.owner
	if !i.attached() {
		return false
	}
	c := fyne.CurrentApp().Driver().CanvasForObject(t.super())
	f, ok := t.super().(fyne.Focusable)
	if c == nil || !ok {
		return false
	}
	changed := t.currentHighlight != i.id
	t.setItemFocus(i.id)
	if c.Focused() == f {
		if changed && t.OnHighlighted != nil {
			t.OnHighlighted(i.id)
		}
	} else {
		c.Focus(f)
	}
	return c.Focused() == f
}

func (i *treeAccessibilityItem) AccessibilityScrollIntoView() bool {
	if !i.attached() || i.owner.scroller == nil {
		return false
	}
	i.owner.ScrollTo(i.id)
	return true
}
func (i *treeAccessibilityItem) Position() fyne.Position { return i.position }
func (i *treeAccessibilityItem) Size() fyne.Size         { return i.size }
func (i *treeAccessibilityItem) MinSize() fyne.Size      { return i.size }
func (*treeAccessibilityItem) Move(fyne.Position)        {}
func (*treeAccessibilityItem) Resize(fyne.Size)          {}
func (*treeAccessibilityItem) Show()                     {}
func (*treeAccessibilityItem) Hide()                     {}
func (*treeAccessibilityItem) Refresh()                  {}
func (*treeAccessibilityItem) Visible() bool             { return true }

type treeAccessibilityBranch struct{ *treeAccessibilityItem }

func (i *treeAccessibilityBranch) AccessibilityExpanded() bool { return i.owner.IsBranchOpen(i.id) }
func (i *treeAccessibilityBranch) AccessibilitySetExpanded(expanded bool) {
	t := i.owner
	if !i.attached() || !t.IsBranch(i.id) || i.id == t.Root || t.IsBranchOpen(i.id) == expanded {
		return
	}
	if expanded {
		t.OpenBranch(i.id)
	} else {
		t.CloseBranch(i.id)
	}
}

// Guard model cycles and duplicates even when a malformed model has not yet been
// rendered. This path is also used to revalidate direct semantic commands.
func (t *Tree) accessibilityPath(target TreeNodeID) ([]TreeNodeID, bool) {
	seen := make(map[TreeNodeID]bool)
	var path []TreeNodeID
	var visit func(TreeNodeID) bool
	visit = func(id TreeNodeID) bool {
		if seen[id] {
			return false
		}
		seen[id] = true
		path = append(path, id)
		if id == target {
			return true
		}
		if t.IsBranch != nil && t.IsBranch(id) && t.ChildUIDs != nil {
			for _, child := range t.ChildUIDs(id) {
				if visit(child) {
					return true
				}
			}
		}
		path = path[:len(path)-1]
		return false
	}
	found := visit(t.Root)
	return path, found
}

func (t *Tree) reconcileAccessibilityState() {
	if t.currentHighlight != "" {
		path, found := t.accessibilityPath(t.currentHighlight)
		next := t.currentHighlight
		if !found {
			next = t.Root
			if next == "" && t.ChildUIDs != nil {
				if children := t.ChildUIDs(t.Root); len(children) > 0 {
					next = children[0]
				}
			}
		} else {
			for _, id := range path[:len(path)-1] {
				if !t.IsBranchOpen(id) {
					next = id
					break
				}
			}
		}
		if next != t.currentHighlight {
			t.currentHighlight = next
			if t.OnHighlighted != nil {
				t.OnHighlighted(next)
			}
		}
	}
	if len(t.selected) > 0 {
		id := t.selected[0]
		if _, found := t.accessibilityPath(id); !found {
			t.selected = nil
			if t.OnUnselected != nil {
				t.OnUnselected(id)
			}
		}
	}
}
