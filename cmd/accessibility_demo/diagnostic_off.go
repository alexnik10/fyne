//go:build !accessibilitydiagnostics

package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

func newDemoCheck(_, label string, changed func(bool)) *widget.Check {
	return widget.NewCheck(label, changed)
}

func diagnosticControls(fyne.Window) []fyne.CanvasObject { return nil }

func setupDiagnostics(fyne.App) {}
