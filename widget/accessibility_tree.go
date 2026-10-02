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

// AccessibilityElements snapshots model keys, not recycled visual cells. Closed
// descendants retain keys but expose no semantics or commands. Model enumeration
// is linear in the number of nodes; no CreateNode or UpdateNode calls are made.
//
// Since: 2.9
func (t *Tree) AccessibilityElements() []fyne.AccessibilityElement {
	if t.IsBranch == nil {
		return nil
	}
	var elements []fyne.AccessibilityElement
	seen := make(map[TreeNodeID]bool)
	pad, y := t.Theme().Size(theme.SizeNamePadding), float32(0)
	var visit func(TreeNodeID, TreeNodeID, int, int, int, bool)
	visit = func(id, parent TreeNodeID, level, position, count int, hidden bool) {
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		branch := t.IsBranch(id)
		element := fyne.AccessibilityElement{Key: id, Parent: parent, Hidden: hidden}
		if !hidden {
			if y > 0 {
				y += pad
			}
			height := t.leafMinSize.Height
			if branch {
				height = t.branchMinSize.Height
			}
			item := &treeAccessibilityItem{
				owner: t, id: id, level: level, index: position, count: count,
				position: fyne.NewPos(0, y-t.offset.Y), size: fyne.NewSize(t.Size().Width, height),
			}
			element.Object = item
			if branch {
				element.Object = &treeAccessibilityBranch{item}
			}
			y += height
		}
		elements = append(elements, element)
		if !branch || t.ChildUIDs == nil {
			return
		}
		children := t.ChildUIDs(id)
		for i, child := range children {
			visit(child, id, level+1, i+1, len(children), hidden || !t.IsBranchOpen(id))
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
	return elements
}

// This is an immutable geometry snapshot with commands routed by model key.
// It is not a Widget and cannot construct a renderer.
type treeAccessibilityItem struct {
	owner               *Tree
	id                  TreeNodeID
	level, index, count int
	position            fyne.Position
	size                fyne.Size
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
	if _, found := t.accessibilityPath(i.id); !found {
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
	if _, found := t.accessibilityPath(i.id); !found {
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
	if _, found := i.owner.accessibilityPath(i.id); !found || i.owner.scroller == nil {
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
	if _, found := t.accessibilityPath(i.id); !found || !t.IsBranch(i.id) || i.id == t.Root || t.IsBranchOpen(i.id) == expanded {
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
