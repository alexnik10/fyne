package accessibility

import (
	"reflect"

	"fyne.io/fyne/v2"
)

// Key membership, including hidden elements, is authoritative on every visible
// owner snapshot. Commands use only the fresh Object, never a recycled cell.
func (t *Tree) snapshotElements(owner fyne.CanvasObject, provider fyne.AccessibleElements, pos fyne.Position, parent uint32, scope fyne.CanvasObject, describers []fyne.AccessibleChildDescriber, clip *bounds) []Node {
	previous := t.elements[owner]
	current := make(map[string]uint32)
	visible := make(map[string]bool)
	position, size := clippedBounds(pos, owner.Size(), clip)
	clip = &bounds{position: position, size: size}
	var out []Node
	for _, element := range provider.AccessibilityElements() {
		if !validElement(element, current) {
			t.Issues = append(t.Issues, Issue{owner, "invalid-element", "Logical keys must be unique, nonempty and follow their parent; exposed elements need Accessible objects."})
			continue
		}
		id := previous[element.Key]
		if id == 0 {
			t.next++
			id = t.next
		}
		current[element.Key] = id
		shown := scope == nil && !element.Hidden && (element.Parent == "" || visible[element.Parent])
		if !shown || !validObject(element.Object) || !element.Object.Visible() {
			continue
		}
		visible[element.Key] = true
		parentID := parent
		if element.Parent != "" {
			parentID = current[element.Parent]
		}
		n := t.snapshotObject(element.Object, id, pos.Add(element.Object.Position()), parentID, describers, clip)
		if d, ok := owner.(fyne.Disableable); ok && d.Disabled() {
			n.Disabled, n.ReadOnly, n.Focusable = true, true, false
			t.nodes[id] = n
		}
		out = append(out, n)
	}
	t.elements[owner] = current
	return out
}

func validObject(obj fyne.CanvasObject) bool {
	if obj == nil || !reflect.TypeOf(obj).Comparable() {
		return false
	}
	v := reflect.ValueOf(obj)
	return v.Kind() != reflect.Pointer || !v.IsNil()
}

func validElement(element fyne.AccessibilityElement, keys map[string]uint32) bool {
	if element.Key == "" || keys[element.Key] != 0 || (element.Parent != "" && keys[element.Parent] == 0) {
		return false
	}
	if element.Hidden {
		return true
	}
	_, accessible := element.Object.(fyne.Accessible)
	return accessible && validObject(element.Object)
}

func (t *Tree) liveID(id uint32) uint32 {
	if _, live := t.nodes[id]; live {
		return id
	}
	return 0
}

// NodeForElement finds a currently exposed owner/key pair.
func (t *Tree) NodeForElement(owner fyne.CanvasObject, key string) (Node, bool) {
	if !validObject(owner) {
		return Node{}, false
	}
	n, ok := t.nodes[t.elements[owner][key]]
	return n, ok
}
