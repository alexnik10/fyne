package main

import (
	"fmt"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	listDemoCount  = 200
	listDemoTarget = "report-199"
)

func showListDemo(application fyne.App) {
	w := application.NewWindow("Keyed list accessibility")
	items := make([]string, listDemoCount)
	names := make(map[string]string, listDemoCount)
	for i := range items {
		items[i] = fmt.Sprintf("report-%03d", i)
		names[items[i]] = fmt.Sprintf("Report %d", i+1)
	}
	list := widget.NewList(func() int { return len(items) },
		func() fyne.CanvasObject { return widget.NewLabel("Template") },
		func(id int, obj fyne.CanvasObject) { obj.(*widget.Label).SetText(names[items[id]]) })
	list.ItemKey = func(id int) string { return items[id] }
	list.DescribeItem = func(id int) fyne.AccessibilityInfo { return fyne.AccessibilityInfo{Name: names[items[id]]} }
	list.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Reports", Description: "Arrow keys move focus. Space selects a report."})
	status := widget.NewLabel("No report selected")
	list.OnSelected = func(id int) { status.SetText("Selected: " + names[items[id]]) }
	list.OnUnselected = func(int) { status.SetText("No report selected") }
	reverse := widget.NewButton("Reverse reports", func() { slices.Reverse(items); list.Refresh() })
	remove := widget.NewButton("Remove Report 200", func() {
		items = slices.DeleteFunc(items, func(key string) bool { return key == listDemoTarget })
		list.Refresh()
	})
	restore := widget.NewButton("Restore Report 200", func() {
		if !slices.Contains(items, listDemoTarget) {
			items = append(items, listDemoTarget)
			list.Refresh()
		}
	})
	reveal := widget.NewButton("Go to Report 200", func() {
		if index := slices.Index(items, listDemoTarget); index >= 0 {
			list.Highlight(index)
			w.Canvas().Focus(list)
		}
	})
	hint := widget.NewLabel("Check row names, position and selection. Select Report 200, then reverse the list: selection should follow the report. Removing and restoring it creates a new item.")
	hint.Wrapping = fyne.TextWrapWord
	actions := container.NewVBox(status, container.NewGridWithColumns(2, reverse, reveal, remove, restore), widget.NewButton("Close list demo", w.Close))
	w.SetContent(container.NewBorder(hint, actions, nil, nil, list))
	w.Resize(fyne.NewSize(treeDemoWidth, treeDemoHeight))
	w.Canvas().Focus(list)
	w.Show()
}
