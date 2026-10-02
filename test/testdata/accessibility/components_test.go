package components_test

import (
	"testing"

	components "example.com/fyne-accessibility"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestExternalCustomWidgets(t *testing.T) {
	test.NewTempApp(t)
	calls := 0
	picker := components.NewFilePicker(func() { calls++ })
	toggle := components.NewToggle("Remember")
	w := test.NewWindow(container.NewVBox(picker, toggle))
	defer w.Close()
	semantics := test.NewAccessibilityTree(w.Canvas())
	browse, ok := semantics.Node(picker.Browse)
	if !ok || !semantics.Perform(browse.ID, test.AccessibilityActivate, "", 0) || calls != 1 {
		t.Fatal("nested button lost its semantics or original command")
	}
	field, ok := semantics.Node(picker.Path)
	if !ok || !semantics.Perform(field.ID, test.AccessibilitySetValue, "file.txt", 0) || picker.Path.Text != "file.txt" {
		t.Fatal("nested field lost its value command")
	}
	node, ok := semantics.Node(toggle)
	if !ok || !semantics.Perform(node.ID, test.AccessibilityToggle, "", 0) || !toggle.Checked {
		t.Fatal("custom toggle lost its own semantics")
	}
	if !semantics.Perform(node.ID, test.AccessibilityFocus, "", 0) {
		t.Fatal("cannot focus toggle")
	}
	w.Canvas().Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	if toggle.Checked {
		t.Fatal("keyboard and accessibility commands disagree")
	}
	if issues := semantics.Issues(); len(issues) != 0 {
		t.Fatalf("unexpected diagnostics: %v", issues)
	}
}

func TestExternalIndependentActionsAndReorder(t *testing.T) {
	test.NewTempApp(t)
	opened, removed := 0, 0
	open := widget.NewButton("Open", func() { opened++ })
	remove := widget.NewButton("Remove", func() { removed++ })
	row := container.NewHBox(open, remove)
	group := container.NewAccessibilityGroup("Document", row)
	w := test.NewWindow(group)
	defer w.Close()
	semantics := test.NewAccessibilityTree(w.Canvas())
	o, _ := semantics.Node(open)
	r, _ := semantics.Node(remove)
	row.Objects = []fyne.CanvasObject{remove, open}
	row.Refresh()
	o2, _ := semantics.Node(open)
	r2, _ := semantics.Node(remove)
	if o.ID != o2.ID || r.ID != r2.ID {
		t.Fatal("reorder changed child identities")
	}
	semantics.Perform(o.ID, test.AccessibilityActivate, "", 0)
	semantics.Perform(r.ID, test.AccessibilityActivate, "", 0)
	if opened != 1 || removed != 1 {
		t.Fatal("group merged independent commands")
	}
}
