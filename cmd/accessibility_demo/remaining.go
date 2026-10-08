package main

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/internal/accessibilitydemo"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
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

	split := accessibilitydemo.NewTextSplit()

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
		container.NewTabItem("Grid and nested controls", accessibilitydemo.NewGrid(status)),
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
