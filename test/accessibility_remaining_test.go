package test_test

import (
	"fmt"
	"image/color"
	"math"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func remainingNode(t *testing.T, tree *test.AccessibilityTree, name string) test.AccessibilityNode {
	t.Helper()
	for _, node := range tree.Snapshot() {
		if node.Name == name {
			return node
		}
	}
	t.Fatalf("missing %q", name)
	return test.AccessibilityNode{}
}

func TestAccessibilityAccordionKeyboardOrder(t *testing.T) {
	test.NewTempApp(t)
	checks := widget.NewCheckGroup([]string{"Email", "Desktop"}, nil)
	second := widget.NewEntry()
	a := widget.NewAccordion(widget.NewAccordionItem("Notification settings", checks), widget.NewAccordionItem("More details", second))
	before, after := widget.NewButton("Before", nil), widget.NewButton("After", nil)
	w := test.NewWindow(container.NewVBox(before, a, after))
	defer w.Close()
	headers := a.AccessibilityChildren()
	firstHeader, secondHeader := headers[0].(fyne.Focusable), headers[1].(fyne.Focusable)
	a.Open(0)
	children := checks.AccessibilityChildren()
	email, desktop := children[0].(*widget.Check), children[1].(*widget.Check)
	checkOrder := func(order ...fyne.Focusable) {
		t.Helper()
		w.Canvas().Unfocus()
		for _, expected := range order {
			w.Canvas().FocusNext()
			require.Same(t, expected, w.Canvas().Focused())
		}
		w.Canvas().FocusNext()
		require.Same(t, order[0], w.Canvas().Focused(), "forward wrap")
		for i := len(order) - 1; i >= 0; i-- {
			w.Canvas().FocusPrevious()
			require.Same(t, order[i], w.Canvas().Focused(), "reverse traversal must be the inverse")
		}
	}
	checkOrder(before, firstHeader, email, desktop, secondHeader, after)
	email.Disable()
	checkOrder(before, firstHeader, desktop, secondHeader, after)
	email.Enable()
	a.CloseAll()
	checkOrder(before, firstHeader, secondHeader, after)
	a.Open(1)
	checkOrder(before, firstHeader, secondHeader, second, after)
	a.MultiOpen = true
	a.Open(0)
	checkOrder(before, firstHeader, email, desktop, secondHeader, second, after)
	a.Items[0], a.Items[1] = a.Items[1], a.Items[0]
	a.Refresh()
	checkOrder(before, secondHeader, second, firstHeader, email, desktop, after)
}

func TestAccessibilityLiveLabel(t *testing.T) {
	test.NewTempApp(t)
	status := widget.NewLabel("Ready")
	save := widget.NewButton("Save", func() { status.SetText("Saved") })
	w := test.NewWindow(container.NewVBox(save, status))
	defer w.Close()
	w.Canvas().Focus(save)
	tree := test.NewAccessibilityTree(w.Canvas())
	n, ok := tree.Node(status)
	require.True(t, ok)
	assert.Equal(t, fyne.AccessibilityLiveOff, n.LiveSetting)
	status.SetAccessibilityLiveSetting(fyne.AccessibilityLivePolite)
	n, _ = tree.Node(status)
	assert.Equal(t, fyne.AccessibilityLivePolite, n.LiveSetting)
	assert.False(t, n.Focusable)
	saveNode, ok := tree.Node(save)
	require.True(t, ok)
	for range 2 {
		previous := n
		require.True(t, tree.Perform(saveNode.ID, test.AccessibilityActivate, "", 0))
		n, _ = tree.Node(status)
		assert.Equal(t, "Saved", n.Name)
		assert.Greater(t, n.LiveRevision, previous.LiveRevision, "equal action results can be announced again")
		assert.Equal(t, previous.ID, n.ID)
		assert.Same(t, save, w.Canvas().Focused())
	}
	status.Refresh()
	unchanged, _ := tree.Node(status)
	assert.Equal(t, n.LiveRevision, unchanged.LiveRevision, "repainting is not an announcement")
	status.Hide()
	_, ok = tree.Node(status)
	assert.False(t, ok)
	status.Show()
	status.SetAccessibilityLiveSetting(fyne.AccessibilityLiveAssertive)
	n, _ = tree.Node(status)
	assert.Equal(t, fyne.AccessibilityLiveAssertive, n.LiveSetting)
	status.SetAccessibilityLiveSetting(fyne.AccessibilityLiveOff)
	status.SetText("Silent")
	n, _ = tree.Node(status)
	assert.Equal(t, fyne.AccessibilityLiveOff, n.LiveSetting)
	assert.Zero(t, n.LiveRevision)
}

func TestAccessibilityRemainingStatusAndGroups(t *testing.T) {
	test.NewTempApp(t)
	progress := widget.NewProgressBar()
	progress.SetValue(0.25)
	activity := widget.NewActivity()
	activity.Start()
	defer activity.Stop()
	infinite := widget.NewProgressBarInfinite()
	defer infinite.Stop()
	action := widget.NewToolbarAction(theme.DocumentSaveIcon(), nil)
	action.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Save document"})
	toolbar := widget.NewToolbar(action)
	icon := widget.NewIcon(theme.InfoIcon())
	icon.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Information"})
	decorative := widget.NewIcon(theme.InfoIcon())
	image := canvas.NewImageFromResource(theme.FyneLogo())
	image.AltText = "Logo"
	text := canvas.NewText("Canvas caption", color.Black)
	card := widget.NewCard("Summary", "Details", widget.NewButton("Action", nil))
	w := test.NewWindow(container.NewVBox(progress, activity, infinite, toolbar, icon, decorative, image, text, card))
	defer w.Close()
	tree := test.NewAccessibilityTree(w.Canvas())
	p, ok := tree.Node(progress)
	require.True(t, ok)
	assert.Equal(t, fyne.AccessibleRoleProgressBar, p.Role)
	assert.True(t, p.ReadOnly && p.Range && p.Value)
	assert.Equal(t, 0.25, p.Number)
	assert.Equal(t, "25%", p.Text)
	assert.True(t, math.IsNaN(p.Step))
	assert.False(t, tree.Perform(p.ID, test.AccessibilitySetRangeValue, "", 0.5))
	assert.Equal(t, 0.25, progress.Value)
	for _, object := range []fyne.CanvasObject{activity, infinite} {
		n, found := tree.Node(object)
		require.True(t, found)
		assert.False(t, n.Range)
		assert.True(t, n.ReadOnly)
	}
	_, ok = tree.Node(decorative)
	assert.False(t, ok)
	assert.Equal(t, fyne.AccessibleRoleImage, remainingNode(t, tree, "Logo").Role)
	assert.Equal(t, fyne.AccessibleRoleText, remainingNode(t, tree, "Canvas caption").Role)
	assert.True(t, remainingNode(t, tree, "Save document").Invoke)
	group := remainingNode(t, tree, "Summary. Details")
	assert.Equal(t, group.ID, remainingNode(t, tree, "Action").Parent)
}

func TestAccessibilityRemainingAccordionAndSplit(t *testing.T) {
	test.NewTempApp(t)
	accordion := widget.NewAccordion(widget.NewAccordionItem("Section", widget.NewButton("Inside", nil)))
	split := container.NewHSplit(accordion, widget.NewLabel("Other"))
	w := test.NewWindow(split)
	defer w.Close()
	w.Resize(fyne.NewSize(600, 300))
	tree := test.NewAccessibilityTree(w.Canvas())
	header := remainingNode(t, tree, "Section")
	assert.True(t, header.Expandable)
	assert.False(t, header.Expanded)
	require.True(t, tree.Perform(header.ID, test.AccessibilityExpand, "", 0))
	assert.True(t, remainingNode(t, tree, "Inside").Invoke)
	require.True(t, tree.Perform(header.ID, test.AccessibilityCollapse, "", 0))
	divider := remainingNode(t, tree, "Resize panes")
	require.True(t, divider.Range && divider.Focusable)
	require.True(t, tree.Perform(divider.ID, test.AccessibilitySetRangeValue, "", 0.6))
	assert.Equal(t, 0.6, split.Offset)
	assert.False(t, tree.Perform(divider.ID, test.AccessibilitySetRangeValue, "", math.NaN()))
	require.True(t, tree.Perform(divider.ID, test.AccessibilityFocus, "", 0))
	w.Canvas().Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyLeft})
	assert.Less(t, split.Offset, 0.6)
}

func TestAccessibilityRemainingGridWrapIdentity(t *testing.T) {
	test.NewTempApp(t)
	keys := make([]string, 1000)
	for i := range keys {
		keys[i] = fmt.Sprint(i)
	}
	created := 0
	grid := widget.NewGridWrap(func() int { return len(keys) }, func() fyne.CanvasObject { created++; return widget.NewLabel("Template") }, func(int, fyne.CanvasObject) {})
	grid.ItemKey = func(i int) string { return keys[i] }
	grid.DescribeItem = func(i int) fyne.AccessibilityInfo { return fyne.AccessibilityInfo{Name: "Record " + keys[i]} }
	w := test.NewWindow(grid)
	defer w.Close()
	w.Resize(fyne.NewSize(260, 180))
	tree := test.NewAccessibilityTree(w.Canvas())
	before := created
	last, ok := tree.Element(grid, "999")
	require.True(t, ok)
	assert.Equal(t, before, created, "semantic discovery must not render distant cells")
	assert.Less(t, len(tree.Snapshot()), 100)
	require.True(t, tree.Perform(last.ID, test.AccessibilitySelect, "", 0))
	keys[0], keys[999] = keys[999], keys[0]
	grid.Refresh()
	moved, ok := tree.Element(grid, "999")
	require.True(t, ok)
	assert.Equal(t, last.ID, moved.ID)
	assert.True(t, moved.Selected)
	assert.Equal(t, 1, moved.SetPosition)
	keys = keys[1:]
	grid.Refresh()
	keys = append(keys, "999")
	grid.Refresh()
	newNode, ok := tree.Element(grid, "999")
	require.True(t, ok)
	assert.NotEqual(t, last.ID, newNode.ID)
	assert.False(t, tree.Perform(last.ID, test.AccessibilitySelect, "", 0))
}

func TestAccessibilityRemainingCollectionControls(t *testing.T) {
	test.NewTempApp(t)
	keys := []string{"a", "b"}
	values := map[string]bool{"a": false, "b": false}
	list := widget.NewList(func() int { return len(keys) }, func() fyne.CanvasObject { return widget.NewLabel("Row") }, func(int, fyne.CanvasObject) {})
	list.ItemKey = func(i int) string { return keys[i] }
	list.ItemElements = func(i int) []fyne.AccessibilityElement {
		key := keys[i]
		check := widget.NewCheck("Enabled "+key, func(v bool) { values[key] = v })
		check.Checked = values[key]
		return []fyne.AccessibilityElement{{Key: "enabled", Object: check}}
	}
	w := test.NewWindow(list)
	defer w.Close()
	tree := test.NewAccessibilityTree(w.Canvas())
	child, ok := tree.Element(list, widget.CollectionControlKey("a", "enabled"))
	require.True(t, ok)
	assert.True(t, child.Toggle)
	assert.False(t, child.Focusable, "temporary model objects are not keyboard targets")
	assert.False(t, tree.Perform(child.ID, test.AccessibilityFocus, "", 0))
	require.True(t, tree.Perform(child.ID, test.AccessibilityToggle, "", 0))
	assert.True(t, values["a"])
	keys[0], keys[1] = keys[1], keys[0]
	list.Refresh()
	require.True(t, tree.Perform(child.ID, test.AccessibilityToggle, "", 0))
	assert.False(t, values["a"])
	assert.False(t, values["b"])
	keys = keys[:1]
	list.Refresh()
	keys = append(keys, "a")
	list.Refresh()
	assert.False(t, tree.Perform(child.ID, test.AccessibilityToggle, "", 0))
	newChild, ok := tree.Element(list, widget.CollectionControlKey("a", "enabled"))
	require.True(t, ok)
	assert.NotEqual(t, child.ID, newChild.ID)
}

func TestAccessibilityRemainingCalendarAndTextGrid(t *testing.T) {
	test.NewTempApp(t)
	var chosen time.Time
	calendar := widget.NewCalendar(time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC), func(d time.Time) { chosen = d })
	grid := widget.NewTextGridFromString("A\tB\nРусский 😀\n")
	w := test.NewWindow(container.NewGridWithColumns(2, calendar, grid))
	defer w.Close()
	w.Resize(fyne.NewSize(650, 300))
	tree := test.NewAccessibilityTree(w.Canvas())
	day, ok := tree.Element(calendar, "2026-10-05")
	require.True(t, ok)
	assert.True(t, day.GridItem && day.Selectable && day.Invoke)
	require.True(t, tree.Perform(day.ID, test.AccessibilitySelect, "", 0))
	assert.Equal(t, 5, chosen.Day())
	require.True(t, tree.Perform(day.ID, test.AccessibilityFocus, "", 0))
	day, ok = tree.Element(calendar, "2026-10-05")
	require.True(t, ok)
	assert.True(t, day.Focused)
	w.Canvas().Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	next, ok := tree.Element(calendar, "2026-10-06")
	require.True(t, ok)
	assert.True(t, next.Focused)
	nextMonth := remainingNode(t, tree, "Next month")
	require.True(t, tree.Perform(nextMonth.ID, test.AccessibilityActivate, "", 0))
	selected, ok := tree.Element(calendar, "2026-10-05")
	require.True(t, ok)
	assert.True(t, selected.Selected && selected.Size.IsZero())
	require.True(t, tree.Perform(selected.ID, test.AccessibilityFocus, "", 0))
	selected, ok = tree.Element(calendar, "2026-10-05")
	require.True(t, ok)
	assert.True(t, selected.Focused && !selected.Size.IsZero())
	n, ok := tree.Node(grid)
	require.True(t, ok)
	require.NotNil(t, n.Document)
	assert.Equal(t, grid.Text(), n.Document.Text)
	assert.Len(t, n.Document.Positions, len([]rune(n.Document.Text))+1)
	assert.True(t, n.Document.ReadOnly && n.Document.SelectionDisabled)
	assert.False(t, tree.SelectText(n.ID, 0, 1))
}

func TestAccessibilityRemainingSelectEntry(t *testing.T) {
	test.NewTempApp(t)
	entry := widget.NewSelectEntry([]string{"One", "Two"})
	entry.SetText("Free text")
	w := test.NewWindow(entry)
	defer w.Close()
	tree := test.NewAccessibilityTree(w.Canvas())
	n, ok := tree.Node(entry)
	require.True(t, ok)
	assert.True(t, n.Selection && n.Expandable && n.Value)
	assert.NotNil(t, n.Document)
	choice := remainingNode(t, tree, "Two")
	assert.Equal(t, n.ID, choice.Parent)
	assert.False(t, choice.Focusable, "collapsed model options cannot receive keyboard focus")
	assert.Equal(t, fyne.Size{}, choice.BoundsSize)
	button := remainingNode(t, tree, "Show choices")
	assert.Equal(t, n.ID, button.Parent)
	require.True(t, tree.Perform(choice.ID, test.AccessibilitySelect, "", 0))
	assert.Equal(t, "Two", entry.Text)
	require.True(t, tree.Perform(n.ID, test.AccessibilityExpand, "", 0))
	assert.True(t, entry.AccessibilityExpanded())
	w.Canvas().Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	assert.False(t, entry.AccessibilityExpanded())
	collapsed := remainingNode(t, tree, "Two")
	assert.Equal(t, choice.ID, collapsed.ID, "collapse preserves the model option")
	assert.False(t, collapsed.Focusable)
	assert.Equal(t, fyne.Size{}, collapsed.BoundsSize)
	entry.SetOptions([]string{"One", "Three"})
	assert.False(t, tree.Perform(choice.ID, test.AccessibilitySelect, "", 0))
}

func TestAccessibilityDatePopupKeyboard(t *testing.T) {
	for _, key := range []fyne.KeyName{fyne.KeyReturn, fyne.KeyEnter, fyne.KeySpace} {
		t.Run(string(key), func(t *testing.T) {
			test.NewTempApp(t)
			date := widget.NewDateEntry()
			original := time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC)
			date.SetDate(&original)
			w := test.NewWindow(date)
			defer w.Close()
			tree := test.NewAccessibilityTree(w.Canvas())
			node, ok := tree.Node(date)
			require.True(t, ok)
			button := remainingNode(t, tree, "Choose date")
			assert.Equal(t, node.ID, button.Parent)
			require.True(t, tree.Perform(node.ID, test.AccessibilityExpand, "", 0))
			canvas := w.Canvas()
			day := canvas.Focused()
			canvas.FocusNext()
			previous := canvas.Focused()
			require.NotSame(t, day, previous)
			canvas.FocusNext()
			canvas.FocusNext()
			require.Same(t, day, canvas.Focused(), "popup has two month buttons and one date Tab stop")
			canvas.FocusPrevious()
			canvas.FocusPrevious()
			require.Same(t, previous, canvas.Focused())
			canvas.FocusPrevious()
			require.Same(t, day, canvas.Focused())
			day.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
			canvas.Focused().TypedKey(&fyne.KeyEvent{Name: key})
			tree.Snapshot()
			require.False(t, date.AccessibilityExpanded())
			require.Same(t, date, canvas.Focused())
			require.NotNil(t, date.Date)
			assert.Equal(t, "2026-02-01", date.Date.Format("2006-01-02"))
			require.True(t, tree.Perform(node.ID, test.AccessibilityExpand, "", 0))
			canvas.Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
			canvas.Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
			tree.Snapshot()
			require.Same(t, date, canvas.Focused())
			assert.Equal(t, "2026-02-01", date.Date.Format("2006-01-02"))
		})
	}
}

func TestAccessibilityRemainingDateEntryAndInnerWindow(t *testing.T) {
	test.NewTempApp(t)
	date := widget.NewDateEntry()
	original := time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC)
	date.SetDate(&original)
	inner := container.NewInnerWindow("Editor", date)
	windows := container.NewMultipleWindows(inner)
	w := test.NewWindow(windows)
	defer w.Close()
	w.Resize(fyne.NewSize(700, 500))
	tree := test.NewAccessibilityTree(w.Canvas())
	node, ok := tree.Node(date)
	require.True(t, ok)
	assert.False(t, tree.Perform(node.ID, test.AccessibilitySetValue, "invalid date", 0))
	require.NotNil(t, date.Date)
	assert.Equal(t, 4, date.Date.Day())
	require.True(t, tree.Perform(node.ID, test.AccessibilityExpand, "", 0))
	assert.True(t, date.AccessibilityExpanded())
	w.Canvas().Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	tree.Snapshot()
	assert.False(t, date.AccessibilityExpanded())
	assert.Same(t, date, w.Canvas().Focused())
	move := remainingNode(t, tree, "Move Editor")
	before := inner.Position()
	require.True(t, tree.Perform(move.ID, test.AccessibilityFocus, "", 0))
	w.Canvas().Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	assert.Greater(t, inner.Position().X, before.X)
	resize := remainingNode(t, tree, "Resize window")
	beforeSize := inner.Size()
	require.True(t, tree.Perform(resize.ID, test.AccessibilityFocus, "", 0))
	w.Canvas().Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	assert.Greater(t, inner.Size().Width, beforeSize.Width)
	assert.False(t, tree.Perform(resize.ID, test.AccessibilitySetValue, "NaN,100", 0))
}
