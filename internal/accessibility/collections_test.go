package accessibility_test

import (
	"fmt"
	"strconv"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/internal/accessibility"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDemandCollectionBoundsAndLifetime(t *testing.T) {
	test.NewTempApp(t)
	count, keysRead, described, created := 100000, 0, 0, 0
	list := widget.NewList(func() int { return count }, func() fyne.CanvasObject { created++; return widget.NewLabel("Template") }, func(int, fyne.CanvasObject) {})
	list.ItemKey = func(i int) string { keysRead++; return strconv.Itoa(i) }
	list.DescribeItem = func(i int) fyne.AccessibilityInfo { described++; return fyne.AccessibilityInfo{Name: strconv.Itoa(i)} }
	w := test.NewWindow(list)
	defer w.Close()
	w.Resize(fyne.NewSize(300, 300))
	var tree accessibility.Tree
	roots := []accessibility.Root{{Object: list}}
	tree.Build(roots, nil)
	beforeCreated := created
	keysRead, described = 0, 0
	nodes := tree.Build(roots, nil)
	require.Less(t, len(nodes), 20)
	assert.Less(t, described, 20)
	assert.Less(t, keysRead, 3, "a warm snapshot must not scan model keys")
	_, found := tree.NodeForElement(list, "99999")
	assert.False(t, found, "offscreen rows start virtualized")
	require.True(t, tree.RequestElement(list, "99999"))
	tree.Build(roots, nil)
	last, found := tree.NodeForElement(list, "99999")
	require.True(t, found)
	assert.Zero(t, last.BoundsSize.Height)
	assert.False(t, last.Selected)
	assert.True(t, last.VirtualizedItem)
	assert.Equal(t, beforeCreated, created)
	assert.Nil(t, w.Canvas().Focused())
	for i := 100; i < 200; i++ {
		require.True(t, tree.RequestElement(list, strconv.Itoa(i)))
	}
	nodes = tree.Build(roots, nil)
	assert.Less(t, len(nodes), 85, "LRU bounds heavy semantic snapshots")
	_, found = tree.NodeForElement(list, "99999")
	assert.False(t, found)
	require.True(t, tree.Realize(last.ID))
	tree.Build(roots, nil)
	realized, found := tree.NodeForElement(list, "99999")
	require.True(t, found)
	assert.Equal(t, last.ID, realized.ID)
	assert.Equal(t, beforeCreated, created)

	list.Select(99999)
	list.ScrollTo(0)
	tree.Build(roots, nil)
	selected, found := tree.NodeForElement(list, "99999")
	require.True(t, found)
	assert.True(t, selected.Selected, "offscreen selection is pinned")
	count--
	list.Refresh()
	count++
	list.Refresh()
	tree.Build(roots, nil)
	assert.False(t, tree.Realize(last.ID), "a retired key cannot be realized as a new record")
	require.True(t, tree.RequestElement(list, "99999"))
	tree.Build(roots, nil)
	replacement, _ := tree.NodeForElement(list, "99999")
	assert.NotEqual(t, last.ID, replacement.ID)
	assert.False(t, replacement.Selected)
	tree.Build([]accessibility.Root{{Object: list, Suppressed: true}}, nil)
	assert.False(t, tree.Realize(replacement.ID), "modal/background scope must be respected")
	tree.Build(roots, nil)
	require.True(t, tree.Realize(replacement.ID))
}

func TestDemandCollectionFindAndTreeCollapse(t *testing.T) {
	test.NewTempApp(t)
	keys := make([]string, 1000)
	for i := range keys {
		keys[i] = strconv.Itoa(i)
	}
	model := map[string][]string{"": {"folder"}, "folder": keys}
	widgetTree := widget.NewTreeWithStrings(model)
	other := widget.NewList(func() int { return 1 }, func() fyne.CanvasObject { return widget.NewLabel("Other") }, func(int, fyne.CanvasObject) {})
	w := test.NewWindow(container.NewVBox(widgetTree, other))
	defer w.Close()
	w.Resize(fyne.NewSize(300, 300))
	var tree accessibility.Tree
	roots := accessibility.Roots(w.Canvas())
	tree.Build(roots, nil)
	folder, _ := tree.NodeForElement(widgetTree, "folder")
	_, key, valid := tree.FindItem(folder.ID, 0, accessibility.FindName, "999")
	assert.True(t, valid)
	assert.Empty(t, key, "collapsed descendants cannot be discovered")
	widgetTree.OpenBranch("folder")
	tree.Build(roots, nil)
	owner, key, valid := tree.FindItem(folder.ID, 0, accessibility.FindName, "999")
	require.True(t, valid)
	require.Equal(t, "999", key)
	require.True(t, tree.RequestElement(owner, key))
	tree.Build(roots, nil)
	last, _ := tree.NodeForElement(widgetTree, "999")
	assert.False(t, last.Focused)
	assert.False(t, last.Selected)
	_, key, valid = tree.FindItem(folder.ID, last.ID, accessibility.FindNext, "ignored")
	assert.True(t, valid)
	assert.Empty(t, key)
	_, key, valid = tree.FindItem(folder.ID, 0, accessibility.FindAutomationID, fmt.Sprintf("fyne_%d", last.ID))
	assert.True(t, valid)
	assert.Equal(t, "999", key)
	otherOwner, _ := tree.NodeForObject(other)
	_, _, valid = tree.FindItem(otherOwner.ID, last.ID, accessibility.FindNext, "")
	assert.False(t, valid)
	widgetTree.Select("999")
	tree.Build(roots, nil)
	_, key, valid = tree.FindItem(folder.ID, 0, accessibility.FindSelected, "true")
	assert.True(t, valid)
	assert.Equal(t, "999", key)
	widgetTree.CloseBranch("folder")
	tree.Build(roots, nil)
	assert.False(t, tree.Realize(last.ID))
	widgetTree.OpenBranch("folder")
	tree.Build(roots, nil)
	require.True(t, tree.Realize(last.ID), "collapse preserves model identity")
	assert.Empty(t, tree.Issues)
}

type demandSource struct{ indexedSource }

func (*demandSource) Revision() uint64       { return 1 }
func (*demandSource) ViewportKeys() []string { return nil }
func (*demandSource) SelectedKeys() []string { return nil }
func (s *demandSource) Index(key string) (string, int, bool) {
	e, ok := s.elements[key]
	if !ok {
		return "", 0, false
	}
	for i, child := range s.children[e.Parent] {
		if child == key {
			return e.Parent, i, true
		}
	}
	return "", 0, false
}

func TestDemandRequestValidatesAncestors(t *testing.T) {
	test.NewTempApp(t)
	label, invisible := widget.NewLabel("Child"), widget.NewLabel("Hidden")
	invisible.Hide()
	source := &demandSource{indexedSource{
		children: map[string][]string{"": {"hidden", "invisible"}, "hidden": {"child"}, "cycle": {"cycle"}},
		elements: map[string]fyne.AccessibilityElement{
			"hidden":    {Key: "hidden", Hidden: true},
			"child":     {Key: "child", Parent: "hidden", Object: label},
			"invisible": {Key: "invisible", Object: invisible},
			"cycle":     {Key: "cycle", Parent: "cycle", Object: label},
		},
	}}
	owner := &indexedOwner{source: source}
	var tree accessibility.Tree
	tree.Build([]accessibility.Root{{Object: owner}}, nil)
	for _, key := range []string{"", "missing", "hidden", "child", "invisible", "cycle"} {
		assert.False(t, tree.RequestElement(owner, key), key)
	}
}

func TestGridLookupScopeAndEviction(t *testing.T) {
	test.NewTempApp(t)
	table := widget.NewTableWithHeaders(func() (int, int) { return 200, 3 }, func() fyne.CanvasObject { return widget.NewLabel("Template") }, func(widget.TableCellID, fyne.CanvasObject) {})
	w := test.NewWindow(table)
	defer w.Close()
	w.Resize(fyne.NewSize(300, 150))
	var tree accessibility.Tree
	roots := []accessibility.Root{{Object: table}}
	tree.Build(roots, nil)
	root, ok := tree.NodeForObject(table)
	require.True(t, ok)
	owner, key, valid := tree.FindItem(root.ID, 0, accessibility.FindGridCell, "199,2")
	require.True(t, valid)
	require.Same(t, table, owner)
	require.Equal(t, table.AccessibilityCellKey(199, 2), key)
	require.True(t, tree.RequestElement(owner, key))
	tree.Build(roots, nil)
	last, ok := tree.NodeForElement(table, key)
	require.True(t, ok)
	for row := 0; row < 100; row++ {
		require.True(t, tree.RequestElement(table, table.AccessibilityCellKey(row, 0)))
	}
	tree.Build(roots, nil)
	_, ok = tree.NodeForElement(table, key)
	assert.False(t, ok)
	require.True(t, tree.Realize(last.ID))
	tree.Build(roots, nil)
	again, _ := tree.NodeForElement(table, key)
	assert.Equal(t, last.ID, again.ID)
	_, header, valid := tree.FindItem(root.ID, 0, accessibility.FindGridCell, "-1,2")
	assert.True(t, valid)
	assert.Equal(t, table.AccessibilityCellKey(-1, 2), header)
	for _, coordinate := range []string{"200,2", "0,3", "-2,0", "-1,-1", "x,0"} {
		_, _, valid = tree.FindItem(root.ID, 0, accessibility.FindGridCell, coordinate)
		assert.False(t, valid)
	}
	tree.Build([]accessibility.Root{{Object: table, Suppressed: true}}, nil)
	_, _, valid = tree.FindItem(root.ID, 0, accessibility.FindGridCell, "0,0")
	assert.False(t, valid)
	assert.False(t, tree.Realize(last.ID))
}
