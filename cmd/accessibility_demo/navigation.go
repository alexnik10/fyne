package main

import (
	"fmt"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

func showNavigationDemo(application fyne.App) {
	const reportCount, navigationWidth, navigationHeight = 200, 760, 500
	window := application.NewWindow("Tabs, menus and table accessibility")
	status := widget.NewLabel("Ready")
	notes := widget.NewMultiLineEntry()
	notes.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Document text"})
	notes.SetText("This document can be closed from its tab.")
	docs := container.NewDocTabs(container.NewTabItem("Document one", notes), container.NewTabItem("Document two", widget.NewEntry()))
	docs.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Documents"})
	nextDocument := 2
	docs.CreateTab = func() *container.TabItem {
		nextDocument++
		entry := widget.NewEntry()
		entry.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Document text"})
		return container.NewTabItem(fmt.Sprintf("Document %d", nextDocument), entry)
	}
	docs.OnClosed = func(item *container.TabItem) { status.SetText("Closed " + item.Text) }

	rows := make([]string, reportCount)
	for i := range rows {
		rows[i] = fmt.Sprintf("Report %03d", i+1)
	}
	columns := []string{"Report", "State", "Owner"}
	text := func(id widget.TableCellID) string {
		if id.Row < 0 {
			return columns[id.Col]
		}
		if id.Col < 0 {
			return fmt.Sprint(id.Row + 1)
		}
		switch id.Col {
		case 0:
			return rows[id.Row]
		case 1:
			return "Ready"
		default:
			return "Team"
		}
	}
	table := widget.NewTableWithHeaders(func() (int, int) { return len(rows), len(columns) }, func() fyne.CanvasObject { return widget.NewLabel("Report 200") }, func(id widget.TableCellID, obj fyne.CanvasObject) { obj.(*widget.Label).SetText(text(id)) })
	table.RowKey = func(row int) string { return rows[row] }
	table.ColumnKey = func(column int) string { return columns[column] }
	table.DescribeCell = func(id widget.TableCellID) fyne.AccessibilityInfo { return fyne.AccessibilityInfo{Name: text(id)} }
	table.CreateHeader = func() fyne.CanvasObject { return widget.NewLabel("Header") }
	table.UpdateHeader = func(id widget.TableCellID, obj fyne.CanvasObject) {
		if id.Row >= 0 || id.Col >= 0 {
			obj.(*widget.Label).SetText(text(id))
		}
	}
	table.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Reports table"})
	table.OnSelected = func(id widget.TableCellID) {
		status.SetText(fmt.Sprintf("Selected %s, %s", rows[id.Row], columns[id.Col]))
	}
	reverse := widget.NewButton("Reverse table rows", func() { slices.Reverse(rows); table.Refresh() })
	replace := widget.NewButton("Remove and restore Report 200", func() {
		n := slices.Index(rows, "Report 200")
		if n >= 0 {
			rows = slices.Delete(rows, n, n+1)
			table.Refresh()
			rows = append(rows, "Report 200")
			table.Refresh()
		}
	})
	reports := container.NewBorder(nil, container.NewHBox(reverse, replace), nil, nil, table)
	disabled := container.NewTabItem("Unavailable", widget.NewLabel("Disabled page"))
	tabs := container.NewAppTabs(container.NewTabItem("Documents", docs), container.NewTabItem("Reports", reports), disabled)
	tabs.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Sections"})
	tabs.DisableItem(disabled)

	check := fyne.NewMenuItem("Show notifications", nil)
	check.Checkable = true
	check.Action = func() {
		check.Checked = !check.Checked
		status.SetText(fmt.Sprintf("Notifications: %t", check.Checked))
		window.MainMenu().Refresh()
	}
	unavailable := fyne.NewMenuItem("Unavailable command", nil)
	unavailable.Disabled = true
	submenu := fyne.NewMenuItem("More commands", nil)
	submenu.ChildMenu = fyne.NewMenu("More commands", fyne.NewMenuItem("Announce ready", func() { status.SetText("Ready from submenu") }))
	openReports := fyne.NewMenuItem("Reports", func() { tabs.SelectIndex(1) })
	openReports.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyR, Modifier: fyne.KeyModifierControl}
	window.Canvas().AddShortcut(openReports.Shortcut, func(fyne.Shortcut) { openReports.Action() })
	menu := fyne.NewMenu("Actions", openReports, check, unavailable, submenu)
	window.SetMainMenu(fyne.NewMainMenu(menu))
	popup := widget.NewButton("Open context menu", func() { widget.NewPopUpMenu(menu, window.Canvas()).ShowAtPosition(fyne.NewPos(30, 60)) })
	window.SetContent(container.NewBorder(popup, status, nil, nil, tabs))
	window.Resize(fyne.NewSize(navigationWidth, navigationHeight))
	window.Show()
	window.Canvas().Focus(tabs)
}
