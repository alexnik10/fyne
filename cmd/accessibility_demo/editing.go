package main

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	editingDemoWidth  = 850
	editingDemoHeight = 560
	editingDemoRows   = 200
)

func showEditingDemo(application fyne.App) {
	w := application.NewWindow("Editable cells and formatted text")
	status := widget.NewLabel("Ready")
	table := editingTable(status)
	rows := widget.NewLabel("Use arrows in the table; F2 or Enter edits. Enter saves; Escape cancels. ID is read-only. Title is required; Count must be a nonnegative integer.")
	rows.Wrapping = fyne.TextWrapWord
	tablePage := container.NewBorder(rows, status, nil, nil, table)
	const sample = "# Formatted document\n\nPlain, **bold**, *italic*, and ~~struck through~~ text.\n\n* First item\n* Second item\n\n```\nCode sample\n```\n\nРусский текст и emoji 😀."
	editor := widget.NewRichTextEntryFromMarkdown(sample)
	editor.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Formatted editor", Description: "Select text then use Bold or Italic. Ctrl+Tab leaves the editor."})
	preview := widget.NewRichTextFromMarkdown(sample)
	preview.Selectable = true
	preview.Wrapping = fyne.TextWrapWord
	preview.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Read-only formatted document"})
	styleButton := func(name string, change func(*widget.RichTextStyle)) *widget.Button {
		return widget.NewButton(name, func() {
			style := editor.StyleAtCursor()
			change(&style)
			editor.SetStyleForSelection(style)
			w.Canvas().Focus(editor)
		})
	}
	format := container.NewHBox(styleButton("Bold", func(s *widget.RichTextStyle) { s.TextStyle.Bold = !s.TextStyle.Bold }), styleButton("Italic", func(s *widget.RichTextStyle) { s.TextStyle.Italic = !s.TextStyle.Italic }), widget.NewButton("Update read-only document", func() { preview.ParseMarkdown(editor.Markdown()) }))
	richPage := container.NewBorder(format, nil, nil, nil, container.NewHSplit(editor, container.NewVScroll(preview)))
	w.SetContent(container.NewAppTabs(container.NewTabItem("Editable Table", tablePage), container.NewTabItem("Formatted text", richPage)))
	w.Resize(fyne.NewSize(editingDemoWidth, editingDemoHeight))
	w.Show()
}

func editingTable(status *widget.Label) fyne.CanvasObject {
	type record struct {
		key, title string
		count      int
	}
	rows := make([]record, editingDemoRows)
	for i := range rows {
		rows[i] = record{fmt.Sprintf("report-%d", i+1), fmt.Sprintf("Report %d", i+1), i + 1}
	}
	columns := []string{"ID", "Title", "Count"}
	value := func(id widget.TableCellID) string {
		if id.Row < 0 {
			return columns[id.Col]
		}
		if id.Col < 0 {
			return rows[id.Row].key
		}
		switch id.Col {
		case 0:
			return rows[id.Row].key
		case 1:
			return rows[id.Row].title
		default:
			return strconv.Itoa(rows[id.Row].count)
		}
	}
	table := widget.NewTableWithHeaders(func() (int, int) { return len(rows), len(columns) }, func() fyne.CanvasObject { return widget.NewLabel("Report 200") }, func(id widget.TableCellID, obj fyne.CanvasObject) { obj.(*widget.Label).SetText(value(id)) })
	table.RowKey = func(row int) string { return rows[row].key }
	table.ColumnKey = func(col int) string { return columns[col] }
	table.CreateHeader = func() fyne.CanvasObject { return widget.NewLabel("Report 200") }
	table.UpdateHeader = func(id widget.TableCellID, obj fyne.CanvasObject) { obj.(*widget.Label).SetText(value(id)) }
	table.DescribeCell = func(id widget.TableCellID) fyne.AccessibilityInfo {
		if id.Row < 0 || id.Col < 0 {
			return fyne.AccessibilityInfo{Name: value(id)}
		}
		return fyne.AccessibilityInfo{Name: rows[id.Row].key + ", " + columns[id.Col]}
	}
	table.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Editable reports"})
	table.CellValue = func(id widget.TableCellID) (string, bool) { return value(id), id.Col == 0 }
	table.OnCellChanged = func(id widget.TableCellID, text string) error {
		switch id.Col {
		case 1:
			if strings.TrimSpace(text) == "" {
				return errors.New("title is required")
			}
			rows[id.Row].title = text
		case 2:
			n, err := strconv.Atoi(text)
			if err != nil || n < 0 {
				return errors.New("count must be a nonnegative integer")
			}
			rows[id.Row].count = n
		}
		status.SetText("Saved " + rows[id.Row].key + ", " + columns[id.Col])
		return nil
	}
	controls := container.NewHBox(widget.NewButton("Reverse rows", func() { slices.Reverse(rows); table.Refresh() }), widget.NewButton("Remove first row", func() {
		if len(rows) != 0 {
			rows = rows[1:]
			table.Refresh()
		}
	}))
	return container.NewBorder(controls, nil, nil, nil, table)
}
