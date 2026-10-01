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

func main() {
	application := app.NewWithID("io.fyne.accessibility-demo")
	window := application.NewWindow("Fyne accessibility")
	email := widget.NewEntry()
	email.Validator = func(value string) error {
		if !strings.Contains(value, "@") {
			return errors.New("Enter an email address containing @")
		}
		return nil
	}
	password := widget.NewPasswordEntry()
	remember := widget.NewCheck("Remember this account", nil)
	volume := widget.NewSlider(0, 10)
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
		other.SetContent(widget.NewButton("Close this window", other.Close))
		other.Show()
	})
	window.SetContent(container.NewVBox(form, status, second))
	window.Resize(fyne.NewSize(520, 420))
	window.ShowAndRun()
}
