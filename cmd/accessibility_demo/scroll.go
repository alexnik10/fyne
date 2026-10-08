package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func showScrollDemo(application fyne.App) {
	const rowCount, columnCount, scrollWindowWidth, scrollWindowHeight = 40, 8, 760, 500
	window := application.NewWindow("Scroll accessibility")
	status := widget.NewLabel("Choose a cell; scrolling keeps its selection.")
	var cells []fyne.CanvasObject
	for row := 0; row < rowCount; row++ {
		for column := 0; column < columnCount; column++ {
			name := fmt.Sprintf("Row %02d, column %d", row+1, column+1)
			cells = append(cells, widget.NewButton(name, func() { status.SetText("Activated " + name) }))
		}
	}
	content := container.NewGridWithColumns(columnCount, cells...)
	scroll := container.NewScroll(content)
	scroll.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Scrollable button grid", Description: "Forty rows and eight columns. Both axes can be scrolled."})
	fit := widget.NewCheck("Use short content", func(short bool) {
		if short {
			scroll.Content = widget.NewButton("All content is visible", nil)
		} else {
			scroll.Content = content
		}
		scroll.Refresh()
	})
	table := widget.NewTableWithHeaders(func() (int, int) { return rowCount, columnCount }, func() fyne.CanvasObject { return widget.NewLabel("Row 40, column 8") }, func(id widget.TableCellID, obj fyne.CanvasObject) {
		obj.(*widget.Label).SetText(fmt.Sprintf("Row %02d, column %d", id.Row+1, id.Col+1))
	})
	table.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Scrollable table"})
	table.DescribeCell = func(id widget.TableCellID) fyne.AccessibilityInfo {
		if id.Row < 0 {
			return fyne.AccessibilityInfo{Name: fmt.Sprintf("Column %d", id.Col+1)}
		}
		if id.Col < 0 {
			return fyne.AccessibilityInfo{Name: fmt.Sprintf("Row %d", id.Row+1)}
		}
		return fyne.AccessibilityInfo{Name: fmt.Sprintf("Row %02d, column %d", id.Row+1, id.Col+1)}
	}
	table.OnSelected = func(id widget.TableCellID) {
		status.SetText(fmt.Sprintf("Selected row %d, column %d", id.Row+1, id.Col+1))
	}
	table.StickyRowCount, table.StickyColumnCount = 1, 1
	tabs := container.NewAppTabs(container.NewTabItem("Buttons", container.NewBorder(fit, nil, nil, nil, scroll)), container.NewTabItem("Table", table))
	tabs.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Scrolling examples"})
	window.SetContent(container.NewBorder(nil, status, nil, nil, tabs))
	window.Resize(fyne.NewSize(scrollWindowWidth, scrollWindowHeight))
	window.Show()
}
