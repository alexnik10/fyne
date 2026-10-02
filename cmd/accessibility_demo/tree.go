package main

import (
	"fmt"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func showTreeDemo(application fyne.App) {
	w := application.NewWindow("Keyed tree accessibility")
	children := make([]string, 200)
	names := map[string]string{"documents": "Documents", "archive": "Archive", "readme": "Read me"}
	for i := range children {
		children[i] = fmt.Sprintf("document-%03d", i)
		names[children[i]] = fmt.Sprintf("Report %d", i+1)
	}
	data := map[string][]string{"": {"documents", "archive"}, "documents": children, "archive": {"readme"}}
	tree := widget.NewTree(func(id string) []string { return data[id] },
		func(id string) bool { _, branch := data[id]; return branch },
		func(bool) fyne.CanvasObject { return widget.NewLabel("Template") },
		func(id string, _ bool, object fyne.CanvasObject) { object.(*widget.Label).SetText(names[id]) })
	tree.DescribeNode = func(id string) fyne.AccessibilityInfo { return fyne.AccessibilityInfo{Name: names[id]} }
	tree.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Project files", Description: "Arrow keys navigate and expand branches. Space selects a report."})
	status := widget.NewLabel("No report selected")
	tree.OnSelected = func(id string) { status.SetText("Selected: " + names[id]) }
	tree.OnUnselected = func(string) { status.SetText("No report selected") }
	reverse := widget.NewButton("Reverse reports", func() { slices.Reverse(data["documents"]); tree.Refresh() })
	remove := widget.NewButton("Remove Report 200", func() {
		data["documents"] = slices.DeleteFunc(data["documents"], func(id string) bool { return id == "document-199" })
		tree.Refresh()
	})
	restore := widget.NewButton("Restore Report 200", func() {
		if !slices.Contains(data["documents"], "document-199") {
			data["documents"] = append(data["documents"], "document-199")
			tree.Refresh()
		}
	})
	reveal := widget.NewButton("Go to Report 200", func() {
		tree.Highlight("document-199")
		w.Canvas().Focus(tree)
	})
	hint := widget.NewLabel("Check names, level, position, expansion and selection. Report 200 starts outside the viewport. Reorder, remove and restore it to check identity.")
	hint.Wrapping = fyne.TextWrapWord
	actions := container.NewVBox(status, container.NewGridWithColumns(2, reverse, reveal, remove, restore), widget.NewButton("Close tree demo", w.Close))
	w.SetContent(container.NewBorder(hint, actions, nil, nil, tree))
	w.Resize(fyne.NewSize(600, 500))
	w.Canvas().Focus(tree)
	w.Show()
}
