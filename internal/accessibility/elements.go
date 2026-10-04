package accessibility

import (
	"reflect"

	"fyne.io/fyne/v2"
)

type elementIdentity struct {
	id         uint32
	generation uint64
}

type elementSnapshot struct {
	tree              *Tree
	owner             fyne.CanvasObject
	previous, current map[string]elementIdentity
	visible           map[string]bool
	position          fyne.Position
	parent            uint32
	scope             fyne.CanvasObject
	describers        []fyne.AccessibleChildDescriber
	clip              *bounds
	nodes             []Node
}

// Key membership, including hidden elements, is authoritative on every visible
// owner snapshot. Commands use only the fresh Object, never a recycled cell.
func (t *Tree) snapshotElements(owner fyne.CanvasObject, pos fyne.Position, parent uint32, scope fyne.CanvasObject, describers []fyne.AccessibleChildDescriber, clip *bounds) ([]Node, bool) {
	collection, indexed := owner.(fyne.AccessibleCollection)
	elements, sliced := owner.(fyne.AccessibleElements)
	if !indexed && !sliced {
		return nil, false
	}
	position, size := clippedBounds(pos, owner.Size(), clip)
	s := elementSnapshot{
		tree: t, owner: owner, previous: t.elements[owner],
		current: make(map[string]elementIdentity), visible: make(map[string]bool),
		position: pos, parent: parent, scope: scope, describers: describers,
		clip: &bounds{position: position, size: size},
	}
	if indexed {
		if source := collection.AccessibilityCollection(); source != nil {
			if view, ok := source.(fyne.AccessibilityCollectionView); ok && parent != 0 && parent == t.ids[owner] {
				return s.window(view), true
			}
			s.visit(source, "", make(map[string]bool))
		}
	} else {
		for _, element := range elements.AccessibilityElements() {
			s.add(element)
		}
	}
	t.elements[owner] = s.current
	return s.nodes, true
}

func (s *elementSnapshot) invalid() {
	s.tree.Issues = append(s.tree.Issues, Issue{s.owner, "invalid-element", "Logical keys must be unique, nonempty and follow their parent; exposed elements need Accessible objects."})
}

func (s *elementSnapshot) visit(source fyne.AccessibilityCollection, parent string, seen map[string]bool) {
	count := source.ChildCount(parent)
	if count < 0 {
		s.invalid()
		return
	}
	for index := 0; index < count; index++ {
		key := source.ChildKey(parent, index)
		if key == "" || seen[key] {
			s.invalid()
			continue
		}
		seen[key] = true
		element, ok := source.Element(key)
		if !ok || element.Key != key || element.Parent != parent {
			s.invalid()
			continue
		}
		if s.add(element) {
			s.visit(source, key, seen)
		}
	}
}

func (s *elementSnapshot) add(element fyne.AccessibilityElement) bool {
	if !validElement(element, s.current) {
		s.invalid()
		return false
	}
	identity := s.previous[element.Key]
	if identity.id == 0 || identity.generation != element.Generation {
		s.tree.next++
		identity = elementIdentity{s.tree.next, element.Generation}
	}
	id := identity.id
	s.current[element.Key] = identity
	shown := s.scope == nil && !element.Hidden && (element.Parent == "" || s.visible[element.Parent])
	if !shown || !validObject(element.Object) || !element.Object.Visible() {
		return true
	}
	s.visible[element.Key] = true
	parentID := s.parent
	if element.Parent != "" {
		parentID = s.current[element.Parent].id
	}
	position := element.Object.Position()
	if element.Position != nil {
		position = *element.Position
	}
	n := s.tree.snapshotObject(element.Object, id, s.position.Add(position), parentID, s.describers, s.clip)
	_, routed := element.Object.(fyne.AccessibleFocusHandler)
	n.Focusable = !n.Disabled && (element.FocusTarget != nil || (routed && n.Focusable))
	if element.FocusTarget != nil {
		s.tree.focusTargets[id] = element.FocusTarget
	}
	if d, ok := s.owner.(fyne.Disableable); ok && d.Disabled() {
		n.Disabled, n.ReadOnly, n.Focusable = true, true, false
	}
	s.tree.nodes[id] = n
	s.nodes = append(s.nodes, n)
	return true
}

func validObject(obj fyne.CanvasObject) bool {
	if obj == nil || !reflect.TypeOf(obj).Comparable() {
		return false
	}
	v := reflect.ValueOf(obj)
	return v.Kind() != reflect.Pointer || !v.IsNil()
}

func validElement(element fyne.AccessibilityElement, keys map[string]elementIdentity) bool {
	if element.Key == "" || keys[element.Key].id != 0 || (element.Parent != "" && keys[element.Parent].id == 0) {
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
	n, ok := t.nodes[t.elements[owner][key].id]
	return n, ok
}
