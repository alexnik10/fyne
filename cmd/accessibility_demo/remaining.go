package main

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	remainingRecords  = 500
	remainingLogoSize = 48
	remainingWidth    = 850
	remainingHeight   = 650
)

func showRemainingDemo(application fyne.App) {
	window := application.NewWindow("Remaining widget accessibility")
	status := widget.NewLabel("Ready")
	status.SetAccessibilityLiveSetting(fyne.AccessibilityLivePolite)
	progress := widget.NewProgressBar()
	progress.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Download progress"})
	progress.SetValue(0.25)
	activity := widget.NewActivity()
	activity.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Background work"})
	infinite := widget.NewProgressBarInfinite()
	infinite.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Unknown duration"})
	infinite.Stop()
	start := widget.NewButton("Start or stop activity", func() {
		if infinite.Running() {
			infinite.Stop()
			activity.Stop()
		} else {
			infinite.Start()
			activity.Start()
		}
	})
	window.SetOnClosed(func() { activity.Stop(); infinite.Stop() })
	save := widget.NewToolbarAction(theme.DocumentSaveIcon(), func() { status.SetText("Toolbar Save invoked") })
	save.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Save document"})
	toolbar := widget.NewToolbar(save)
	toolbar.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Document actions"})
	logo := canvas.NewImageFromResource(theme.InfoIcon())
	logo.AltText = "Information"
	logo.SetMinSize(fyne.NewSquareSize(remainingLogoSize))
	card := widget.NewCard("Download", "Read the progress below", container.NewVBox(progress, widget.NewButton("Advance progress", func() { progress.SetValue(min(1, progress.Value+0.25)) })))
	card.SetImage(logo)
	checks := widget.NewCheckGroup([]string{"Email", "Desktop"}, func(values []string) { status.SetText(fmt.Sprint(values)) })
	checks.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Notifications"})
	accordion := widget.NewAccordion(widget.NewAccordionItem("Notification settings", checks), widget.NewAccordionItem("More details", widget.NewLabel("Expanded details are available in reading order")))
	indicators := container.NewVBox(toolbar, card, activity, infinite, start, accordion)

	dates := remainingDates(status)

	text := widget.NewTextGridFromString("Read-only TextGrid\nColumns\tValue\nРусский текст 😀\nUse the screen reader's text navigation.")
	text.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Text grid document"})
	split := container.NewHSplit(text, widget.NewRichTextFromMarkdown("# Pane two\n\nTab to **Resize panes**, then use the arrow keys."))

	inner := container.NewInnerWindow("Notes", widget.NewEntry())
	windows := container.NewMultipleWindows(inner)
	dialogs := container.NewVBox(
		widget.NewButton("Choose colour", func() {
			dialog.ShowColorPicker("Choose colour", "Use swatches or named RGB/HSL fields", func(c color.Color) { status.SetText(fmt.Sprint(c)) }, window)
		}),
		widget.NewButton("Open file", func() {
			dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
				if err != nil {
					status.SetText(err.Error())
					return
				}
				if reader != nil {
					status.SetText(reader.URI().Name())
					_ = reader.Close()
				}
			}, window)
		}),
		widget.NewFileIcon(storage.NewFileURI("report.txt")),
	)
	tabs := container.NewAppTabs(
		container.NewTabItem("Status and groups", container.NewVScroll(indicators)),
		container.NewTabItem("Dates and choices", dates),
		container.NewTabItem("Grid and nested controls", remainingGrid(status)),
		container.NewTabItem("Text and split", split),
		container.NewTabItem("Inner windows", windows),
		container.NewTabItem("Dialogs", dialogs),
	)
	window.SetContent(container.NewBorder(nil, status, nil, nil, tabs))
	window.Resize(fyne.NewSize(remainingWidth, remainingHeight))
	window.Show()
}

func remainingDates(status *widget.Label) fyne.CanvasObject {
	date := widget.NewDateEntry()
	date.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Appointment date", Description: "Alt+Down opens the calendar; arrows move between dates; Enter or Space chooses a date; Escape closes it"})
	choice := widget.NewSelectEntry([]string{"English", "Russian", "German", "French"})
	choice.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Language or custom text"})
	calendar := widget.NewCalendar(time.Now(), func(value time.Time) { status.SetText(value.Format("Monday, 2 January 2006")) })
	calendar.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Appointment calendar"})
	fields := container.NewVBox(date, choice)
	return container.New(layout.NewBorderLayout(fields, nil, nil, nil), fields, calendar)
}

func remainingGrid(status *widget.Label) fyne.CanvasObject {
	keys := make([]string, remainingRecords)
	enabled := make(map[string]bool)
	for i := range keys {
		keys[i] = fmt.Sprintf("Record %03d", i)
	}
	var grid *widget.GridWrap
	change := func(key string, value bool) {
		enabled[key] = value
		status.SetText(fmt.Sprintf("%s enabled: %t", key, value))
		grid.Refresh()
	}
	grid = widget.NewGridWrap(func() int { return len(keys) }, func() fyne.CanvasObject { return widget.NewCheck("Record 000", nil) }, func(i int, obj fyne.CanvasObject) {
		check, ok := obj.(*widget.Check)
		if !ok {
			return
		}
		key := keys[i]
		check.Text, check.Checked = key, enabled[key]
		check.OnChanged = func(value bool) { change(key, value) }
		check.Refresh()
	})
	grid.ItemKey = func(i int) string { return keys[i] }
	grid.DescribeItem = func(i int) fyne.AccessibilityInfo { return fyne.AccessibilityInfo{Name: keys[i]} }
	grid.ItemElements = func(i int) []fyne.AccessibilityElement {
		key := keys[i]
		check := widget.NewCheck("Enable "+key, func(v bool) { change(key, v) })
		check.Checked = enabled[key]
		check.Resize(check.MinSize())
		return []fyne.AccessibilityElement{{Key: "enabled", Object: check}}
	}
	grid.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Keyed wrapping grid"})
	controls := container.NewHBox(widget.NewButton("Swap first and last", func() { keys[0], keys[len(keys)-1] = keys[len(keys)-1], keys[0]; grid.Refresh() }))
	return container.NewBorder(controls, nil, nil, nil, grid)
}
