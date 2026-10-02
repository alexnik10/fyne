package test_test

import (
	"errors"
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type keyedControl struct {
	widget.BaseWidget
	keys    []string
	hidden  map[string]bool
	version int
	calls   []string
}

func (c *keyedControl) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(widget.NewLabel("Decoration"))
}
func (c *keyedControl) AccessibilityElements() []fyne.AccessibilityElement {
	var out []fyne.AccessibilityElement
	for _, key := range c.keys {
		key, version := key, c.version
		// Deliberately replace all objects, including command closures, each time.
		b := widget.NewButton(key, func() { c.calls = append(c.calls, fmt.Sprintf("%s:%d", key, version)) })
		out = append(out, fyne.AccessibilityElement{Key: key, Object: b, Hidden: c.hidden[key]})
	}
	return out
}
func newKeyedControl() *keyedControl {
	c := &keyedControl{keys: []string{"a", "b"}, hidden: make(map[string]bool)}
	c.ExtendBaseWidget(c)
	c.SetAccessibilityMode(fyne.AccessibilityGroup)
	return c
}

func TestAccessibilityKeyedIdentityAndFreshCommands(t *testing.T) {
	test.NewTempApp(t)
	c, other := newKeyedControl(), newKeyedControl()
	w := test.NewWindow(container.NewVBox(c, other))
	defer w.Close()
	s := test.NewAccessibilityTree(w.Canvas())
	a, ok := s.Element(c, "a")
	require.True(t, ok)
	b, _ := s.Element(c, "b")
	otherA, _ := s.Element(other, "a")
	assert.NotEqual(t, a.ID, otherA.ID)
	c.keys, c.version = []string{"b", "a"}, 2
	a2, _ := s.Element(c, "a")
	b2, _ := s.Element(c, "b")
	assert.Equal(t, a.ID, a2.ID)
	assert.Equal(t, b.ID, b2.ID)
	require.True(t, s.Perform(a.ID, test.AccessibilityActivate, "", 0))
	assert.Equal(t, []string{"a:2"}, c.calls)
	c.hidden["a"] = true
	assert.False(t, s.Perform(a.ID, test.AccessibilityActivate, "", 0))
	c.hidden["a"] = false
	a2, _ = s.Element(c, "a")
	assert.Equal(t, a.ID, a2.ID)
	c.Hide()
	assert.False(t, s.Perform(a.ID, test.AccessibilityActivate, "", 0))
	c.Show()
	a2, _ = s.Element(c, "a")
	assert.Equal(t, a.ID, a2.ID)
	c.keys = []string{"b"}
	assert.False(t, s.Perform(a.ID, test.AccessibilityActivate, "", 0))
	c.keys = []string{"a", "b"}
	a2, _ = s.Element(c, "a")
	assert.NotEqual(t, a.ID, a2.ID)
	assert.False(t, s.Perform(a.ID, test.AccessibilityActivate, "", 0))
	p := widget.NewModalPopUp(widget.NewButton("Close", nil), w.Canvas())
	p.Show()
	assert.False(t, s.Perform(a2.ID, test.AccessibilityActivate, "", 0))
	p.Hide()
	a3, _ := s.Element(c, "a")
	assert.Equal(t, a2.ID, a3.ID)
}

func TestAccessibilityMetadataExplicitZeroAndReset(t *testing.T) {
	test.NewTempApp(t)
	e := widget.NewEntry()
	e.SetPlaceHolder("Fallback")
	e.Validator = func(string) error { return errors.New("Current error") }
	e.AlwaysShowValidationError = true
	item := widget.NewFormItem("Inherited name", e)
	item.HintText, item.Required = "Inherited hint", true
	f := widget.NewForm(item)
	w := test.NewWindow(f)
	defer w.Close()
	s := test.NewAccessibilityTree(w.Canvas())
	e.Validate()
	n, _ := s.Node(e)
	assert.True(t, n.Invalid)
	assert.True(t, n.Required)
	assert.Equal(t, "Current error", n.Description)
	e.SetAccessibilityInfo(fyne.AccessibilityInfo{NameSet: true, DescriptionSet: true, RequiredSet: true, InvalidSet: true})
	n, _ = s.Node(e)
	assert.Empty(t, n.Name)
	assert.Empty(t, n.Description)
	assert.False(t, n.Required)
	assert.False(t, n.Invalid)
	e.SetAccessibilityInfo(fyne.AccessibilityInfo{})
	n, _ = s.Node(e)
	assert.Equal(t, "Inherited name", n.Name)
	assert.True(t, n.Required)
	assert.True(t, n.Invalid)
	assert.Equal(t, "Current error", n.Description)
	e.SetAccessibilityInfo(fyne.AccessibilityInfo{Description: "Explicit error guidance"})
	n, _ = s.Node(e)
	assert.Equal(t, "Explicit error guidance", n.Description)
	w.SetContent(e)
	e.SetAccessibilityInfo(fyne.AccessibilityInfo{})
	n, _ = s.Node(e)
	assert.Equal(t, "Fallback", n.Name)
}

func TestAccessibilityTreeHierarchyFocusAndLifetime(t *testing.T) {
	test.NewTempApp(t)
	data := map[string][]string{"": {"folder", "last"}, "folder": {"first", "second"}}
	tree := widget.NewTreeWithStrings(data)
	tree.DescribeNode = func(id string) fyne.AccessibilityInfo { return fyne.AccessibilityInfo{Name: "Name " + id} }
	w := test.NewWindow(tree)
	defer w.Close()
	w.Resize(fyne.NewSize(240, 150))
	s := test.NewAccessibilityTree(w.Canvas())
	root, _ := s.Node(tree)
	assert.Equal(t, fyne.AccessibleRoleTree, root.Role)
	folder, ok := s.Element(tree, "folder")
	require.True(t, ok)
	assert.Equal(t, 1, folder.Level)
	assert.Equal(t, 1, folder.SetPosition)
	assert.Equal(t, 2, folder.SetSize)
	_, ok = s.Element(tree, "first")
	assert.False(t, ok)
	require.True(t, s.Perform(folder.ID, test.AccessibilityExpand, "", 0))
	first, ok := s.Element(tree, "first")
	require.True(t, ok)
	assert.Equal(t, "Name first", first.Name)
	assert.Equal(t, folder.ID, first.Parent)
	assert.Equal(t, root.ID, first.SelectionOwner)
	assert.Equal(t, 2, first.Level)
	assert.False(t, first.Expandable)
	assert.True(t, first.ScrollItem)
	require.True(t, s.Perform(first.ID, test.AccessibilityFocus, "", 0))
	assert.Same(t, tree, w.Canvas().Focused())
	first2, _ := s.Element(tree, "first")
	assert.True(t, first2.Focused)
	assert.False(t, first2.Selected, "focus must not select")
	w.Canvas().Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	second, _ := s.Element(tree, "second")
	assert.True(t, second.Focused)
	require.True(t, s.Perform(first.ID, test.AccessibilitySelect, "", 0))
	first2, _ = s.Element(tree, "first")
	assert.True(t, first2.Selected)
	assert.False(t, s.Perform(second.ID, test.AccessibilityAddToSelection, "", 0))
	require.True(t, s.Perform(first.ID, test.AccessibilityRemoveFromSelection, "", 0))
	require.True(t, s.Perform(folder.ID, test.AccessibilityCollapse, "", 0))
	folder2, _ := s.Element(tree, "folder")
	assert.True(t, folder2.Focused, "collapse reconciles actual keyboard highlight")
	assert.False(t, s.Perform(first.ID, test.AccessibilityFocus, "", 0))
	require.True(t, s.Perform(folder.ID, test.AccessibilityExpand, "", 0))
	first2, _ = s.Element(tree, "first")
	assert.Equal(t, first.ID, first2.ID)
	data["folder"] = []string{"second", "first"}
	tree.Refresh()
	first2, _ = s.Element(tree, "first")
	assert.Equal(t, first.ID, first2.ID)
	assert.Equal(t, 2, first2.SetPosition)
	tree.CloseBranch("folder")
	data["folder"] = []string{"second"}
	tree.Refresh()
	s.Snapshot() // deletion must be observed even inside a closed branch
	data["folder"] = []string{"first", "second"}
	tree.Refresh()
	tree.OpenBranch("folder")
	first2, _ = s.Element(tree, "first")
	assert.NotEqual(t, first.ID, first2.ID)
	assert.False(t, s.Perform(first.ID, test.AccessibilitySelect, "", 0))
	require.Empty(t, s.Issues())
}

func TestAccessibilityTreeOffscreenDoesNotMaterializeCells(t *testing.T) {
	test.NewTempApp(t)
	ids := make([]string, 1000)
	for i := range ids {
		ids[i] = fmt.Sprintf("node-%04d", i)
	}
	created, updated := 0, 0
	tree := widget.NewTree(func(id string) []string {
		if id == "" {
			return ids
		}
		return nil
	},
		func(id string) bool { return id == "" },
		func(bool) fyne.CanvasObject { created++; return widget.NewLabel("Template") },
		func(id string, _ bool, obj fyne.CanvasObject) { updated++; obj.(*widget.Label).SetText(id) })
	w := test.NewWindow(tree)
	defer w.Close()
	w.Resize(fyne.NewSize(220, 120))
	s := test.NewAccessibilityTree(w.Canvas())
	beforeCreated, beforeUpdated := created, updated
	last, ok := s.Element(tree, ids[len(ids)-1])
	require.True(t, ok)
	assert.Zero(t, last.BoundsSize.Height)
	assert.Equal(t, beforeCreated, created)
	assert.Equal(t, beforeUpdated, updated)
	assert.Less(t, created, 100)
	require.True(t, s.Perform(last.ID, test.AccessibilityScrollIntoView, "", 0))
	last2, _ := s.Element(tree, ids[len(ids)-1])
	assert.Equal(t, last.ID, last2.ID)
	assert.Positive(t, last2.BoundsSize.Height)
	assert.Nil(t, w.Canvas().Focused())
	assert.False(t, last2.Selected)
	require.True(t, s.Perform(last.ID, test.AccessibilityFocus, "", 0))
	last2, _ = s.Element(tree, ids[len(ids)-1])
	assert.True(t, last2.Focused)
	tree.ScrollToTop()
	last2, _ = s.Element(tree, ids[len(ids)-1])
	assert.Equal(t, last.ID, last2.ID)
	assert.Zero(t, last2.BoundsSize.Height)
}

func TestAccessibilityTreeCustomRootAndRemovalFocus(t *testing.T) {
	test.NewTempApp(t)
	data := map[string][]string{"root": {"a", "b"}}
	tree := widget.NewTreeWithStrings(data)
	tree.Root = "root"
	w := test.NewWindow(tree)
	defer w.Close()
	s := test.NewAccessibilityTree(w.Canvas())
	w.Canvas().Focus(tree)
	r, ok := s.Element(tree, "root")
	require.True(t, ok)
	assert.True(t, r.Focused)
	assert.True(t, r.Expanded)
	assert.False(t, s.Perform(r.ID, test.AccessibilityCollapse, "", 0))
	a, _ := s.Element(tree, "a")
	require.True(t, s.Perform(a.ID, test.AccessibilitySelect, "", 0))
	data["root"] = []string{"b"}
	tree.Refresh()
	r, _ = s.Element(tree, "root")
	assert.True(t, r.Focused)
	assert.False(t, s.Perform(a.ID, test.AccessibilitySelect, "", 0))
	assert.Empty(t, s.Issues())
}

func TestAccessibilityEmptyFormLabelKeepsControlName(t *testing.T) {
	test.NewTempApp(t)
	check := widget.NewCheck("Remember", nil)
	w := test.NewWindow(widget.NewForm(widget.NewFormItem("", check)))
	defer w.Close()
	n, ok := test.NewAccessibilityTree(w.Canvas()).Node(check)
	require.True(t, ok)
	assert.Equal(t, "Remember", n.Name)
}
