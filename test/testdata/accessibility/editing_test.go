package components_test

import (
	"errors"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestPublicEditableTableAndRichText(t *testing.T) {
	test.NewTempApp(t)
	value := "Cell"
	table := widget.NewTable(func() (int, int) { return 1, 1 }, func() fyne.CanvasObject { return widget.NewLabel("") }, func(widget.TableCellID, fyne.CanvasObject) {})
	table.CellValue = func(widget.TableCellID) (string, bool) { return value, false }
	table.OnCellChanged = func(_ widget.TableCellID, next string) error {
		if next == "" {
			return errors.New("required")
		}
		value = next
		return nil
	}
	w := test.NewWindow(table)
	defer w.Close()
	tree := test.NewAccessibilityTree(w.Canvas())
	cell, ok := tree.Element(table, table.AccessibilityCellKey(0, 0))
	if !ok || !cell.Value || cell.ReadOnly {
		t.Fatal("missing editable Value contract")
	}
	if tree.Perform(cell.ID, test.AccessibilitySetValue, "", 0) || value != "Cell" {
		t.Fatal("validation did not reject the command")
	}
	if !table.EditCell(widget.TableCellID{}) {
		t.Fatal("cannot open editor")
	}
	table.CancelCellEdit()
	rich := widget.NewRichTextEntryFromMarkdown("Plain **bold**")
	w.SetContent(rich)
	w.Resize(fyne.NewSize(200, 100))
	var text fyne.AccessibleText = rich
	info := text.AccessibilityText()
	if len(info.Runs) != 2 || !info.Runs[1].Style.Bold {
		t.Fatalf("incorrect public runs: %+v", info.Runs)
	}
	text.AccessibilitySelectText(6, 10)
	if rich.SelectedText() != "bold" {
		t.Fatal("selection does not match rune offsets")
	}
	var setter fyne.AccessibleValue = rich
	setter.AccessibilitySetValue("Replaced")
	if rich.Text != "Replaced" || rich.Markdown() != "Replaced" {
		t.Fatal("rich value setter bypassed the rich model")
	}
}
