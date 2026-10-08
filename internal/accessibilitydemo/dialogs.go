package accessibilitydemo

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

// NewDialogs is shared by the manual demo and native keyboard/UIA tests.
func NewDialogs(window fyne.Window, status *widget.Label) fyne.CanvasObject {
	open := widget.NewButton("Open file", nil)
	open.SetAccessibilityInfo(fyne.AccessibilityInfo{Description: "No file selected"})
	open.OnTapped = func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				status.SetText(err.Error())
				return
			}
			if reader == nil {
				return // Cancel preserves the previous choice.
			}
			defer reader.Close()
			message := "Selected file: " + reader.URI().Name()
			open.SetAccessibilityInfo(fyne.AccessibilityInfo{Description: message})
			status.SetText(message)
		}, window)
	}
	choose := widget.NewButton("Choose colour", func() {
		picker := dialog.NewColorPicker("Choose colour", "Use swatches or named RGB/HSL fields", func(c color.Color) {
			status.SetText(fmt.Sprint(c))
		}, window)
		picker.Advanced = true
		picker.Show()
	})
	// This is a static FileIcon example, independent of the chosen file.
	icon := widget.NewFileIcon(storage.NewFileURI("report.txt"))
	icon.SetAccessibilityInfo(fyne.AccessibilityInfo{Description: "Example file icon"})
	example := widget.NewRichTextWithText("Example file icon: report.txt")
	example.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Example file icon: report.txt"})
	return container.NewVBox(choose, open, container.NewBorder(nil, nil, icon, nil, example))
}
