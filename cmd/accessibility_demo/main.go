// accessibility_demo is a small Windows screen-reader acceptance scenario.
// Run with: go run -tags accessibility ./cmd/accessibility_demo
package main

import (
	"errors"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	maximumVolume = 10
	windowWidth   = 520
	windowHeight  = 420
)

func main() {
	application := app.NewWithID("io.fyne.accessibility-demo")
	window := application.NewWindow("Fyne accessibility")
	email := widget.NewEntry()
	email.Validator = func(value string) error {
		if !strings.Contains(value, "@") {
			return errors.New("enter an email address containing @")
		}
		return nil
	}
	password := widget.NewPasswordEntry()
	remember := widget.NewCheck("Remember this account", nil)
	volume := widget.NewSlider(0, maximumVolume)
	status := widget.NewLabel("Ready")
	form := widget.NewForm(
		widget.NewFormItem("Email", email),
		widget.NewFormItem("Password", password),
		widget.NewFormItem("", remember),
		widget.NewFormItem("Volume", volume),
	)
	form.Items[0].Required = true
	form.Items[0].HintText = "For example user@example.org"
	form.SubmitText = "Save"
	form.OnSubmit = func() {
		content := container.NewVBox(widget.NewLabel("Account settings saved"))
		popup := widget.NewModalPopUp(content, window.Canvas())
		popup.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Settings saved"})
		content.Add(widget.NewButton("Close confirmation", popup.Hide))
		popup.Show()
		status.SetText(fmt.Sprintf("Saved settings for %s", email.Text))
	}
	second := widget.NewButton("Open another window", func() {
		other := application.NewWindow("Second accessible window")
		closeButton := widget.NewButton("Close this window", other.Close)
		other.SetContent(closeButton)
		other.Canvas().Focus(closeButton)
		other.Show()
	})
	choices := widget.NewButton("Open selection demo", func() { showSelectionDemo(application) })
	textEditing := widget.NewButton("Open multiline text demo", func() { showTextDemo(application) })
	window.SetContent(container.NewVBox(form, status, choices, textEditing, second))
	window.Resize(fyne.NewSize(windowWidth, windowHeight))
	window.ShowAndRun()
}

func showSelectionDemo(application fyne.App) {
	window := application.NewWindow("Selection accessibility demo 4")
	status := widget.NewLabel("Change a choice using the keyboard")
	language := widget.NewSelect([]string{"English", "Russian", "German"}, func(value string) {
		status.SetText("Language: " + value)
	})
	language.SetSelected("English")
	notifications := widget.NewRadioGroup([]string{"Daily", "Weekly", "Never"}, func(value string) {
		status.SetText("Notifications: " + value)
	})
	notifications.Required = true
	notifications.SetSelected("Daily")
	optional := widget.NewRadioGroup([]string{"Light", "Dark"}, func(value string) {
		status.SetText("Optional appearance: " + value)
	})
	optional.Horizontal = true
	form := widget.NewForm(
		widget.NewFormItem("Language", language),
		widget.NewFormItem("Notifications", notifications),
		widget.NewFormItem("Optional appearance", optional),
	)
	form.Items[1].Required = true
	disable := widget.NewCheck("Disable choices", func(disabled bool) {
		if disabled {
			language.Disable()
			notifications.Disable()
			optional.Disable()
		} else {
			language.Enable()
			notifications.Enable()
			optional.Enable()
		}
	})
	window.SetContent(container.NewVBox(form, disable, status, widget.NewButton("Close selection demo", window.Close)))
	window.Resize(fyne.NewSize(windowWidth, windowHeight))
	window.Canvas().Focus(language)
	window.Show()
}

func showTextDemo(application fyne.App) {
	window := application.NewWindow("Multiline text accessibility")
	entry := widget.NewMultiLineEntry()
	entry.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Notes", Description: "Multiline editing and selection"})
	entry.SetText("First line: user@example.org\nSecond line: foo_bar\nТретья строка: проверка выделения")
	window.SetContent(container.NewBorder(nil, widget.NewButton("Close text demo", window.Close), nil, nil, entry))
	window.Resize(fyne.NewSize(windowWidth, windowHeight))
	window.Canvas().Focus(entry)
	window.Show()
}
