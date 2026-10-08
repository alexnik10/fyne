package accessibilitydemo

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// NewTextSplit exposes both read-only documents in the split's keyboard order.
func NewTextSplit() *container.Split {
	text := widget.NewTextGridFromString("Read-only TextGrid\nColumns\tValue\nРусский текст 😀\nUse the arrow keys to read; Shift selects and Control+C copies.")
	text.Selectable = true
	text.Scroll = fyne.ScrollBoth
	text.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Text grid document", Description: "Read-only text. Arrow keys read, Shift selects, Control+C copies. Tab moves to Resize panes."})
	second := widget.NewRichTextFromMarkdown("# Pane two\n\nBoth panes contain read-only text. Use the arrow keys to read, Shift to select, and Control+C to copy.\n\nTab or Shift+Tab moves between the documents and Resize panes.")
	second.Selectable = true
	second.Wrapping = fyne.TextWrapWord
	second.Scroll = fyne.ScrollVerticalOnly
	second.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Pane two document", Description: "Read-only text. Shift+Tab moves to Resize panes."})
	return container.NewHSplit(text, second)
}
