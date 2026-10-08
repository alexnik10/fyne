package widget

import (
	"errors"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
)

type tableCellEdit struct {
	key        string
	generation uint64
	entry      *tableCellEntry
	popup      *PopUp
}

// SetCellValue validates and stores a model value, then refreshes the table.
// It does not select the cell or move keyboard focus. Read-only, removed and
// unconfigured cells reject the command. Callbacks run on the Fyne event thread.
// Since: 2.9
func (t *Table) SetCellValue(id TableCellID, value string) error {
	if !t.ensureAccessibilitySource().valid(id) || t.CellValue == nil || t.OnCellChanged == nil {
		return errors.New("cell is not editable")
	}
	if _, readOnly := t.CellValue(id); readOnly {
		return errors.New("cell is read-only")
	}
	if err := t.OnCellChanged(id, value); err != nil {
		return err
	}
	t.Refresh()
	return nil
}

// EditCell opens a text editor for a model cell. F2 and Enter use this same path.
// Enter saves, Escape cancels, and Tab visits the editor's Save/Cancel buttons.
// The editor follows RowKey/ColumnKey across reorder and cancels if its cell is
// removed or becomes read-only. Positional tables retain positional identity.
// Since: 2.9
func (t *Table) EditCell(id TableCellID) bool {
	s := t.ensureAccessibilitySource()
	if !s.valid(id) || t.CellValue == nil || t.OnCellChanged == nil {
		return false
	}
	value, readOnly := t.CellValue(id)
	c := fyne.CurrentApp().Driver().CanvasForObject(t.super())
	if readOnly || c == nil {
		return false
	}
	t.CancelCellEdit()
	t.Highlight(id)
	t.ScrollTo(id)
	element, ok := s.Element(s.key(id))
	if !ok {
		return false
	}
	name := element.Object.(fyne.Accessible).AccessibilityLabel()
	if info := element.Object.(fyne.AccessibleDescribed).AccessibilityInfo(); info.NameSet || info.Name != "" {
		name = info.Name
	}
	entry := &tableCellEntry{table: t}
	entry.ExtendBaseWidget(entry)
	entry.AlwaysShowValidationError = true
	entry.SetText(value)
	entry.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: name, Description: lang.L("Enter saves; Escape cancels")})
	save := func() { _ = t.CommitCellEdit() }
	entry.OnSubmitted = func(string) { save() }
	button := func(label string, action func()) *tableCellButton {
		b := &tableCellButton{Button: Button{Text: label, OnTapped: action}, table: t}
		b.ExtendBaseWidget(b)
		return b
	}
	buttons := &fyne.Container{Layout: layout.NewHBoxLayout(), Objects: []fyne.CanvasObject{button(lang.L("Save"), save), button(lang.L("Cancel"), t.CancelCellEdit)}}
	content := &fyne.Container{Layout: layout.NewVBoxLayout(), Objects: []fyne.CanvasObject{NewLabel(name), entry, buttons}}
	popup := NewModalPopUp(content, c)
	popup.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: lang.L("Edit cell")})
	edit := &tableCellEdit{key: element.Key, generation: element.Generation, entry: entry, popup: popup}
	t.cellEdit = edit
	popup.OnDismiss = func() {
		if t.cellEdit == edit {
			t.cellEdit = nil
		}
	}
	if focus, ok := t.super().(fyne.Focusable); ok {
		c.Focus(focus)
	}
	popup.Show()
	c.Focus(entry)
	entry.AccessibilitySelectText(0, len([]rune(value)))
	return true
}

// CommitCellEdit saves the active editor. Validation errors leave it open.
// Since: 2.9
func (t *Table) CommitCellEdit() error {
	edit := t.cellEdit
	if edit == nil {
		return errors.New("no cell editor is open")
	}
	id, ok := t.editCellID(edit)
	if !ok {
		t.CancelCellEdit()
		return errors.New("edited cell is no longer available")
	}
	if err := t.SetCellValue(id, edit.entry.Text); err != nil {
		if t.cellEdit == edit {
			edit.entry.SetValidationError(err)
			edit.popup.Canvas.Focus(edit.entry)
		}
		return err
	}
	if t.cellEdit == edit {
		t.CancelCellEdit()
	}
	return nil
}

// CancelCellEdit closes the editor without storing its draft.
// Since: 2.9
func (t *Table) CancelCellEdit() {
	if edit := t.cellEdit; edit != nil {
		t.cellEdit = nil
		edit.popup.Hide()
	}
}

// Hide closes any cell editor before hiding the table.
func (t *Table) Hide() {
	t.CancelCellEdit()
	t.BaseWidget.Hide()
}

func (t *Table) editCellID(edit *tableCellEdit) (TableCellID, bool) {
	s := t.ensureAccessibilitySource()
	e, ok := s.Element(edit.key)
	if !ok || e.Generation != edit.generation || t.CellValue == nil || t.OnCellChanged == nil {
		return TableCellID{}, false
	}
	id, ok := s.resolve(edit.key)
	if !ok {
		return id, false
	}
	_, readOnly := t.CellValue(id)
	return id, !readOnly
}

func (t *Table) reconcileCellEdit() {
	if edit := t.cellEdit; edit != nil {
		if _, ok := t.editCellID(edit); !ok {
			t.CancelCellEdit()
		}
	}
}

type tableCellEntry struct {
	Entry
	table *Table
}

func (e *tableCellEntry) TypedKey(event *fyne.KeyEvent) {
	if event.Name == fyne.KeyEscape {
		e.table.CancelCellEdit()
		return
	}
	e.Entry.TypedKey(event)
}

type tableAccessibilityValueCell struct{ tableAccessibilityCell }

func (i *tableAccessibilityValueCell) AccessibilityValue() (text string, readOnly, protected bool) {
	id, ok := i.resolve()
	if !ok || i.owner.CellValue == nil {
		return "", true, false
	}
	value, readOnly := i.owner.CellValue(id)
	return value, readOnly || i.owner.OnCellChanged == nil, false
}

type tableCellButton struct {
	Button
	table *Table
}

func (b *tableCellButton) TypedKey(event *fyne.KeyEvent) {
	if event.Name == fyne.KeyEscape {
		b.table.CancelCellEdit()
		return
	}
	b.Button.TypedKey(event)
}

func (i *tableAccessibilityValueCell) AccessibilitySetValue(value string) {
	i.AccessibilitySetValueChecked(value)
}

func (i *tableAccessibilityValueCell) AccessibilitySetValueChecked(value string) bool {
	id, ok := i.resolve()
	return ok && i.owner.SetCellValue(id, value) == nil
}

type tableAccessibilityEditableCell struct{ tableAccessibilityValueCell }

func (i *tableAccessibilityEditableCell) AccessibilityActivate() {
	if id, ok := i.resolve(); ok {
		i.owner.EditCell(id)
	}
}
