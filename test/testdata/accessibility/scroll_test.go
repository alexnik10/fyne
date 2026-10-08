package components_test

import (
	"math"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

type viewport struct {
	widget.DisableableWidget
	info  fyne.AccessibilityScrollInfo
	calls int
}

var _ fyne.AccessibleScroll = (*viewport)(nil)

func (*viewport) AccessibilityLabel() string { return "Custom viewport" }

func (*viewport) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleContainer }

func (v *viewport) AccessibilityScroll() fyne.AccessibilityScrollInfo { return v.info }

func (v *viewport) AccessibilityScrollTo(offset fyne.Position) bool {
	v.info.Offset = offset
	v.calls++
	return true
}

func (*viewport) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(widget.NewLabel("Viewport renderer"))
}

func TestPublicScrollContract(t *testing.T) {
	test.NewTempApp(t)
	v := &viewport{info: fyne.AccessibilityScrollInfo{
		ContentSize: fyne.NewSize(1000, 800), ViewportSize: fyne.NewSize(100, 200),
		SmallStep: fyne.NewSize(5, 10), Direction: fyne.ScrollBoth,
	}}
	v.ExtendBaseWidget(v)
	w := test.NewWindow(v)
	defer w.Close()
	tree := test.NewAccessibilityTree(w.Canvas())
	n, ok := tree.Node(v)
	if !ok || !n.Scroll || n.HorizontalViewSize != 10 || n.VerticalViewSize != 25 {
		t.Fatalf("incorrect custom viewport: %+v", n)
	}
	if !tree.Scroll(n.ID, fyne.AccessibilityScrollSmallIncrement, fyne.AccessibilityScrollLargeIncrement) || v.info.Offset != fyne.NewPos(5, 190) {
		t.Fatalf("wrong steps: %v", v.info.Offset)
	}
	if !tree.SetScrollPercent(n.ID, 100, 50) || v.info.Offset != fyne.NewPos(900, 300) {
		t.Fatalf("wrong absolute offset: %v", v.info.Offset)
	}
	before := v.calls
	v.Disable()
	if tree.SetScrollPercent(n.ID, 0, 0) || tree.Scroll(n.ID, fyne.AccessibilityScrollNone, fyne.AccessibilityScrollSmallIncrement) || v.calls != before {
		t.Fatal("disabled viewport accepted a command")
	}
	n, _ = tree.Node(v)
	if n.VerticalScrollPercent != 50 || n.VerticalViewSize != 25 {
		t.Fatal("disabled state changed scroll metrics")
	}
	v.Enable()
	for _, direction := range []fyne.ScrollDirection{fyne.ScrollBoth, fyne.ScrollHorizontalOnly, fyne.ScrollVerticalOnly, fyne.ScrollNone} {
		v.info.Direction = direction
		n, _ = tree.Node(v)
		horizontal := direction == fyne.ScrollBoth || direction == fyne.ScrollHorizontalOnly
		vertical := direction == fyne.ScrollBoth || direction == fyne.ScrollVerticalOnly
		if (n.HorizontalScrollPercent >= 0) != horizontal || (n.VerticalScrollPercent >= 0) != vertical {
			t.Fatalf("wrong axes for direction %v: %+v", direction, n)
		}
	}
	v.info.Direction = fyne.ScrollBoth
	for _, invalid := range []float32{0, -1, float32(math.NaN()), float32(math.Inf(1))} {
		v.info.ViewportSize.Height = invalid
		n, _ = tree.Node(v)
		if n.VerticalScrollPercent != -1 || n.VerticalViewSize != 100 {
			t.Fatalf("invalid viewport leaked metrics: %+v", n)
		}
		if tree.SetScrollPercent(n.ID, 0, 50) || v.calls != before {
			t.Fatal("invalid axis partially changed viewport")
		}
	}
	v.info.ViewportSize.Height = 200
	v.info.ContentSize.Height = float32(math.NaN())
	n, _ = tree.Node(v)
	if n.VerticalScrollPercent != -1 || n.VerticalViewSize != 100 {
		t.Fatal("nonfinite content extent leaked")
	}
}
