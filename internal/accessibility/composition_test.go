package accessibility_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/internal/accessibility"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dormantWidget struct{ widget.BaseWidget }

func (*dormantWidget) CreateRenderer() fyne.WidgetRenderer { panic("hidden renderer constructed") }

func TestHiddenWidgetDoesNotConstructRenderer(t *testing.T) {
	obj := &dormantWidget{}
	obj.Hidden = true
	var tree accessibility.Tree
	assert.Empty(t, tree.Build([]accessibility.Root{{Object: obj}}, nil))
}

func TestCompositionClipsBoundsWithoutMovingTextOrigin(t *testing.T) {
	a := test.NewTempApp(t)
	defer a.Quit()
	entry := widget.NewEntry()
	entry.SetText("text")
	content := container.NewWithoutLayout(entry)
	clip := container.NewClip(content)
	clip.Move(fyne.NewPos(20, 30))
	clip.Resize(fyne.NewSize(100, 80))
	entry.Move(fyne.NewPos(75, 60))
	entry.Resize(fyne.NewSize(100, 40))
	var tree accessibility.Tree
	tree.Build([]accessibility.Root{{Object: clip}}, nil)
	n, ok := tree.NodeForObject(entry)
	require.True(t, ok)
	assert.Equal(t, fyne.NewPos(95, 90), n.Position)
	assert.Equal(t, fyne.NewPos(95, 90), n.BoundsPosition)
	assert.Equal(t, fyne.NewSize(25, 20), n.BoundsSize)
	assert.LessOrEqual(t, n.Document.ViewportSize.Width, float32(25))
	assert.LessOrEqual(t, n.Document.ViewportSize.Height, float32(20))
	entry.Move(fyne.NewPos(120, 90))
	tree.Build([]accessibility.Root{{Object: clip}}, nil)
	n2, _ := tree.NodeForObject(entry)
	assert.Equal(t, n.ID, n2.ID)
	assert.Equal(t, fyne.NewSize(0, 0), n2.BoundsSize)
	assert.NotNil(t, n2.Document, "offscreen content remains semantically attached")
}

func TestVirtualizedWidgetsDoNotExposeRecycledRendererItems(t *testing.T) {
	test.NewTempApp(t)
	list := widget.NewList(func() int { return 100 }, func() fyne.CanvasObject {
		return widget.NewButton("Recycled visual button", nil)
	}, func(widget.ListItemID, fyne.CanvasObject) {})
	w := test.NewWindow(list)
	defer w.Close()
	var tree accessibility.Tree
	nodes := tree.Build([]accessibility.Root{{Object: list}}, nil)
	require.Len(t, nodes, 101)
	assert.Equal(t, fyne.AccessibleRoleList, nodes[0].Role)
	for _, item := range nodes[1:] {
		assert.Equal(t, fyne.AccessibleRoleListItem, item.Role)
		assert.NotEqual(t, "Recycled visual button", item.Name)
		assert.False(t, item.Invoke, "a recycled visual button is not the model item")
	}
}
