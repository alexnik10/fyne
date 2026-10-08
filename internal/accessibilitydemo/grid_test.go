package accessibilitydemo

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/internal/driver"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGridKeyboardAndModelFocus(t *testing.T) {
	test.NewTempApp(t)
	status := widget.NewLabel("Ready")
	content := NewGrid(status).(*fyne.Container)
	g := content.Objects[1].(*checkGrid)
	swap := content.Objects[2].(*fyne.Container).Objects[0].(*widget.Button)
	before, after := widget.NewButton("Before", nil), widget.NewButton("After", nil)
	footer := container.NewVBox(after, status)
	w := test.NewWindow(container.New(layout.NewBorderLayout(before, footer, nil, nil), before, content, footer))
	defer w.Close()
	w.Resize(fyne.NewSize(550, 360))
	canvas := w.Canvas()
	tree := test.NewAccessibilityTree(canvas)
	canvas.Focus(g)
	first := focusedCheckbox(t, tree, "Enable Record 000")
	assert.False(t, first.Checked)
	canvas.FocusNext()
	require.Same(t, swap, canvas.Focused(), "Tab must skip all pooled renderer checks")
	canvas.FocusPrevious()
	require.Same(t, g, canvas.Focused())
	focusedCheckbox(t, tree, "Enable Record 000")

	g.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	focusedCheckbox(t, tree, "Enable Record 001")
	g.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnd})
	last := focusedCheckbox(t, tree, "Enable Record 499")
	require.Greater(t, g.GetScrollOffset(), float32(0))
	require.Greater(t, last.BoundsSize.Height, float32(0))
	previousStatus, ok := tree.Node(status)
	require.True(t, ok)
	require.Equal(t, fyne.AccessibilityLivePolite, previousStatus.LiveSetting)

	for _, key := range []fyne.KeyName{fyne.KeySpace, fyne.KeyReturn, fyne.KeyEnter} {
		wasChecked := g.enabled["Record 499"]
		g.TypedKey(&fyne.KeyEvent{Name: key})
		current := focusedCheckbox(t, tree, "Enable Record 499")
		require.Equal(t, last.ID, current.ID)
		assert.Equal(t, !wasChecked, current.Checked)
		assert.Same(t, g, canvas.Focused())
		currentStatus, exists := tree.Node(status)
		require.True(t, exists)
		assert.Equal(t, previousStatus.ID, currentStatus.ID)
		assert.Greater(t, currentStatus.LiveRevision, previousStatus.LiveRevision)
		assert.Contains(t, currentStatus.Name, "Record 499 enabled:")
		previousStatus = currentStatus
	}

	// Invoke need not move focus; the same model checkbox survives reordering.
	swap.AccessibilityActivate()
	current := focusedCheckbox(t, tree, "Enable Record 499")
	require.Equal(t, last.ID, current.ID)
	require.Equal(t, "Record 499", g.keys[0])
	require.Greater(t, current.BoundsSize.Height, float32(0))
	require.True(t, tree.Perform(last.ID, test.AccessibilityToggle, "", 0))
	assert.False(t, g.enabled["Record 499"])
	assert.False(t, g.enabled["Record 000"])
	focusedCheckbox(t, tree, "Enable Record 499")

	canvas.FocusNext()
	require.Same(t, swap, canvas.Focused())
	canvas.FocusNext()
	require.Same(t, after, canvas.Focused(), "leave the demo section without visiting renderer cells")
	canvas.FocusPrevious()
	require.Same(t, swap, canvas.Focused())
	swap.AccessibilityActivate()
	require.Same(t, swap, canvas.Focused(), "reordering from the button must not steal focus")
	canvas.FocusPrevious()
	require.Same(t, g, canvas.Focused())
	assert.Equal(t, last.ID, focusedCheckbox(t, tree, "Enable Record 499").ID)
}

func TestGridPointerAndOffscreenFocus(t *testing.T) {
	test.NewTempApp(t)
	status := widget.NewLabel("Ready")
	content := NewGrid(status).(*fyne.Container)
	g := content.Objects[1].(*checkGrid)
	w := test.NewWindow(container.NewBorder(nil, status, nil, nil, content))
	defer w.Close()
	w.Resize(fyne.NewSize(500, 300))
	tree := test.NewAccessibilityTree(w.Canvas())
	last, ok := tree.Element(g, widget.CollectionControlKey("Record 499", "enabled"))
	require.True(t, ok)
	require.True(t, last.Focusable)
	require.True(t, tree.Perform(last.ID, test.AccessibilityFocus, "", 0))
	focusedCheckbox(t, tree, "Enable Record 499")

	var rendered *gridCheck
	driver.WalkVisibleObjectTree(g, func(object fyne.CanvasObject, _, _ fyne.Position, _ fyne.Size) bool {
		check, isCheck := object.(*gridCheck)
		if isCheck && check.key == "Record 499" {
			rendered = check
			return true
		}
		return false
	}, nil)
	require.NotNil(t, rendered)
	test.Tap(rendered)
	require.Same(t, g, w.Canvas().Focused(), "mouse activation must not focus a recycled cell")
	current := focusedCheckbox(t, tree, "Enable Record 499")
	assert.Equal(t, last.ID, current.ID)
	assert.True(t, current.Checked)
	require.True(t, tree.Perform(last.ID, test.AccessibilityToggle, "", 0))
	assert.False(t, focusedCheckbox(t, tree, "Enable Record 499").Checked)
	assert.Equal(t, "Record 499 enabled: false", status.Text)
	assert.Empty(t, tree.Issues(), "canvas focus must always resolve to a semantic control")

	// Remove and replace a record: old providers must not act on its replacement.
	g.keys = g.keys[:len(g.keys)-1]
	g.Refresh()
	require.False(t, tree.Perform(last.ID, test.AccessibilityToggle, "", 0))
	g.keys = append(g.keys, "Record 499")
	g.Refresh()
	require.False(t, tree.Perform(last.ID, test.AccessibilityFocus, "", 0))
}

func focusedCheckbox(t *testing.T, tree *test.AccessibilityTree, name string) test.AccessibilityNode {
	t.Helper()
	var focused []test.AccessibilityNode
	for _, node := range tree.Snapshot() {
		if node.Focused {
			focused = append(focused, node)
		}
	}
	require.Len(t, focused, 1)
	node := focused[0]
	require.Equal(t, name, node.Name)
	require.Equal(t, fyne.AccessibleRoleCheck, node.Role)
	require.True(t, node.Focusable && node.Toggle)
	return node
}
