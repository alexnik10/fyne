//go:build accessibilitydiagnostics

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/internal/accessibility/diagnostic"
	"fyne.io/fyne/v2/widget"
)

type diagnosticCheck struct {
	widget.Check
	control string
}

func newDemoCheck(id, label string, changed func(bool)) *diagnosticCheck {
	c := &diagnosticCheck{control: id}
	c.Text = label
	c.OnChanged = func(checked bool) {
		kind := "toggle_off"
		if checked {
			kind = "toggle_on"
		}
		diagnostic.Add(diagnostic.Sample{Kind: kind, Control: id})
		start := diagnostic.Start()
		if changed != nil {
			changed(checked)
		}
		diagnostic.Duration("widget_callback", 0, id, start)
	}
	c.ExtendBaseWidget(c)
	return c
}

func (c *diagnosticCheck) DiagnosticControlID() string { return c.control }

func (c *diagnosticCheck) TypedRune(r rune) {
	if r != ' ' {
		c.Check.TypedRune(r)
		return
	}
	start := diagnostic.Start()
	c.Check.TypedRune(r)
	diagnostic.Duration("check_space_dispatch", 0, c.control, start)
}

func (c *diagnosticCheck) Tapped(event *fyne.PointEvent) {
	diagnostic.Add(diagnostic.Sample{Kind: "check_tap", Control: c.control})
	start := diagnostic.Start()
	c.Check.Tapped(event)
	diagnostic.Duration("check_tap_dispatch", 0, c.control, start)
}

var (
	reportMutex sync.Mutex
	reportPath  string
)

func saveDiagnosticReport() (string, error) {
	reportMutex.Lock()
	defer reportMutex.Unlock()
	var file *os.File
	var err error
	if reportPath == "" {
		var executable string
		executable, err = os.Executable()
		if err != nil {
			return "", err
		}
		file, err = os.CreateTemp(filepath.Dir(executable), "fyne-diagnostics-*.json")
		if err == nil {
			reportPath = file.Name()
		}
	} else {
		file, err = os.Create(reportPath)
	}
	if err != nil {
		return "", err
	}
	err = diagnostic.Write(file)
	closeErr := file.Close()
	if err != nil {
		return "", err
	}
	return reportPath, closeErr
}

func diagnosticControls(window fyne.Window) []fyne.CanvasObject {
	return []fyne.CanvasObject{widget.NewButton("Save diagnostic report", func() {
		path, err := saveDiagnosticReport()
		if err != nil {
			dialog.ShowError(fmt.Errorf("cannot save diagnostic report: %w", err), window)
			return
		}
		dialog.ShowInformation("Diagnostic report saved", path, window)
	})}
}

func setupDiagnostics(application fyne.App) {
	application.Lifecycle().SetOnStopped(func() {
		if _, err := saveDiagnosticReport(); err != nil {
			fyne.LogError("Cannot save diagnostic report on exit; use Save diagnostic report before closing", err)
		}
	})
}
