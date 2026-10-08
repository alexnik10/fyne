package test_test

import (
	"fmt"
	"slices"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func semanticNamed(t *testing.T, tree *test.AccessibilityTree, role fyne.AccessibleRole, name string) test.AccessibilityNode {
	t.Helper()
	for _, n := range tree.Snapshot() {
		if n.Role == role && n.Name == name {
			return n
		}
	}
	t.Fatalf("missing semantic node %s %q", role, name)
	return test.AccessibilityNode{}
}

func TestAccessibilityTabsKeyboardLifetimeAndClose(t *testing.T) {
	test.NewTempApp(t)
	entry := widget.NewEntry()
	entry.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Editor"})
	a := container.NewTabItem("First", entry)
	b := container.NewTabItem("Disabled", widget.NewButton("Hidden button", nil))
	c := container.NewTabItem("Last", widget.NewLabel("Last page"))
	tabs := container.NewDocTabs(a, b, c)
	w := test.NewWindow(tabs)
	defer w.Close()
	w.Resize(fyne.NewSize(600, 300))
	tabs.DisableItem(b)
	tree := test.NewAccessibilityTree(w.Canvas())
	first := semanticNamed(t, tree, fyne.AccessibleRoleTabItem, "First")
	require.True(t, first.Selected)
	require.True(t, first.Selectable)
	assert.False(t, first.Invoke)
	for _, n := range tree.Snapshot() {
		assert.NotEqual(t, "Hidden button", n.Name)
	}
	require.True(t, tree.Perform(first.ID, test.AccessibilityFocus, "", 0))
	assert.Same(t, tabs, w.Canvas().Focused())
	tabs.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	last := semanticNamed(t, tree, fyne.AccessibleRoleTabItem, "Last")
	assert.True(t, last.Focused)
	assert.True(t, last.Selected)
	tabs.SetItems([]*container.TabItem{c, b, a})
	last2 := semanticNamed(t, tree, fyne.AccessibleRoleTabItem, "Last")
	assert.Equal(t, last.ID, last2.ID)
	assert.True(t, last2.Selected)
	tabs.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnd})
	assert.Same(t, a, tabs.Selected())
	w.Canvas().Focus(entry)
	closeCommand := semanticNamed(t, tree, fyne.AccessibleRoleButton, "Close First")
	require.True(t, tree.Perform(closeCommand.ID, test.AccessibilityActivate, "", 0))
	assert.Same(t, tabs, w.Canvas().Focused())
	tabs.Append(a)
	first2 := semanticNamed(t, tree, fyne.AccessibleRoleTabItem, "First")
	assert.NotEqual(t, first.ID, first2.ID)
	assert.False(t, tree.Perform(first.ID, test.AccessibilitySelect, "", 0))
	assert.Empty(t, tree.Issues())
}

func TestAccessibilityAppTabsOverflowAndVerticalKeys(t *testing.T) {
	test.NewTempApp(t)
	var items []*container.TabItem
	for n := 0; n < 12; n++ {
		items = append(items, container.NewTabItem(fmt.Sprintf("Section %d", n), widget.NewLabel("Page")))
	}
	tabs := container.NewAppTabs(items...)
	w := test.NewWindow(tabs)
	defer w.Close()
	w.Resize(fyne.NewSize(200, 150))
	tree := test.NewAccessibilityTree(w.Canvas())
	last := semanticNamed(t, tree, fyne.AccessibleRoleTabItem, "Section 11")
	require.True(t, tree.Perform(last.ID, test.AccessibilityFocus, "", 0))
	assert.Same(t, items[11], tabs.Selected())
	tabs.SetTabLocation(container.TabLocationLeading)
	forward, backward := fyne.KeyDown, fyne.KeyUp
	if fyne.CurrentDevice().IsMobile() && fyne.IsVertical(fyne.CurrentDevice().Orientation()) {
		// Portrait mobile layouts move leading tabs to the top edge.
		forward, backward = fyne.KeyRight, fyne.KeyLeft
	}
	tabs.TypedKey(&fyne.KeyEvent{Name: forward})
	assert.Same(t, items[0], tabs.Selected())
	tabs.TypedKey(&fyne.KeyEvent{Name: backward})
	assert.Same(t, items[11], tabs.Selected())
}

func TestAccessibilityMenuStateSubmenuScopeAndFocus(t *testing.T) {
	test.NewTempApp(t)
	button := widget.NewButton("Open", nil)
	w := test.NewWindow(button)
	defer w.Close()
	w.Resize(fyne.NewSize(600, 400))
	w.Canvas().Focus(button)
	invoked := 0
	disabled := fyne.NewMenuItem("Unavailable", func() { t.Error("disabled command invoked") })
	disabled.Disabled = true
	check := fyne.NewMenuItem("Checked option", nil)
	check.Checkable = true
	check.Action = func() { check.Checked = !check.Checked; invoked++ }
	check.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyK, Modifier: fyne.KeyModifierControl}
	branch := fyne.NewMenuItem("More", nil)
	branch.ChildMenu = fyne.NewMenu("More commands", fyne.NewMenuItem("Nested", func() { invoked++ }))
	popup := widget.NewPopUpMenu(fyne.NewMenu("Actions", disabled, check, branch), w.Canvas())
	popup.ShowAtPosition(fyne.NewPos(30, 30))
	popup.AccessibilityFocus()
	tree := test.NewAccessibilityTree(w.Canvas())
	option := semanticNamed(t, tree, fyne.AccessibleRoleMenuItem, "Checked option")
	assert.True(t, option.Focused)
	assert.True(t, option.Toggle)
	assert.False(t, option.Checked)
	assert.Equal(t, "Ctrl+K", option.Shortcut)
	more := semanticNamed(t, tree, fyne.AccessibleRoleMenuItem, "More")
	assert.True(t, more.Expandable)
	assert.False(t, more.Invoke)
	assert.False(t, more.Toggle)
	require.True(t, tree.Perform(more.ID, test.AccessibilityExpand, "", 0))
	nested := semanticNamed(t, tree, fyne.AccessibleRoleMenuItem, "Nested")
	assert.True(t, nested.Focused)
	assert.Greater(t, nested.BoundsSize.Width, float32(0), "submenu escapes opener viewport")
	assert.False(t, tree.Perform(semanticNamed(t, tree, fyne.AccessibleRoleMenuItem, "Unavailable").ID, test.AccessibilityActivate, "", 0))
	popup.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	assert.True(t, popup.Visible())
	assert.True(t, semanticNamed(t, tree, fyne.AccessibleRoleMenuItem, "More").Focused)
	assert.False(t, tree.Perform(nested.ID, test.AccessibilityActivate, "", 0))
	require.True(t, tree.Perform(option.ID, test.AccessibilityToggle, "", 0))
	assert.Equal(t, 1, invoked)
	assert.True(t, check.Checked)
	assert.Same(t, button, w.Canvas().Focused())
}

func TestAccessibilityTableKeysHeadersAndVirtualCells(t *testing.T) {
	test.NewTempApp(t)
	rows := []string{"one", "two", "three"}
	cols := []string{"name", "status"}
	created, updated := 0, 0
	table := widget.NewTableWithHeaders(func() (int, int) { return len(rows), len(cols) }, func() fyne.CanvasObject { created++; return widget.NewLabel("Template") }, func(_ widget.TableCellID, _ fyne.CanvasObject) { updated++ })
	table.RowKey = func(row int) string { return rows[row] }
	table.ColumnKey = func(col int) string { return cols[col] }
	table.DescribeCell = func(id widget.TableCellID) fyne.AccessibilityInfo {
		if id.Row < 0 {
			return fyne.AccessibilityInfo{Name: cols[id.Col]}
		}
		if id.Col < 0 {
			return fyne.AccessibilityInfo{Name: rows[id.Row]}
		}
		return fyne.AccessibilityInfo{Name: rows[id.Row] + " " + cols[id.Col]}
	}
	w := test.NewWindow(table)
	defer w.Close()
	w.Resize(fyne.NewSize(350, 180))
	tree := test.NewAccessibilityTree(w.Canvas())
	key := table.AccessibilityCellKey(1, 1)
	beforeCreated, beforeUpdated := created, updated
	cell, ok := tree.Element(table, key)
	require.True(t, ok)
	assert.True(t, cell.GridItem)
	assert.True(t, cell.Table)
	assert.Equal(t, 1, cell.Row)
	assert.Equal(t, 1, cell.Column)
	assert.Equal(t, "two status", cell.Name)
	header, ok := tree.Element(table, table.AccessibilityCellKey(-1, 1))
	require.True(t, ok)
	assert.Equal(t, "status", header.Name)
	assert.False(t, header.GridItem)
	assert.Equal(t, beforeCreated, created)
	assert.Equal(t, beforeUpdated, updated)
	require.True(t, tree.Perform(cell.ID, test.AccessibilitySelect, "", 0))
	require.True(t, tree.Perform(cell.ID, test.AccessibilityFocus, "", 0))
	slices.Reverse(rows)
	slices.Reverse(cols)
	table.Refresh()
	cell2, ok := tree.Element(table, key)
	require.True(t, ok)
	assert.Equal(t, cell.ID, cell2.ID)
	assert.True(t, cell2.Selected)
	assert.True(t, cell2.Focused)
	assert.Equal(t, 0, cell2.Column)
	rows = []string{"three", "one"}
	table.Refresh()
	rows = append(rows, "two")
	table.Refresh()
	cell2, ok = tree.Element(table, key)
	require.True(t, ok)
	assert.NotEqual(t, cell.ID, cell2.ID)
	assert.False(t, tree.Perform(cell.ID, test.AccessibilitySelect, "", 0))
	assert.Empty(t, tree.Issues())
}

func TestAccessibilityLargeTableBoundedSemantics(t *testing.T) {
	test.NewTempApp(t)
	created, described := 0, 0
	table := widget.NewTable(func() (int, int) { return 100000, 100 }, func() fyne.CanvasObject { created++; return widget.NewLabel("Template") }, func(widget.TableCellID, fyne.CanvasObject) {})
	table.DescribeCell = func(id widget.TableCellID) fyne.AccessibilityInfo {
		described++
		return fyne.AccessibilityInfo{Name: fmt.Sprintf("%d,%d", id.Row, id.Col)}
	}
	w := test.NewWindow(table)
	defer w.Close()
	w.Resize(fyne.NewSize(300, 150))
	before := created
	tree := test.NewAccessibilityTree(w.Canvas())
	nodes := tree.Snapshot()
	assert.Less(t, len(nodes), 100)
	assert.Less(t, described, 100)
	last, ok := tree.Element(table, table.AccessibilityCellKey(99999, 99))
	require.True(t, ok)
	assert.Equal(t, "99999,99", last.Name)
	assert.Zero(t, last.BoundsSize.Width)
	assert.Equal(t, before, created)
	require.True(t, tree.Perform(last.ID, test.AccessibilityScrollIntoView, "", 0))
	last2, ok := tree.Element(table, table.AccessibilityCellKey(99999, 99))
	require.True(t, ok)
	assert.Equal(t, last.ID, last2.ID)
	assert.Greater(t, last2.BoundsSize.Width, float32(0))
}
