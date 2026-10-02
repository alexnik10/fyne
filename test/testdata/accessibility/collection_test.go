package components_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

type collection struct{ widget.BaseWidget }

type collectionSource struct{}

func (*collection) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(widget.NewLabel("Decorative renderer"))
}
func (*collection) AccessibilityCollection() fyne.AccessibilityCollection { return collectionSource{} }
func (collectionSource) ChildCount(parent string) int {
	if parent == "" {
		return 1
	}
	return 0
}

func (collectionSource) ChildKey(parent string, index int) string {
	if parent == "" && index == 0 {
		return "record"
	}
	return ""
}

func (collectionSource) Element(key string) (fyne.AccessibilityElement, bool) {
	if key != "record" {
		return fyne.AccessibilityElement{}, false
	}
	return fyne.AccessibilityElement{Key: key, Generation: 1, Object: widget.NewLabel("Model record")}, true
}

func TestPublicIndexedCollection(t *testing.T) {
	test.NewTempApp(t)
	c := &collection{}
	c.ExtendBaseWidget(c)
	c.SetAccessibilityMode(fyne.AccessibilityGroup)
	c.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Model collection"})
	w := test.NewWindow(c)
	defer w.Close()
	s := test.NewAccessibilityTree(w.Canvas())
	element, ok := s.Element(c, "record")
	if !ok || element.Name != "Model record" || len(s.Snapshot()) != 2 || len(s.Issues()) != 0 {
		t.Fatal("public indexed source did not replace renderer semantics")
	}
}
