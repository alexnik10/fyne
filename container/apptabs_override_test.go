//go:build !mobile

package container_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
)

func TestAppTabs_OverrideMobile(t *testing.T) {
	test.NewTempApp(t)

	item1 := &container.TabItem{Text: "Test1", Content: widget.NewLabel("Text 1")}
	item2 := &container.TabItem{Text: "Test2", Content: widget.NewLabel("Text 2")}
	item3 := &container.TabItem{Text: "Test3", Content: widget.NewLabel("Text 3")}
	tabs := container.NewAppTabs(item1, item2, item3)
	w := test.NewWindow(tabs)
	defer w.Close()
	w.SetPadded(false)
	c := w.Canvas()
	w.Resize(tabs.MinSize())

	test.AssertRendersToMarkup(t, "apptabs/desktop/tab_location_top.xml", c)

	override := container.NewThemeOverride(tabs, test.Theme())
	override.SetDeviceIsMobile(true)
	w.Resize(tabs.MinSize())

	test.AssertRendersToMarkup(t, "apptabs/mobile/tab_location_top.xml", c)
}

func TestTabs_MobileTapPreservesFocus(t *testing.T) {
	test.NewTempApp(t)
	for name, tabs := range map[string]interface {
		fyne.CanvasObject
		fyne.Focusable
		fyne.AccessibleChildren
		SelectedIndex() int
	}{
		"app": container.NewAppTabs(container.NewTabItem("One", widget.NewLabel("One")), container.NewTabItem("Two", widget.NewLabel("Two"))),
		"doc": container.NewDocTabs(container.NewTabItem("One", widget.NewLabel("One")), container.NewTabItem("Two", widget.NewLabel("Two"))),
	} {
		t.Run(name, func(t *testing.T) {
			override := container.NewThemeOverride(tabs, test.Theme())
			override.SetDeviceIsMobile(true)
			w := test.NewWindow(override)
			defer w.Close()
			w.SetPadded(false)
			w.Resize(fyne.NewSize(200, 120))
			second := tabs.AccessibilityChildren()[1]
			test.TapCanvas(w.Canvas(), second.Position().Add(fyne.NewPos(5, 5)))
			assert.Equal(t, 1, tabs.SelectedIndex())
			assert.Nil(t, w.Canvas().Focused())
			w.Canvas().Focus(tabs)
			assert.Same(t, tabs, w.Canvas().Focused())
		})
	}
}
