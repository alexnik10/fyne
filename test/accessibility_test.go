package test_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This component deliberately implements no accessibility interfaces.
type filePicker struct {
	widget.BaseWidget
	content fyne.CanvasObject
}

func newFilePicker(content fyne.CanvasObject) *filePicker {
	p := &filePicker{content: content}
	p.ExtendBaseWidget(p)
	return p
}

func (p *filePicker) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(p.content)
}

func TestAccessibilityCompositionKeepsOriginalCommandsAndIdentity(t *testing.T) {
	a := test.NewTempApp(t)
	entry := widget.NewEntry()
	entry.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Path"})
	calls := 0
	button := widget.NewButton("Browse", func() { calls++ })
	row := container.NewHBox(entry, button)
	outer := newFilePicker(newFilePicker(row))
	w := a.NewWindow("")
	defer w.Close()
	w.SetContent(outer)
	inspector := test.NewAccessibilityTree(w.Canvas())
	b, ok := inspector.Node(button)
	require.True(t, ok)
	assert.True(t, b.Invoke)
	e, ok := inspector.Node(entry)
	require.True(t, ok)
	assert.Equal(t, "Path", e.Name)
	_, exists := inspector.Node(outer)
	assert.False(t, exists)
	require.True(t, inspector.Perform(b.ID, test.AccessibilityActivate, "", 0))
	require.True(t, inspector.Perform(e.ID, test.AccessibilitySetValue, "report.txt", 0))
	assert.Equal(t, 1, calls)
	assert.Equal(t, "report.txt", entry.Text)

	group := container.NewAccessibilityGroup("Choose file", outer)
	w.SetContent(group)
	g, ok := inspector.Node(group)
	require.True(t, ok)
	assert.Equal(t, "Choose file", g.Name)
	assert.False(t, g.Invoke)
	row.Objects = []fyne.CanvasObject{button, entry}
	row.Refresh()
	b2, _ := inspector.Node(button)
	e2, _ := inspector.Node(entry)
	assert.Equal(t, b.ID, b2.ID)
	assert.Equal(t, e.ID, e2.ID)

	group.SetAccessibilityMode(fyne.AccessibilityExclude)
	assert.Empty(t, inspector.Snapshot())
	assert.False(t, inspector.Perform(b.ID, test.AccessibilityActivate, "", 0))
	group.SetAccessibilityMode(fyne.AccessibilityTransparent)
	b2, _ = inspector.Node(button)
	assert.Equal(t, b.ID, b2.ID)
	_, exists = inspector.Node(group)
	assert.False(t, exists)

	row.Remove(button)
	assert.False(t, inspector.Perform(b.ID, test.AccessibilityActivate, "", 0))
	replacement := widget.NewButton("Browse", nil)
	row.Add(replacement)
	b2, _ = inspector.Node(replacement)
	assert.NotEqual(t, b.ID, b2.ID)
}

type semanticButton struct {
	*filePicker
	calls int
}

func (*semanticButton) AccessibilityLabel() string             { return "Whole action" }
func (*semanticButton) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleButton }
func (b *semanticButton) AccessibilityActivate()               { b.calls++ }

type explicitPicker struct {
	*filePicker
	children []fyne.CanvasObject
}

func (p *explicitPicker) AccessibilityChildren() []fyne.CanvasObject { return p.children }

func TestAccessibilityBoundariesAndExplicitEmptyChildren(t *testing.T) {
	a := test.NewTempApp(t)
	decoration := widget.NewLabel("Internal rendering label")
	inner := widget.NewButton("Independent", nil)
	b := &semanticButton{filePicker: newFilePicker(container.NewVBox(decoration, inner))}
	w := a.NewWindow("")
	defer w.Close()
	w.SetContent(b)
	inspector := test.NewAccessibilityTree(w.Canvas())
	nodes := inspector.Snapshot()
	require.Len(t, nodes, 1)
	assert.Equal(t, "Whole action", nodes[0].Name)
	require.True(t, inspector.Perform(nodes[0].ID, test.AccessibilityActivate, "", 0))
	assert.Equal(t, 1, b.calls)
	b.SetAccessibilityMode(fyne.AccessibilityMode(255))
	require.Len(t, inspector.Snapshot(), 1, "unknown modes preserve the Auto boundary")
	b.SetAccessibilityMode(fyne.AccessibilityGroup)
	_, ok := inspector.Node(inner)
	require.True(t, ok)
	b.SetAccessibilityMode(fyne.AccessibilitySingle)
	require.Len(t, inspector.Snapshot(), 1)

	p := &explicitPicker{filePicker: newFilePicker(container.NewVBox(decoration, inner))}
	w.SetContent(p)
	assert.Empty(t, inspector.Snapshot(), "nil explicit children must not fall back to the renderer")
	p.children = []fyne.CanvasObject{inner, inner, p}
	nodes = inspector.Snapshot()
	require.Len(t, nodes, 1, "duplicates and cycles are ignored")
	assert.Equal(t, "Independent", nodes[0].Name)
	p.SetAccessibilityMode(fyne.AccessibilitySingle)
	assert.Empty(t, inspector.Snapshot(), "Single also stops explicit children")
	p.SetAccessibilityMode(fyne.AccessibilityGroup)
	p.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Actions"})
	nodes = inspector.Snapshot()
	require.Len(t, nodes, 2)
	assert.Equal(t, "Actions", nodes[0].Name)
	assert.Equal(t, nodes[0].ID, nodes[1].Parent)
}

func TestAccessibilityCommandsRevalidateScopeAndState(t *testing.T) {
	a := test.NewTempApp(t)
	button := widget.NewButton("Background", nil)
	w := a.NewWindow("")
	defer w.Close()
	w.SetContent(newFilePicker(button))
	inspector := test.NewAccessibilityTree(w.Canvas())
	b, ok := inspector.Node(button)
	require.True(t, ok)
	button.Disable()
	assert.False(t, inspector.Perform(b.ID, test.AccessibilityActivate, "", 0))
	button.Enable()
	button.Hide()
	assert.False(t, inspector.Perform(b.ID, test.AccessibilityActivate, "", 0))
	button.Show()
	field := widget.NewEntry()
	popup := widget.NewModalPopUp(newFilePicker(field), w.Canvas())
	popup.Show()
	assert.False(t, inspector.Perform(b.ID, test.AccessibilityActivate, "", 0))
	e, ok := inspector.Node(field)
	require.True(t, ok)
	require.True(t, inspector.Perform(e.ID, test.AccessibilityFocus, "", 0))
	assert.Same(t, field, w.Canvas().Focused())
	popup.Hide()
	b2, _ := inspector.Node(button)
	assert.Equal(t, b.ID, b2.ID)
	assert.False(t, inspector.Perform(e.ID, test.AccessibilitySetValue, "stale", 0))
}

func TestAccessibilityWrapperContentReplacement(t *testing.T) {
	a := test.NewTempApp(t)
	w := a.NewWindow("")
	defer w.Close()
	wrapper := container.NewAccessibilityGroup("Items", nil)
	w.SetContent(wrapper)
	inspector := test.NewAccessibilityTree(w.Canvas())
	require.Len(t, inspector.Snapshot(), 1)
	first := widget.NewButton("First", nil)
	wrapper.Content = first
	wrapper.Refresh()
	old, ok := inspector.Node(first)
	require.True(t, ok)
	second := widget.NewButton("Second", nil)
	wrapper.Content = second
	wrapper.Refresh()
	_, ok = inspector.Node(second)
	require.True(t, ok)
	assert.False(t, inspector.Perform(old.ID, test.AccessibilityActivate, "", 0))
	w.Canvas().FocusNext()
	assert.Same(t, second, w.Canvas().Focused(), "visual and semantic children must agree")
}

func TestAccessibilityReportsMetadataAndFocusWithoutSemantics(t *testing.T) {
	test.NewTempApp(t)
	button := widget.NewButton("Action", nil)
	picker := newFilePicker(button)
	picker.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Unused name"})
	w := test.NewWindow(picker)
	defer w.Close()
	inspector := test.NewAccessibilityTree(w.Canvas())
	issues := inspector.Issues()
	require.Len(t, issues, 1)
	assert.Equal(t, "unrepresented-metadata", issues[0].Code)
	picker.SetAccessibilityMode(fyne.AccessibilityGroup)
	assert.Empty(t, inspector.Issues())
	w.Canvas().Focus(button)
	picker.SetAccessibilityMode(fyne.AccessibilityExclude)
	issues = inspector.Issues()
	require.Len(t, issues, 1)
	assert.Equal(t, "unrepresented-focus", issues[0].Code)
}
