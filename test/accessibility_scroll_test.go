package test_test

import (
	"fmt"
	"image/color"
	"math"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccessibilityScrollViewportCommandsAndScope(t *testing.T) {
	test.NewTempApp(t)
	extent := canvas.NewRectangle(color.Transparent)
	extent.SetMinSize(fyne.NewSize(1000, 800))
	button := widget.NewButton("Far corner", nil)
	button.Move(fyne.NewPos(800, 650))
	button.Resize(button.MinSize())
	scroll := container.NewScroll(container.NewWithoutLayout(extent, button))
	w := test.NewWindow(scroll)
	w.SetPadded(false)
	defer w.Close()
	w.Resize(fyne.NewSize(200, 150))
	w.Canvas().Focus(button)
	tree := test.NewAccessibilityTree(w.Canvas())
	n, ok := tree.Node(scroll)
	require.True(t, ok)
	require.True(t, n.Scroll)
	assert.Equal(t, 0.0, n.HorizontalScrollPercent)
	assert.Equal(t, 0.0, n.VerticalScrollPercent)
	assert.InDelta(t, scroll.Size().Width/10, n.HorizontalViewSize, 0.001)
	assert.InDelta(t, scroll.Size().Height/8, n.VerticalViewSize, 0.001)
	calls := 0
	scroll.OnScrolled = func(fyne.Position) { calls++ }
	require.True(t, tree.SetScrollPercent(n.ID, 50, 100))
	assert.Equal(t, fyne.NewPos(400, 650), scroll.Offset)
	assert.Equal(t, 1, calls)
	require.True(t, tree.SetScrollPercent(n.ID, 50, 100))
	assert.Equal(t, 1, calls, "unchanged offsets must not emit another callback")
	require.True(t, tree.Scroll(n.ID, fyne.AccessibilityScrollNone, fyne.AccessibilityScrollLargeIncrement))
	assert.Equal(t, 1, calls, "edge is a successful no-op")
	require.True(t, tree.Scroll(n.ID, fyne.AccessibilityScrollSmallDecrement, fyne.AccessibilityScrollLargeDecrement))
	assert.Less(t, scroll.Offset.X, float32(400))
	assert.InDelta(t, 650-150*0.95, scroll.Offset.Y, 0.001)
	before := scroll.Offset
	for _, bad := range []float64{-2, 101, math.NaN(), math.Inf(1), math.Inf(-1)} {
		assert.False(t, tree.SetScrollPercent(n.ID, 0, bad))
		assert.False(t, tree.SetScrollPercent(n.ID, bad, 0))
		assert.Equal(t, before, scroll.Offset)
	}
	assert.False(t, tree.Scroll(n.ID, 255, fyne.AccessibilityScrollSmallIncrement))
	assert.Equal(t, before, scroll.Offset)
	scroll.Direction = container.ScrollVerticalOnly
	assert.False(t, tree.SetScrollPercent(n.ID, 0, 0))
	assert.False(t, tree.Scroll(n.ID, fyne.AccessibilityScrollSmallIncrement, fyne.AccessibilityScrollSmallIncrement))
	assert.Equal(t, before, scroll.Offset, "unsupported axis rejects both changes")
	n, _ = tree.Node(scroll)
	assert.Equal(t, -1.0, n.HorizontalScrollPercent)
	assert.Equal(t, 100.0, n.HorizontalViewSize)
	require.True(t, tree.SetScrollPercent(n.ID, -1, 0))
	assert.Equal(t, before.X, scroll.Offset.X)
	assert.Zero(t, scroll.Offset.Y)
	scroll.Direction = container.ScrollBoth
	child, _ := tree.Node(button)
	require.True(t, child.ScrollItem)
	require.True(t, tree.Perform(child.ID, test.AccessibilityScrollIntoView, "", 0))
	child, _ = tree.Node(button)
	assert.Equal(t, child.Size, child.BoundsSize)
	assert.Same(t, button, w.Canvas().Focused())
	assert.Equal(t, child.ID, semanticNamed(t, tree, fyne.AccessibleRoleButton, "Far corner").ID)
	popup := widget.NewModalPopUp(widget.NewLabel("Modal"), w.Canvas())
	popup.Show()
	before = scroll.Offset
	assert.False(t, tree.SetScrollPercent(n.ID, 0, 0))
	assert.Equal(t, before, scroll.Offset)
	popup.Hide()
	scroll.Hide()
	assert.False(t, tree.Scroll(n.ID, fyne.AccessibilityScrollSmallIncrement, fyne.AccessibilityScrollNone))
	scroll.Show()
	w.Resize(fyne.NewSize(1200, 1000))
	n, _ = tree.Node(scroll)
	assert.Equal(t, -1.0, n.HorizontalScrollPercent)
	assert.Equal(t, -1.0, n.VerticalScrollPercent)
	assert.Equal(t, 100.0, n.VerticalViewSize)
	assert.True(t, tree.SetScrollPercent(n.ID, -1, -1))
	assert.True(t, tree.Scroll(n.ID, fyne.AccessibilityScrollNone, fyne.AccessibilityScrollNone))
	assert.False(t, tree.SetScrollPercent(n.ID, 0, -1))
	w.SetContent(widget.NewLabel("Replacement"))
	assert.False(t, tree.SetScrollPercent(n.ID, -1, -1))
}

func TestAccessibilityNestedScrollIntoView(t *testing.T) {
	test.NewTempApp(t)
	target := widget.NewButton("Nested target", nil)
	target.Resize(target.MinSize())
	target.Move(fyne.NewPos(500, 500))
	inside := canvas.NewRectangle(color.Transparent)
	inside.SetMinSize(fyne.NewSize(700, 700))
	inner := container.NewScroll(container.NewWithoutLayout(inside, target))
	inner.Resize(fyne.NewSize(200, 200))
	inner.Move(fyne.NewPos(400, 400))
	outside := canvas.NewRectangle(color.Transparent)
	outside.SetMinSize(fyne.NewSize(800, 800))
	outer := container.NewScroll(container.NewWithoutLayout(outside, inner))
	w := test.NewWindow(outer)
	defer w.Close()
	w.Resize(fyne.NewSize(300, 300))
	tree := test.NewAccessibilityTree(w.Canvas())
	n, ok := tree.Node(target)
	require.True(t, ok)
	require.True(t, tree.Perform(n.ID, test.AccessibilityScrollIntoView, "", 0))
	n, _ = tree.Node(target)
	assert.Equal(t, n.Size, n.BoundsSize)
	assert.Greater(t, inner.Offset.Y, float32(0))
	assert.Greater(t, outer.Offset.Y, float32(0))
	assert.Nil(t, w.Canvas().Focused())
}

func TestAccessibilityScrollEmptyCollectionAndResize(t *testing.T) {
	test.NewTempApp(t)
	count := 100
	list := widget.NewList(func() int { return count }, func() fyne.CanvasObject { return widget.NewLabel("Row") }, func(int, fyne.CanvasObject) {})
	w := test.NewWindow(list)
	defer w.Close()
	w.Resize(fyne.NewSize(200, 150))
	tree := test.NewAccessibilityTree(w.Canvas())
	n, ok := tree.Node(list)
	require.True(t, ok)
	require.True(t, tree.SetScrollPercent(n.ID, -1, 100))
	w.Resize(fyne.NewSize(200, 300))
	resized, _ := tree.Node(list)
	assert.Greater(t, resized.VerticalViewSize, n.VerticalViewSize)
	count = 0
	list.Refresh()
	empty, _ := tree.Node(list)
	assert.Equal(t, n.ID, empty.ID)
	assert.Equal(t, -1.0, empty.VerticalScrollPercent)
	assert.Equal(t, 100.0, empty.VerticalViewSize)
	assert.True(t, tree.SetScrollPercent(n.ID, -1, -1))
	assert.False(t, tree.SetScrollPercent(n.ID, -1, 0))
	count = 100
	list.Refresh()
	require.True(t, tree.SetScrollPercent(n.ID, -1, 50))
	refilled, _ := tree.Node(list)
	assert.InDelta(t, 50, refilled.VerticalScrollPercent, 0.001)
}

func TestAccessibilityCollectionScrollPreservesSelectionAndFocus(t *testing.T) {
	test.NewTempApp(t)
	created := 0
	list := widget.NewList(func() int { return 1000 }, func() fyne.CanvasObject {
		created++
		return widget.NewLabel("Template")
	}, func(int, fyne.CanvasObject) {})
	list.DescribeItem = func(i int) fyne.AccessibilityInfo { return fyne.AccessibilityInfo{Name: fmt.Sprint(i)} }
	list.SetItemHeight(2, 80)
	treeWidget := widget.NewTreeWithStrings(map[string][]string{"": {"root"}, "root": {"a", "b", "c"}})
	children := make([]string, 1000)
	for i := range children {
		children[i] = fmt.Sprintf("item-%d", i)
	}
	treeWidget.ChildUIDs = func(id string) []string {
		if id == "" {
			return children
		}
		return nil
	}
	treeWidget.IsBranch = func(id string) bool { return id == "" }
	treeWidget.DescribeNode = func(id string) fyne.AccessibilityInfo { return fyne.AccessibilityInfo{Name: id} }
	table := widget.NewTableWithHeaders(func() (int, int) { return 1000, 10 }, func() fyne.CanvasObject { return widget.NewLabel("Cell template") }, func(widget.TableCellID, fyne.CanvasObject) {})
	table.StickyRowCount, table.StickyColumnCount = 1, 1
	table.DescribeCell = func(id widget.TableCellID) fyne.AccessibilityInfo {
		return fyne.AccessibilityInfo{Name: fmt.Sprint(id)}
	}
	table.SetColumnWidth(3, 200)
	table.SetRowHeight(3, 70)
	for _, tc := range []struct {
		name        string
		object      fyne.CanvasObject
		focus       fyne.Focusable
		selectItem  func()
		active      func() string
		selectedKey string
	}{
		{"List", list, list, func() { list.Select(2) }, list.AccessibilityActiveElement, "2"},
		{"Tree", treeWidget, treeWidget, func() { treeWidget.Select("item-2") }, treeWidget.AccessibilityActiveElement, "item-2"},
		{"Table", table, table, func() { table.Select(widget.TableCellID{Row: 2, Col: 2}) }, table.AccessibilityActiveElement, table.AccessibilityCellKey(2, 2)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := test.NewWindow(tc.object)
			defer w.Close()
			w.Resize(fyne.NewSize(300, 200))
			w.Canvas().Focus(tc.focus)
			tc.selectItem()
			tree := test.NewAccessibilityTree(w.Canvas())
			n, ok := tree.Node(tc.object)
			require.True(t, ok)
			require.True(t, n.Scroll)
			active := tc.active()
			selected, ok := tree.Element(tc.object, tc.selectedKey)
			require.True(t, ok)
			require.True(t, selected.Selected)
			cells := created
			for i := 0; i < 3; i++ {
				tree.Snapshot()
			}
			assert.Equal(t, cells, created, "queries must not create visual list cells")
			require.True(t, tree.SetScrollPercent(n.ID, -1, 100))
			n, _ = tree.Node(tc.object)
			assert.InDelta(t, 100, n.VerticalScrollPercent, 0.001)
			assert.Less(t, n.VerticalViewSize, 1.0)
			if n.HorizontalScrollPercent >= 0 {
				require.True(t, tree.SetScrollPercent(n.ID, 100, -1))
				n, _ = tree.Node(tc.object)
				assert.InDelta(t, 100, n.HorizontalScrollPercent, 0.001)
			}
			assert.Same(t, tc.focus, w.Canvas().Focused())
			assert.Equal(t, active, tc.active())
			current, ok := tree.Element(tc.object, tc.selectedKey)
			require.True(t, ok)
			assert.Equal(t, selected.ID, current.ID)
			assert.True(t, current.Selected)
			focused, ok := tree.Element(tc.object, active)
			require.True(t, ok)
			assert.True(t, focused.Focused)
			assert.Zero(t, current.BoundsSize.Height)
			assert.Less(t, len(tree.Snapshot()), 100, "scrolling must preserve bounded collection publication")
		})
	}
}
