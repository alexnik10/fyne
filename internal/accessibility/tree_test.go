package accessibility_test

import (
	"math"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/internal/accessibility"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func nodeNamed(t *testing.T, nodes []accessibility.Node, name string) accessibility.Node {
	t.Helper()
	for _, n := range nodes {
		if n.Name == name {
			return n
		}
	}
	t.Fatalf("node %q missing", name)
	return accessibility.Node{}
}

func TestIdentityAndStructure(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	button := widget.NewButton("Save", nil)
	label := widget.NewLabel("Status")
	nested := container.NewWithoutLayout(button, label)
	nested.Move(fyne.NewPos(10, 20))
	button.Move(fyne.NewPos(3, 4))
	roots := []accessibility.Root{{Object: nested}}
	var tree accessibility.Tree
	first := tree.Build(roots, button)
	save := nodeNamed(t, first, "Save")
	assert.Equal(t, fyne.NewPos(13, 24), save.Position)
	assert.Equal(t, first[0].ID, save.Parent)
	assert.True(t, save.Focused)
	assert.False(t, nodeNamed(t, first, "Status").Focusable)

	button.SetText("Saved")
	nested.Objects = []fyne.CanvasObject{label, button}
	next := tree.Build(roots, button)
	assert.Equal(t, save.ID, nodeNamed(t, next, "Saved").ID)
	assert.Equal(t, "Status", next[1].Name)
	assert.Equal(t, save.ID, tree.FocusedID(button))

	nested.Hide()
	assert.Empty(t, tree.Build(roots, button))
	assert.False(t, tree.Perform(save.ID, accessibility.Activate, "", 0, nil))
	nested.Show()
	assert.Equal(t, save.ID, nodeNamed(t, tree.Build(roots, nil), "Saved").ID)

	nested.Objects = []fyne.CanvasObject{label}
	tree.Build(roots, nil)
	assert.False(t, tree.Perform(save.ID, accessibility.Activate, "", 0, nil))
	replacement := widget.NewButton("Saved", nil)
	nested.Objects = append(nested.Objects, replacement)
	assert.NotEqual(t, save.ID, nodeNamed(t, tree.Build(roots, nil), "Saved").ID)
}

func TestActionsAndProtectedValues(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	calls := 0
	button := widget.NewButton("Save", func() { calls++ })
	check := widget.NewCheck("Remember", func(bool) { calls++ })
	entry := widget.NewPasswordEntry()
	entry.SetPlaceHolder("Password")
	entry.SetText("secret")
	slider := widget.NewSlider(0, 10)
	slider.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Volume"})
	root := container.NewVBox(button, check, entry, slider)
	var tree accessibility.Tree
	roots := []accessibility.Root{{Object: root}}
	nodes := tree.Build(roots, nil)
	b, c, e, s := nodeNamed(t, nodes, "Save"), nodeNamed(t, nodes, "Remember"), nodeNamed(t, nodes, "Password"), nodeNamed(t, nodes, "Volume")
	require.True(t, tree.Perform(b.ID, accessibility.Activate, "", 0, nil))
	require.True(t, tree.Perform(c.ID, accessibility.Toggle, "", 0, nil))
	assert.True(t, check.Checked)
	assert.Equal(t, 2, calls)
	assert.True(t, e.Protected)
	assert.Empty(t, e.Text)
	assert.True(t, tree.Perform(e.ID, accessibility.SetValue, "new secret", 0, nil))
	assert.Equal(t, "new secret", entry.Text)
	assert.True(t, tree.Perform(s.ID, accessibility.SetRangeValue, "", 6, nil))
	assert.Equal(t, 6.0, slider.Value)
	for _, invalid := range []float64{-1, 11, math.NaN(), math.Inf(1)} {
		assert.False(t, tree.Perform(s.ID, accessibility.SetRangeValue, "", invalid, nil))
	}
	button.Disable()
	check.Disable()
	entry.Disable()
	slider.Disable()
	// Even a command queued before the disabling is rejected at execution time.
	assert.False(t, tree.Perform(b.ID, accessibility.Activate, "", 0, nil))
	assert.False(t, tree.Perform(c.ID, accessibility.Toggle, "", 0, nil))
	assert.False(t, tree.Perform(e.ID, accessibility.SetValue, "overwrite", 0, nil))
	assert.False(t, tree.Perform(s.ID, accessibility.SetRangeValue, "", 5, nil))
	assert.Equal(t, 2, calls)
}

func TestOverlayScopeAndFocus(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	button := widget.NewButton("Background", nil)
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Dialog field")
	win := test.NewWindow(container.NewVBox(button))
	defer win.Close()
	win.Canvas().Focus(button)
	popup := widget.NewModalPopUp(entry, win.Canvas())
	popup.Show()
	defer popup.Hide()
	var tree accessibility.Tree
	roots := []accessibility.Root{{Object: win.Content()}, {Object: win.Canvas().Overlays().Top(), Suppressed: true}}
	backgroundID := nodeNamed(t, tree.Build(roots, nil), "Background").ID
	roots[0].Suppressed, roots[1].Suppressed = true, false
	nodes := tree.Build(roots, nil)
	field := nodeNamed(t, nodes, "Dialog field")
	assert.False(t, tree.Perform(backgroundID, accessibility.Focus, "", 0, win.Canvas()))
	assert.True(t, tree.Perform(field.ID, accessibility.Focus, "", 0, win.Canvas()))
	assert.Same(t, entry, win.Canvas().Focused())
	popup.Hide()
	assert.Same(t, button, win.Canvas().Focused())
	assert.Equal(t, backgroundID, nodeNamed(t, tree.Build([]accessibility.Root{{Object: win.Content()}}, button), "Background").ID)
}

func TestFormLabelsAndExplicitNames(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	entry := widget.NewEntry()
	entry.SetPlaceHolder("placeholder")
	item := widget.NewFormItem("Email", entry)
	item.HintText = "Work address"
	item.Required = true
	form := widget.NewForm(item)
	form.OnSubmit = func() {}
	win := test.NewWindow(form)
	defer win.Close()
	form.Refresh()
	var tree accessibility.Tree
	roots := []accessibility.Root{{Object: form}}
	node := nodeNamed(t, tree.Build(roots, nil), "Email")
	assert.Equal(t, "Work address", node.Description)
	assert.True(t, node.Required)
	entry.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Personal email"})
	explicit := nodeNamed(t, tree.Build(roots, nil), "Personal email")
	assert.Equal(t, node.ID, explicit.ID)
	assert.Equal(t, "Work address", explicit.Description)
}

type composite struct {
	widget.BaseWidget
	children []fyne.CanvasObject
}

func (c *composite) AccessibilityChildren() []fyne.CanvasObject { return c.children }
func (c *composite) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewWithoutLayout(c.children...))
}

func TestSemanticChildrenAndCycles(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	child := widget.NewLabel("Child")
	c := &composite{children: []fyne.CanvasObject{child}}
	c.ExtendBaseWidget(c)
	c.children = append(c.children, c, child) // malformed graph must not recurse forever
	var tree accessibility.Tree
	nodes := tree.Build([]accessibility.Root{{Object: c}}, nil)
	require.Len(t, nodes, 1)
	assert.Equal(t, "Child", nodes[0].Name)
}

func TestKeyboardAndAccessibleActivationPreserveFocus(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	calls := 0
	button := widget.NewButton("Run", func() { calls++ })
	win := test.NewWindow(button)
	defer win.Close()
	win.Canvas().Focus(button)
	button.TypedKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	assert.Same(t, button, win.Canvas().Focused())
	button.AccessibilityActivate()
	assert.Same(t, button, win.Canvas().Focused())
	assert.Equal(t, 2, calls)
	button.Disable()
	button.AccessibilityActivate()
	assert.Equal(t, 2, calls)
}
