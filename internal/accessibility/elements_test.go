package accessibility_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/internal/accessibility"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type elementOwner struct {
	widget.DisableableWidget
	elements []fyne.AccessibilityElement
}

func (*elementOwner) AccessibilityLabel() string                           { return "Owner" }
func (*elementOwner) AccessibilityRole() fyne.AccessibleRole               { return fyne.AccessibleRoleContainer }
func (o *elementOwner) AccessibilityElements() []fyne.AccessibilityElement { return o.elements }
func (*elementOwner) CreateRenderer() fyne.WidgetRenderer {
	panic("semantic collection must not construct a renderer")
}

func TestKeyedElementsValidateHierarchyAndMembership(t *testing.T) {
	test.NewTempApp(t)
	button := widget.NewButton("Action", nil)
	var typedNil *widget.Button
	o := &elementOwner{elements: []fyne.AccessibilityElement{
		{Key: "parent", Object: button},
		{Key: "child", Parent: "parent", Object: button},
		{Key: "hidden", Hidden: true},
		{Key: "descendant", Parent: "hidden", Hidden: true},
		{Key: "child", Object: button}, // duplicate
		{Key: "bad-parent", Parent: "later", Object: button},
		{Key: "cycle", Parent: "cycle", Object: button},
		{Key: "nil", Object: typedNil},
		{Key: "", Object: button},
	}}
	var tree accessibility.Tree
	roots := []accessibility.Root{{Object: o}}
	nodes := tree.Build(roots, nil)
	require.Len(t, nodes, 3)
	assert.Equal(t, nodes[1].ID, nodes[2].Parent)
	assert.NotEqual(t, nodes[1].ID, nodes[2].ID, "one transient object can describe two independently keyed elements")
	require.Len(t, tree.Issues, 5)
	for _, issue := range tree.Issues {
		assert.Equal(t, "invalid-element", issue.Code)
	}
	o.elements = o.elements[:4]
	o.elements[2].Hidden, o.elements[2].Object = false, button
	o.elements[3].Hidden, o.elements[3].Object = false, button
	nodes = tree.Build(roots, nil)
	require.Len(t, nodes, 5)
	child, ok := tree.NodeForElement(o, "descendant")
	require.True(t, ok)
	o.Disable()
	nodes = tree.Build(roots, nil)
	assert.True(t, nodes[4].Disabled)
	assert.False(t, tree.Perform(child.ID, accessibility.Activate, "", 0, nil))
	o.Enable()
	o.elements[2].Hidden = true
	tree.Build(roots, nil)
	_, ok = tree.NodeForElement(o, "descendant")
	assert.False(t, ok, "hidden parents suppress exposed child descriptors")
	assert.False(t, tree.Perform(child.ID, accessibility.Activate, "", 0, nil))
	o.elements[2].Hidden = false
	tree.Build(roots, nil)
	child2, _ := tree.NodeForElement(o, "descendant")
	assert.Equal(t, child.ID, child2.ID)
}
