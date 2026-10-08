package test_test

import (
	"errors"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccessibilityEditableTableValueAndScope(t *testing.T) {
	test.NewTempApp(t)
	value := "original"
	table := widget.NewTable(func() (int, int) { return 1000, 2 }, func() fyne.CanvasObject { return widget.NewLabel("Template") }, func(widget.TableCellID, fyne.CanvasObject) {})
	table.CellValue = func(id widget.TableCellID) (string, bool) { return value, id.Col == 0 }
	table.OnCellChanged = func(_ widget.TableCellID, v string) error {
		if v == "" {
			return errors.New("required")
		}
		value = v
		return nil
	}
	w := test.NewWindow(table)
	defer w.Close()
	w.Resize(fyne.NewSize(300, 180))
	tree := test.NewAccessibilityTree(w.Canvas())
	key := table.AccessibilityCellKey(999, 1)
	cell, ok := tree.Element(table, key)
	require.True(t, ok)
	assert.True(t, cell.Value && cell.Invoke && cell.GridItem)
	assert.False(t, cell.ReadOnly)
	require.True(t, tree.Perform(cell.ID, test.AccessibilitySetValue, "changed", 0))
	assert.Equal(t, "changed", value)
	assert.False(t, tree.Perform(cell.ID, test.AccessibilitySetValue, "", 0))
	assert.Equal(t, "changed", value)
	readOnly, ok := tree.Element(table, table.AccessibilityCellKey(0, 0))
	require.True(t, ok)
	assert.True(t, readOnly.ReadOnly)
	assert.False(t, readOnly.Invoke)
	assert.False(t, tree.Perform(readOnly.ID, test.AccessibilitySetValue, "bad", 0))
	require.True(t, tree.Perform(cell.ID, test.AccessibilityActivate, "", 0))
	var editor test.AccessibilityNode
	for _, n := range tree.Snapshot() {
		if n.Role == fyne.AccessibleRoleEntry {
			editor = n
		}
	}
	require.NotNil(t, editor.Document)
	assert.True(t, editor.Focused)
	assert.False(t, tree.Perform(cell.ID, test.AccessibilitySetValue, "background", 0))
	table.CancelCellEdit()
	current, ok := tree.Element(table, key)
	require.True(t, ok)
	assert.Equal(t, cell.ID, current.ID)
	assert.False(t, tree.Perform(editor.ID, test.AccessibilitySetValue, "stale editor", 0))
	assert.Equal(t, "changed", value)
}
