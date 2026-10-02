package main

import (
	"fmt"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const listDemoCount = 200

func showListDemo(application fyne.App) { showListDemoSize(application, listDemoCount) }

func showListDemoSize(application fyne.App, count int) {
	w := application.NewWindow(fmt.Sprintf("List accessibility — %d reports", count))
	listDemoTarget := fmt.Sprintf("report-%03d", count-1)
	items := make([]string, count)
	names := make(map[string]string, count)
	for i := range items {
		items[i] = fmt.Sprintf("report-%03d", i)
		names[items[i]] = fmt.Sprintf("Report %d", i+1)
	}
	list := widget.NewList(func() int { return len(items) },
		func() fyne.CanvasObject { return widget.NewLabel("Template") },
		func(id int, obj fyne.CanvasObject) { obj.(*widget.Label).SetText(names[items[id]]) })
	list.ItemKey = func(id int) string { return items[id] }
	list.DescribeItem = func(id int) fyne.AccessibilityInfo { return fyne.AccessibilityInfo{Name: names[items[id]]} }
	list.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Reports"})
	status := widget.NewLabel("No report selected")
	list.OnSelected = func(id int) { status.SetText("Selected: " + names[items[id]]) }
	list.OnUnselected = func(int) { status.SetText("No report selected") }
	reverse := widget.NewButton("Reverse reports", func() { slices.Reverse(items); list.Refresh() })
	remove := widget.NewButton(fmt.Sprintf("Remove Report %d", count), func() {
		items = slices.DeleteFunc(items, func(key string) bool { return key == listDemoTarget })
		list.Refresh()
	})
	restore := widget.NewButton(fmt.Sprintf("Restore Report %d", count), func() {
		if !slices.Contains(items, listDemoTarget) {
			items = append(items, listDemoTarget)
			list.Refresh()
		}
	})
	reveal := widget.NewButton(fmt.Sprintf("Go to Report %d", count), func() {
		if index := slices.Index(items, listDemoTarget); index >= 0 {
			list.Highlight(index)
			w.Canvas().Focus(list)
		}
	})
	hint := widget.NewLabel(fmt.Sprintf("%d reports. Select the last report, then reverse the list: selection should follow it. Removing and restoring it creates a new item.", count))
	hint.Wrapping = fyne.TextWrapWord
	actions := container.NewVBox(status, container.NewGridWithColumns(2, reverse, reveal, remove, restore), widget.NewButton("Close list demo", w.Close))
	w.SetContent(container.NewBorder(hint, actions, nil, nil, list))
	w.Resize(fyne.NewSize(treeDemoWidth, treeDemoHeight))
	w.Canvas().Focus(list)
	w.Show()
}
