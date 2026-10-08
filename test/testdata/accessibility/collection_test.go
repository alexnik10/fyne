package components_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

type collection struct{ widget.BaseWidget }

type collectionSource struct{}

var _ fyne.AccessibilityCollectionView = collectionSource{}

func (collectionSource) Revision() uint64                     { return 1 }
func (collectionSource) ViewportKeys() []string               { return nil }
func (collectionSource) SelectedKeys() []string               { return nil }
func (collectionSource) Index(key string) (string, int, bool) { return "", 0, key == "record" }

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
	if len(s.Snapshot()) != 1 {
		t.Fatal("offscreen model record was eagerly described")
	}
	element, ok := s.Element(c, "record")
	if !ok || element.Name != "Model record" || len(s.Snapshot()) != 2 || len(s.Issues()) != 0 {
		t.Fatal("public indexed source did not replace renderer semantics")
	}
}
