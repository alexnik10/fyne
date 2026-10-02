package main

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func showLargeCollectionsDemo(application fyne.App) {
	w := application.NewWindow("Large collections")
	size := widget.NewSelect([]string{"1000", "10000", "100000"}, nil)
	size.SetSelected("100000")
	form := widget.NewForm(widget.NewFormItem("Number of reports", size))
	list := widget.NewButton("Open large list", func() { count, _ := strconv.Atoi(size.Selected); showListDemoSize(application, count) })
	tree := widget.NewButton("Open large tree", func() { count, _ := strconv.Atoi(size.Selected); showTreeDemoSize(application, count) })
	w.SetContent(container.NewVBox(form, list, tree, widget.NewButton("Close", w.Close)))
	w.Resize(fyne.NewSize(360, 220))
	w.Canvas().Focus(size)
	w.Show()
}
