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

func TestRadioSelectionCommandsAndIdentity(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	calls := 0
	r := widget.NewRadioGroup([]string{"Daily", "Weekly", "Never"}, func(string) { calls++ })
	r.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Notifications"})
	r.Required = true
	r.SetSelected("Daily")
	w := test.NewWindow(r)
	defer w.Close()
	var tree accessibility.Tree
	roots := []accessibility.Root{{Object: r}}
	nodes := tree.Build(roots, nil)
	group, daily, weekly := nodeNamed(t, nodes, "Notifications"), nodeNamed(t, nodes, "Daily"), nodeNamed(t, nodes, "Weekly")
	assert.True(t, group.SelectionRequired)
	assert.Equal(t, group.ID, daily.SelectionOwner)
	assert.True(t, daily.Selected)
	assert.Equal(t, 2, weekly.SetPosition)
	assert.Equal(t, 3, weekly.SetSize)
	assert.False(t, tree.Perform(daily.ID, accessibility.RemoveFromSelection, "", 0, w.Canvas()))
	assert.False(t, tree.Perform(weekly.ID, accessibility.AddToSelection, "", 0, w.Canvas()))
	require.True(t, tree.Perform(weekly.ID, accessibility.Select, "", 0, w.Canvas()))
	assert.Equal(t, "Weekly", r.Selected)
	tree.Build(roots, nil)
	require.True(t, tree.Perform(weekly.ID, accessibility.Select, "", 0, w.Canvas()))
	assert.Equal(t, 2, calls, "selecting an already selected radio does not toggle or notify")
	require.True(t, tree.Perform(weekly.ID, accessibility.Focus, "", 0, w.Canvas()))
	w.Canvas().Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	assert.Equal(t, "Never", r.Selected)
	nodes = tree.Build(roots, w.Canvas().Focused())
	assert.True(t, nodeNamed(t, nodes, "Never").Focused)

	r.Options = []string{"Weekly", "Daily", "New"}
	r.Refresh()
	nodes = tree.Build(roots, nil)
	assert.Equal(t, weekly.ID, nodeNamed(t, nodes, "Weekly").ID)
	assert.Equal(t, daily.ID, nodeNamed(t, nodes, "Daily").ID)
	r.Options = []string{"Weekly", "New"}
	r.Refresh()
	tree.Build(roots, nil)
	assert.False(t, tree.Perform(daily.ID, accessibility.Select, "", 0, nil))
	r.Disable()
	nodes = tree.Build(roots, nil)
	assert.True(t, nodeNamed(t, nodes, "Weekly").Disabled)
	assert.False(t, tree.Perform(weekly.ID, accessibility.Select, "", 0, nil))
	assert.False(t, tree.Perform(weekly.ID, accessibility.Focus, "", 0, w.Canvas()))
}

func TestSelectPopupScopeFocusAndSelection(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	calls := 0
	s := widget.NewSelect([]string{"English", "Russian", "German"}, func(string) { calls++ })
	s.SetSelected("Russian")
	button := widget.NewButton("Background", nil)
	form := widget.NewForm(widget.NewFormItem("Language", s))
	root := container.NewVBox(form, button)
	w := test.NewWindow(root)
	defer w.Close()
	w.Resize(fyne.NewSize(420, 320))
	w.Canvas().Focus(s)
	var tree accessibility.Tree
	roots := []accessibility.Root{{Object: root}}
	nodes := tree.Build(roots, s)
	combo, russian, german := nodeNamed(t, nodes, "Language"), nodeNamed(t, nodes, "Russian"), nodeNamed(t, nodes, "German")
	background := nodeNamed(t, nodes, "Background")
	assert.True(t, combo.Expandable)
	assert.True(t, combo.Selection)
	assert.False(t, combo.Expanded)
	assert.Equal(t, combo.ID, russian.SelectionOwner)
	assert.True(t, russian.Selected)
	assert.Zero(t, russian.Size)
	assert.False(t, russian.Focusable)
	assert.False(t, tree.Perform(german.ID, accessibility.Focus, "", 0, w.Canvas()))

	s.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	var scope accessibility.FocusScope
	scope.Update(w.Canvas())
	require.True(t, s.AccessibilityExpanded())
	owner := w.Canvas().Overlays().Top().(fyne.AccessibleOverlayOwner).AccessibilityOverlayOwner()
	require.Same(t, s, owner)
	scoped := []accessibility.Root{{Object: root, Scope: owner}, {Object: w.Canvas().Overlays().Top(), Scope: owner}}
	nodes = tree.Build(scoped, w.Canvas().Focused())
	require.Len(t, nodes, 4, "only combo box and its three choices are exposed")
	assert.Equal(t, combo.ID, nodeNamed(t, nodes, "Language").ID)
	assert.True(t, nodes[0].Expanded)
	assert.True(t, nodeNamed(t, nodes, "Russian").Focused)
	assert.Positive(t, nodeNamed(t, nodes, "Russian").Size.Height)
	assert.False(t, tree.Perform(background.ID, accessibility.Activate, "", 0, w.Canvas()))
	w.Canvas().Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	nodes = tree.Build(scoped, w.Canvas().Focused())
	assert.True(t, nodeNamed(t, nodes, "German").Focused)
	assert.Equal(t, "Russian", s.Selected, "arrows preview; Enter commits")
	w.Canvas().Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	assert.False(t, s.AccessibilityExpanded())
	assert.Equal(t, "Russian", s.Selected)
	assert.Same(t, s, w.Canvas().Focused())
	nodes = tree.Build(roots, s)
	assert.Equal(t, russian.ID, nodeNamed(t, nodes, "Russian").ID)
	require.True(t, tree.Perform(combo.ID, accessibility.Expand, "", 0, w.Canvas()))
	tree.Build(scoped, w.Canvas().Focused())
	require.True(t, tree.Perform(german.ID, accessibility.Select, "", 0, w.Canvas()))
	assert.Equal(t, "German", s.Selected)
	assert.Equal(t, 2, calls)
	assert.False(t, s.AccessibilityExpanded())
	assert.Same(t, s, w.Canvas().Focused())

	s.SetOptions([]string{"German", "English"})
	nodes = tree.Build(roots, s)
	assert.Equal(t, german.ID, nodeNamed(t, nodes, "German").ID)
	assert.False(t, tree.Perform(russian.ID, accessibility.Select, "", 0, nil))
	s.Disable()
	tree.Build(roots, nil)
	assert.False(t, tree.Perform(combo.ID, accessibility.Expand, "", 0, w.Canvas()))
	assert.False(t, tree.Perform(german.ID, accessibility.Select, "", 0, w.Canvas()))
	s.Enable()
	s.AccessibilitySetExpanded(true)
	require.True(t, s.AccessibilityExpanded())
	s.Disable()
	assert.False(t, s.AccessibilityExpanded(), "disabling also dismisses keyboard input in the popup")
	assert.Same(t, s, w.Canvas().Focused())
}

func TestSelectionOptionalAndHiddenScope(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	r := widget.NewRadioGroup([]string{"A", "B"}, nil)
	r.SetSelected("A")
	w := test.NewWindow(r)
	defer w.Close()
	var tree accessibility.Tree
	roots := []accessibility.Root{{Object: r}}
	n := nodeNamed(t, tree.Build(roots, nil), "A")
	require.True(t, tree.Perform(n.ID, accessibility.RemoveFromSelection, "", 0, w.Canvas()))
	assert.Empty(t, r.Selected)
	parent := container.NewVBox(r)
	parent.Hide()
	assert.Empty(t, tree.Build([]accessibility.Root{{Object: parent, Scope: r}}, nil))
	assert.False(t, tree.Perform(n.ID, accessibility.Select, "", 0, w.Canvas()))
}
