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
	setupDiagnostics(application)
	window := application.NewWindow("Fyne accessibility")
	email := widget.NewEntry()
	email.Validator = func(value string) error {
		if !strings.Contains(value, "@") {
			return errors.New("enter an email address containing @")
		}
		return nil
	}
	password := widget.NewPasswordEntry()
	remember := newDemoCheck("remember", "Remember this account", nil)
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
	trees := widget.NewButton("Open tree demo", func() { showTreeDemo(application) })
	lists := widget.NewButton("Open list demo", func() { showListDemo(application) })
	large := widget.NewButton("Open large collections demo", func() { showLargeCollectionsDemo(application) })
	navigation := widget.NewButton("Open tabs, menus and table demo", func() { showNavigationDemo(application) })
	scrolling := widget.NewButton("Open scrolling demo", func() { showScrollDemo(application) })
	content := []fyne.CanvasObject{form, status, choices, textEditing, trees, lists, large, navigation, scrolling, second}
	content = append(content, diagnosticControls(window)...)
	window.SetContent(container.NewVBox(content...))
	window.Resize(fyne.NewSize(windowWidth, windowHeight))
	window.ShowAndRun()
}

func showSelectionDemo(application fyne.App) {
	window := application.NewWindow("Selection accessibility demo 7")
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
	disable := newDemoCheck("disable", "Disable choices", func(disabled bool) {
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
	content := []fyne.CanvasObject{form, disable, status, widget.NewButton("Close selection demo", window.Close)}
	content = append(content, diagnosticControls(window)...)
	window.SetContent(container.NewVBox(content...))
	window.Resize(fyne.NewSize(windowWidth, windowHeight))
	window.Canvas().Focus(language)
	window.Show()
}

func showTextDemo(application fyne.App) {
	window := application.NewWindow("Multiline text accessibility")
	entry := widget.NewMultiLineEntry()
	entry.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Notes", Description: "Multiline editing. Tab inserts a tab. Ctrl+Tab moves to the next control; Ctrl+Shift+Tab moves to the previous control."})
	entry.SetText("First line: user@example.org\nSecond line: foo_bar\nТретья строка: проверка выделения")
	hint := widget.NewLabel("Tab inserts a tab. Ctrl+Tab moves to Close text demo.")
	hint.Wrapping = fyne.TextWrapWord
	window.SetContent(container.NewBorder(hint, widget.NewButton("Close text demo", window.Close), nil, nil, entry))
	window.Resize(fyne.NewSize(windowWidth, windowHeight))
	window.Canvas().Focus(entry)
	window.Show()
}
