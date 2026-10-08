package widget

import (
	"errors"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTableCellEditing(t *testing.T) {
	rows := []string{"first", "second"}
	values := map[string]string{"first": "One", "second": "Two"}
	table := NewTable(func() (int, int) { return len(rows), 2 }, func() fyne.CanvasObject { return NewLabel("Template") }, func(TableCellID, fyne.CanvasObject) {})
	table.RowKey = func(row int) string { return rows[row] }
	table.CellValue = func(id TableCellID) (string, bool) { return values[rows[id.Row]], id.Col == 1 }
	writes := 0
	table.OnCellChanged = func(id TableCellID, value string) error {
		if value == "" {
			return errors.New("value is required")
		}
		writes++
		values[rows[id.Row]] = value
		return nil
	}
	w := test.NewWindow(table)
	defer w.Close()
	w.Resize(fyne.NewSize(300, 200))
	w.Canvas().Focus(table)
	require.False(t, table.EditCell(TableCellID{0, 1}))
	table.TypedKey(&fyne.KeyEvent{Name: fyne.KeyF2})
	require.NotNil(t, table.cellEdit)
	entry := table.cellEdit.entry
	assert.Same(t, entry, w.Canvas().Focused())
	assert.Equal(t, "One", entry.SelectedText())
	entry.SetText("")
	require.Error(t, table.CommitCellEdit())
	assert.True(t, entry.AccessibilityInfo().Invalid)
	assert.Equal(t, 0, writes)
	rows[0], rows[1] = rows[1], rows[0]
	table.Refresh()
	require.Same(t, entry, table.cellEdit.entry)
	entry.SetText("Changed")
	require.NoError(t, table.CommitCellEdit())
	assert.Equal(t, "Changed", values["first"])
	assert.Equal(t, "Two", values["second"])
	assert.Equal(t, 1, writes)
	assert.Nil(t, table.cellEdit)
	require.True(t, table.EditCell(TableCellID{0, 0}))
	table.cellEdit.entry.SetText("Cancelled")
	table.cellEdit.entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	assert.Equal(t, "Two", values["second"])
	require.True(t, table.EditCell(TableCellID{0, 0}))
	rows = rows[1:]
	table.Refresh()
	assert.Nil(t, table.cellEdit)
	assert.Equal(t, 1, writes)
}

func TestTableCellValueIdentity(t *testing.T) {
	rows := []string{"a", "b"}
	values := map[string]string{"a": "A", "b": "B"}
	table := NewTable(func() (int, int) { return len(rows), 1 }, func() fyne.CanvasObject { return NewLabel("") }, func(TableCellID, fyne.CanvasObject) {})
	table.RowKey = func(row int) string { return rows[row] }
	table.CellValue = func(id TableCellID) (string, bool) { return values[rows[id.Row]], false }
	table.OnCellChanged = func(id TableCellID, value string) error { values[rows[id.Row]] = value; return nil }
	source := table.ensureAccessibilitySource()
	element, ok := source.Element(source.key(TableCellID{}))
	require.True(t, ok)
	cell := element.Object.(fyne.AccessibleValueSetter)
	rows[0], rows[1] = rows[1], rows[0]
	table.Refresh()
	require.True(t, cell.AccessibilitySetValueChecked("updated"))
	assert.Equal(t, "updated", values["a"])
	rows = rows[:1]
	table.Refresh()
	rows = append(rows, "a")
	table.Refresh()
	assert.False(t, cell.AccessibilitySetValueChecked("stale"))
	assert.Equal(t, "updated", values["a"])
}
