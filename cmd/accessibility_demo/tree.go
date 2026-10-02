package main

import (
	"fmt"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	treeDemoCount     = 200
	treeDemoWidth     = 600
	treeDemoHeight    = 500
	treeDemoDocuments = "documents"
	treeDemoArchive   = "archive"
	treeDemoReadme    = "readme"
)

func showTreeDemo(application fyne.App) { showTreeDemoSize(application, treeDemoCount) }

func showTreeDemoSize(application fyne.App, count int) {
	w := application.NewWindow(fmt.Sprintf("Tree accessibility — %d reports", count))
	treeDemoTarget := fmt.Sprintf("document-%03d", count-1)
	children := make([]string, count)
	names := map[string]string{treeDemoDocuments: "Documents", treeDemoArchive: "Archive", treeDemoReadme: "Read me"}
	for i := range children {
		children[i] = fmt.Sprintf("document-%03d", i)
		names[children[i]] = fmt.Sprintf("Report %d", i+1)
	}
	data := map[string][]string{"": {treeDemoDocuments, treeDemoArchive}, treeDemoDocuments: children, treeDemoArchive: {treeDemoReadme}}
	tree := widget.NewTree(func(id string) []string { return data[id] },
		func(id string) bool { _, branch := data[id]; return branch },
		func(bool) fyne.CanvasObject { return widget.NewLabel("Template") },
		func(id string, _ bool, object fyne.CanvasObject) { object.(*widget.Label).SetText(names[id]) })
	tree.DescribeNode = func(id string) fyne.AccessibilityInfo { return fyne.AccessibilityInfo{Name: names[id]} }
	tree.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Project files"})
	status := widget.NewLabel("No report selected")
	tree.OnSelected = func(id string) { status.SetText("Selected: " + names[id]) }
	tree.OnUnselected = func(string) { status.SetText("No report selected") }
	reverse := widget.NewButton("Reverse reports", func() { slices.Reverse(data[treeDemoDocuments]); tree.Refresh() })
	remove := widget.NewButton(fmt.Sprintf("Remove Report %d", count), func() {
		data[treeDemoDocuments] = slices.DeleteFunc(data[treeDemoDocuments], func(id string) bool { return id == treeDemoTarget })
		tree.Refresh()
	})
	restore := widget.NewButton(fmt.Sprintf("Restore Report %d", count), func() {
		if !slices.Contains(data[treeDemoDocuments], treeDemoTarget) {
			data[treeDemoDocuments] = append(data[treeDemoDocuments], treeDemoTarget)
			tree.Refresh()
		}
	})
	reveal := widget.NewButton(fmt.Sprintf("Go to Report %d", count), func() {
		tree.Highlight(treeDemoTarget)
		w.Canvas().Focus(tree)
	})
	hint := widget.NewLabel(fmt.Sprintf("%d reports. Check names, level, position, expansion and selection. Reorder, remove and restore the last report to check identity.", count))
	hint.Wrapping = fyne.TextWrapWord
	actions := container.NewVBox(status, container.NewGridWithColumns(2, reverse, reveal, remove, restore), widget.NewButton("Close tree demo", w.Close))
	w.SetContent(container.NewBorder(hint, actions, nil, nil, tree))
	w.Resize(fyne.NewSize(treeDemoWidth, treeDemoHeight))
	w.Canvas().Focus(tree)
	w.Show()
}
