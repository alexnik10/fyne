package test_test

import (
	"fmt"
	"slices"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccessibilityListKeysCommandsAndLifetime(t *testing.T) {
	test.NewTempApp(t)
	data := []string{"a", "b", "c"}
	list := widget.NewList(func() int { return len(data) },
		func() fyne.CanvasObject { return widget.NewLabel("Template") },
		func(id int, obj fyne.CanvasObject) { obj.(*widget.Label).SetText(data[id]) })
	list.ItemKey = func(id int) string { return data[id] }
	list.DescribeItem = func(id int) fyne.AccessibilityInfo { return fyne.AccessibilityInfo{Name: "Name " + data[id]} }
	w := test.NewWindow(list)
	defer w.Close()
	s := test.NewAccessibilityTree(w.Canvas())
	root, ok := s.Node(list)
	require.True(t, ok)
	assert.Equal(t, fyne.AccessibleRoleList, root.Role)
	assert.True(t, root.Selection)
	a, ok := s.Element(list, "a")
	require.True(t, ok)
	assert.Equal(t, "Name a", a.Name)
	assert.Equal(t, fyne.AccessibleRoleListItem, a.Role)
	assert.Equal(t, root.ID, a.SelectionOwner)
	assert.Equal(t, 1, a.SetPosition)
	assert.Equal(t, 3, a.SetSize)
	assert.False(t, a.Invoke)
	assert.False(t, a.Expandable)
	require.True(t, s.Perform(a.ID, test.AccessibilityFocus, "", 0))
	assert.Same(t, list, w.Canvas().Focused())
	a2, _ := s.Element(list, "a")
	assert.True(t, a2.Focused)
	assert.False(t, a2.Selected)
	w.Canvas().Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	b, _ := s.Element(list, "b")
	assert.True(t, b.Focused)
	selected, unselected := []int{}, []int{}
	list.OnSelected = func(id int) { selected = append(selected, id) }
	list.OnUnselected = func(id int) { unselected = append(unselected, id) }
	require.True(t, s.Perform(a.ID, test.AccessibilitySelect, "", 0))
	assert.False(t, s.Perform(b.ID, test.AccessibilityAddToSelection, "", 0))
	data = []string{"c", "a", "b"}
	list.Refresh()
	a2, _ = s.Element(list, "a")
	b2, _ := s.Element(list, "b")
	assert.Equal(t, a.ID, a2.ID)
	assert.Equal(t, b.ID, b2.ID)
	assert.Equal(t, 2, a2.SetPosition)
	assert.True(t, a2.Selected)
	assert.True(t, b2.Focused)
	assert.Equal(t, []int{0}, selected, "reordering is not a new selection")
	assert.Empty(t, unselected)
	w.Canvas().Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	assert.Equal(t, []int{0, 2}, selected, "keyboard uses the new model position")
	assert.Equal(t, []int{1}, unselected)
	data = []string{"c", "a"}
	list.Refresh()
	assert.Equal(t, []int{1, 2}, unselected)
	data = append(data, "b")
	list.Refresh() // No intervening semantic snapshot: the old provider must still die.
	b2, _ = s.Element(list, "b")
	assert.NotEqual(t, b.ID, b2.ID)
	assert.False(t, s.Perform(b.ID, test.AccessibilitySelect, "", 0))
	assert.False(t, b2.Selected)
	assert.False(t, b2.Focused)
	require.True(t, s.Perform(b2.ID, test.AccessibilitySelect, "", 0))
	require.True(t, s.Perform(b2.ID, test.AccessibilityRemoveFromSelection, "", 0))
	list.Hide()
	assert.False(t, s.Perform(b2.ID, test.AccessibilityFocus, "", 0))
	list.Show()
	b3, _ := s.Element(list, "b")
	assert.Equal(t, b2.ID, b3.ID)
	require.Empty(t, s.Issues())
}

func TestAccessibilityListOffscreenAndCustomHeight(t *testing.T) {
	test.NewTempApp(t)
	const count = 1000
	data := make([]string, count)
	for index := range data {
		data[index] = fmt.Sprintf("record-%d", index)
	}
	created, updated, described := 0, 0, []int{}
	list := widget.NewList(func() int { return len(data) },
		func() fyne.CanvasObject { created++; return widget.NewLabel("Template") },
		func(id int, obj fyne.CanvasObject) { updated++; obj.(*widget.Label).SetText(data[id]) })
	list.ItemKey = func(id int) string { return data[id] }
	list.DescribeItem = func(id int) fyne.AccessibilityInfo {
		described = append(described, id)
		return fyne.AccessibilityInfo{Name: data[id]}
	}
	w := test.NewWindow(list)
	defer w.Close()
	w.Resize(fyne.NewSize(220, 150))
	list.SetItemHeight(count-1, 80)
	beforeCreated, beforeUpdated := created, updated
	source := list.AccessibilityCollection()
	assert.Equal(t, count, source.ChildCount(""))
	assert.Zero(t, source.ChildCount(data[0]))
	assert.Equal(t, data[count-1], source.ChildKey("", count-1))
	assert.Empty(t, source.ChildKey("", -1))
	assert.Empty(t, source.ChildKey("", count))
	assert.Empty(t, source.ChildKey(data[0], 0))
	_, ok := source.Element("missing")
	assert.False(t, ok)
	element, ok := source.Element(data[count-1])
	require.True(t, ok)
	assert.Empty(t, described, "topology does not request row metadata")
	element.Object.(fyne.AccessibleDescribed).AccessibilityInfo()
	assert.Equal(t, []int{count - 1}, described, "a source can describe a single row")
	s := test.NewAccessibilityTree(w.Canvas())
	last, ok := s.Element(list, data[count-1])
	require.True(t, ok)
	assert.Zero(t, last.BoundsSize.Height)
	assert.Equal(t, float32(80), last.Size.Height)
	assert.Equal(t, beforeCreated, created)
	assert.Equal(t, beforeUpdated, updated)
	assert.Less(t, created, 100)
	require.True(t, s.Perform(last.ID, test.AccessibilityScrollIntoView, "", 0))
	last2, _ := s.Element(list, data[count-1])
	assert.Equal(t, last.ID, last2.ID)
	assert.Equal(t, float32(80), last2.BoundsSize.Height, "reveal the entire custom-height row")
	assert.Nil(t, w.Canvas().Focused())
	assert.False(t, last2.Selected)
	slices.Reverse(data)
	list.Refresh()
	last2, _ = s.Element(list, data[0])
	assert.Equal(t, last.ID, last2.ID)
	assert.Equal(t, float32(80), last2.Size.Height, "custom height follows the model key")
	assert.True(t, s.Perform(last.ID, test.AccessibilityFocus, "", 0))
	last2, _ = s.Element(list, data[0])
	assert.True(t, last2.Focused)
}

func TestAccessibilityListPositionalEmptyAndInvalidKeys(t *testing.T) {
	test.NewTempApp(t)
	data := []string{"first", "second"}
	list := widget.NewList(func() int { return len(data) },
		func() fyne.CanvasObject { return widget.NewLabel("Template") },
		func(id int, obj fyne.CanvasObject) { obj.(*widget.Label).SetText(data[id]) })
	w := test.NewWindow(list)
	defer w.Close()
	s := test.NewAccessibilityTree(w.Canvas())
	first, ok := s.Element(list, "0")
	require.True(t, ok)
	assert.Equal(t, "Item 1", first.Name)
	list.Select(1)
	list.Highlight(len(data)) // Clamp to the last valid position.
	w.Canvas().Focus(list)
	second, _ := s.Element(list, "1")
	assert.True(t, second.Focused)
	slices.Reverse(data)
	list.Refresh()
	first2, _ := s.Element(list, "0")
	assert.Equal(t, first.ID, first2.ID, "without keys identity belongs to positions")
	data = nil
	list.Refresh()
	assert.False(t, s.Perform(second.ID, test.AccessibilitySelect, "", 0))
	list.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	list.Highlight(0)
	root, _ := s.Node(list)
	assert.True(t, root.Focused, "empty lists report owner focus")
	assert.Empty(t, s.Issues())
	data = []string{"duplicate", "duplicate"}
	list.ItemKey = func(id int) string { return data[id] }
	list.Refresh()
	_, ok = s.Element(list, "duplicate")
	assert.False(t, ok, "ambiguous keys must not target an arbitrary row")
	assert.Contains(t, s.Issues()[0].Code, "invalid-element")
}

func TestAccessibilityTreeSourceAndUnobservedRemoval(t *testing.T) {
	test.NewTempApp(t)
	data := map[string][]string{"": {"folder"}, "folder": {"child"}}
	tree := widget.NewTreeWithStrings(data)
	described := []string{}
	tree.DescribeNode = func(id string) fyne.AccessibilityInfo {
		described = append(described, id)
		return fyne.AccessibilityInfo{Name: id}
	}
	w := test.NewWindow(tree)
	defer w.Close()
	source := tree.AccessibilityCollection()
	assert.Equal(t, 1, source.ChildCount("folder"))
	assert.Equal(t, "child", source.ChildKey("folder", 0))
	hidden, ok := source.Element("child")
	require.True(t, ok)
	assert.True(t, hidden.Hidden)
	assert.Nil(t, hidden.Object)
	assert.Empty(t, described)
	tree.OpenBranch("folder")
	s := test.NewAccessibilityTree(w.Canvas())
	child, _ := s.Element(tree, "child")
	tree.CloseBranch("folder")
	data["folder"] = nil
	tree.Refresh()
	data["folder"] = []string{"child"}
	tree.Refresh()
	tree.OpenBranch("folder")
	child2, _ := s.Element(tree, "child")
	assert.NotEqual(t, child.ID, child2.ID)
	assert.False(t, s.Perform(child.ID, test.AccessibilityFocus, "", 0))
	// Reparenting is a move, not a new lifetime.
	data[""] = []string{"child", "folder"}
	data["folder"] = nil
	tree.Refresh()
	child3, _ := s.Element(tree, "child")
	assert.Equal(t, child2.ID, child3.ID)
	assert.Equal(t, 1, child3.Level)
	assert.Empty(t, s.Issues())
}

func TestAccessibilityListPositionalRetirementAndFirstHighlight(t *testing.T) {
	test.NewTempApp(t)
	data := []string{"a", "b"}
	list := widget.NewList(func() int { return len(data) },
		func() fyne.CanvasObject { return widget.NewLabel("Template") },
		func(id int, obj fyne.CanvasObject) { obj.(*widget.Label).SetText(data[id]) })
	w := test.NewWindow(list)
	defer w.Close()
	s := test.NewAccessibilityTree(w.Canvas())
	first, _ := s.Element(list, "0")
	second, _ := s.Element(list, "1")
	data = data[:1]
	list.Refresh()
	data = append(data, "c")
	list.Refresh()
	first2, _ := s.Element(list, "0")
	second2, _ := s.Element(list, "1")
	assert.Equal(t, first.ID, first2.ID)
	assert.NotEqual(t, second.ID, second2.ID)
	assert.False(t, s.Perform(second.ID, test.AccessibilitySelect, "", 0))
	list.ItemKey = func(id int) string { return data[id] }
	list.Refresh()
	w.Canvas().Focus(list)
	highlighted := []int{}
	list.OnHighlighted = func(id int) { highlighted = append(highlighted, id) }
	data = data[1:]
	list.Refresh()
	assert.Equal(t, []int{0}, highlighted, "a replacement at the same position is a new highlight")
	c, _ := s.Element(list, "c")
	assert.True(t, c.Focused)
	assert.False(t, c.Selected)
}

func TestAccessibilityListFocusSurvivesInvokedReorder(t *testing.T) {
	test.NewTempApp(t)
	items := make([]string, 120)
	for i := range items {
		items[i] = fmt.Sprintf("item-%03d", i)
	}
	list := widget.NewList(func() int { return len(items) },
		func() fyne.CanvasObject { return widget.NewLabel("Template") },
		func(id int, obj fyne.CanvasObject) { obj.(*widget.Label).SetText(items[id]) })
	list.ItemKey = func(id int) string { return items[id] }
	list.DescribeItem = func(id int) fyne.AccessibilityInfo { return fyne.AccessibilityInfo{Name: items[id]} }
	list.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Native list"})
	reverse := widget.NewButton("Reverse list", func() { slices.Reverse(items); list.Refresh() })
	w := test.NewWindow(container.NewBorder(nil, reverse, nil, nil, list))
	defer w.Close()
	w.Resize(fyne.NewSize(350, 280))
	s := test.NewAccessibilityTree(w.Canvas())
	last, _ := s.Element(list, "item-119")
	button, _ := s.Node(reverse)
	require.True(t, s.Perform(last.ID, test.AccessibilityScrollIntoView, "", 0))
	require.True(t, s.Perform(last.ID, test.AccessibilitySelect, "", 0))
	require.True(t, s.Perform(last.ID, test.AccessibilityFocus, "", 0))
	last2, _ := s.Element(list, "item-119")
	require.True(t, last2.Focused)
	w.Canvas().Focus(reverse) // Native UIA may focus the invoking button first.
	require.True(t, s.Perform(button.ID, test.AccessibilityActivate, "", 0))
	assert.Same(t, reverse, w.Canvas().Focused(), "model refresh must not steal focus")
	assert.Equal(t, "item-119", list.AccessibilityActiveElement())
	root, _ := s.Node(list)
	require.True(t, s.Perform(root.ID, test.AccessibilityFocus, "", 0))
	last2, _ = s.Element(list, "item-119")
	assert.True(t, last2.Focused, "focusing the owner restores the previous model item")
	assert.True(t, last2.Selected)
	assert.Equal(t, "item-119", list.AccessibilityActiveElement())
	assert.Same(t, list, w.Canvas().Focused())
}
