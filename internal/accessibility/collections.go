package accessibility

import (
	"slices"
	"sort"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
)

// Only small identity records survive eviction. Semantic objects, native records
// and descriptions are bounded by the viewport, selection and this request LRU.
const collectionRequestLimit = 64

type elementReference struct {
	owner      fyne.CanvasObject
	key        string
	generation uint64
}

type collectionState struct {
	source     fyne.AccessibilityCollectionView
	revision   uint64
	active     bool
	requested  []string
	describers []fyne.AccessibleChildDescriber
}

func (t *Tree) forgetElements(owner fyne.CanvasObject) {
	for _, identity := range t.elements[owner] {
		delete(t.elementRefs, identity.id)
	}
	delete(t.elements, owner)
	delete(t.collections, owner)
}

func (s *elementSnapshot) window(source fyne.AccessibilityCollectionView) []Node {
	t := s.tree
	c := t.collections[s.owner]
	if c == nil {
		c = &collectionState{}
		t.collections[s.owner] = c
	}
	if s.previous == nil {
		s.previous = make(map[string]elementIdentity)
		t.elements[s.owner] = s.previous
	}
	if c.source == nil || c.revision != source.Revision() {
		for key, identity := range s.previous {
			e, ok := source.Element(key)
			if !ok || e.Key != key || e.Generation != identity.generation {
				delete(s.previous, key)
				delete(t.elementRefs, identity.id)
			}
		}
	}
	c.source, c.revision, c.describers = source, source.Revision(), s.describers
	if s.scope != nil {
		return nil
	}
	c.active = true
	n := t.nodes[s.parent]
	n.ItemContainer = true
	t.nodes[s.parent] = n
	wanted := append([]string(nil), source.ViewportKeys()...)
	wanted = append(wanted, source.SelectedKeys()...)
	for _, key := range c.requested {
		if e, ok := source.Element(key); ok && !e.Hidden {
			wanted = append(wanted, key)
		}
	}
	if active, ok := s.owner.(fyne.AccessibleActiveElement); ok {
		if key := active.AccessibilityActiveElement(); key != "" {
			wanted = append(wanted, key)
		}
	}
	paths := make(map[uint32][]int)
	visiting := make(map[string]bool)
	var materialize func(string) bool
	materialize = func(key string) bool {
		if s.current[key].id != 0 {
			return true
		}
		if visiting[key] {
			s.invalid()
			return false
		}
		visiting[key] = true
		defer delete(visiting, key)
		e, ok := source.Element(key)
		if !ok {
			s.invalid()
			return false
		}
		if e.Hidden {
			return false
		}
		parent, index, indexed := source.Index(key)
		if !indexed || index < 0 || e.Key != key || e.Parent != parent || source.ChildKey(parent, index) != key {
			s.invalid()
			return false
		}
		if parent != "" && !materialize(parent) {
			return false
		}
		if !s.add(e) {
			return false
		}
		identity := s.current[key]
		s.previous[key] = identity
		t.elementRefs[identity.id] = elementReference{s.owner, key, e.Generation}
		path := append([]int(nil), paths[s.current[parent].id]...)
		paths[identity.id] = append(path, index)
		if n, exposed := t.nodes[identity.id]; exposed {
			n.VirtualizedItem = true
			n.ItemContainer = source.ChildCount(key) > 0
			t.nodes[identity.id] = n
			s.nodes[len(s.nodes)-1] = n
		}
		return true
	}
	for _, key := range wanted {
		materialize(key)
	}
	sort.SliceStable(s.nodes, func(i, j int) bool { return slices.Compare(paths[s.nodes[i].ID], paths[s.nodes[j].ID]) < 0 })
	return s.nodes
}

// RequestElement pins a live model key in the bounded request cache. A subsequent
// Build publishes it. Lookup is independent of focus, selection and rendering.
func (t *Tree) RequestElement(owner fyne.CanvasObject, key string) bool {
	if !validObject(owner) {
		return false
	}
	c := t.collections[owner]
	if c == nil || !c.active {
		return false
	}
	if !c.validRequest(key) {
		return false
	}
	if index := slices.Index(c.requested, key); index >= 0 {
		c.requested = slices.Delete(c.requested, index, index+1)
	}
	c.requested = append(c.requested, key)
	if len(c.requested) > collectionRequestLimit {
		c.requested = slices.Delete(c.requested, 0, len(c.requested)-collectionRequestLimit)
	}
	return true
}

func (c *collectionState) validRequest(key string) bool {
	seen := make(map[string]bool)
	if key == "" {
		return false
	}
	for key != "" {
		if seen[key] {
			return false
		}
		seen[key] = true
		e, ok := c.source.Element(key)
		if !ok || e.Key != key || e.Hidden || !validObject(e.Object) || !e.Object.Visible() {
			return false
		}
		if _, accessible := e.Object.(fyne.Accessible); !accessible {
			return false
		}
		parent, index, indexed := c.source.Index(key)
		if !indexed || index < 0 || e.Parent != parent || c.source.ChildKey(parent, index) != key {
			return false
		}
		key = parent
	}
	return true
}

// Realize revalidates a retained logical ID, including its lifetime. Eviction is
// reversible; deletion/reinsertion, hidden branches and input scope are not.
func (t *Tree) Realize(id uint32) bool {
	ref, ok := t.elementRefs[id]
	if !ok {
		return false
	}
	if t.elements[ref.owner][ref.key].id != id {
		return false
	}
	c := t.collections[ref.owner]
	if c == nil || !c.active {
		return false
	}
	e, ok := c.source.Element(ref.key)
	return ok && e.Generation == ref.generation && t.RequestElement(ref.owner, ref.key)
}

// FindProperty is platform-independent; adapters map their property identifiers.
type FindProperty int

const (
	FindNext FindProperty = iota
	FindName
	FindSelected
	FindAutomationID
)

// FindItem finds a direct child of a collection or logical branch. It returns a
// key and owner to request before the next Build. False means an invalid request;
// an empty key with true means that a valid search has no match.
func (t *Tree) FindItem(container, startAfter uint32, property FindProperty, value string) (fyne.CanvasObject, string, bool) {
	owner, parent := t.objects[container], ""
	if ref, ok := t.elementRefs[container]; ok {
		owner, parent = ref.owner, ref.key
	}
	c := t.collections[owner]
	if c == nil || !c.active || !t.nodes[container].ItemContainer {
		return nil, "", false
	}
	if node := t.nodes[container]; node.Expandable && !node.Expanded {
		return owner, "", true
	}
	start, valid := t.findStart(c, owner, parent, startAfter)
	if !valid {
		return nil, "", false
	}
	if property == FindAutomationID {
		return owner, t.findAutomationID(c, owner, parent, start, value), true
	}
	if property < FindNext || property > FindSelected {
		return nil, "", false
	}
	selected := c.source.SelectedKeys()
	if property == FindSelected && value == "true" {
		return owner, c.findSelected(parent, start, selected), true
	}
	for index, count := start, c.source.ChildCount(parent); index < count; index++ {
		key := c.source.ChildKey(parent, index)
		e, ok := c.source.Element(key)
		if !ok || e.Hidden || e.Key != key || e.Parent != parent {
			continue
		}
		_, sibling, indexed := c.source.Index(key)
		if !indexed || sibling != index {
			continue
		}
		if property == FindSelected && slices.Contains(selected, key) != (value == "true") {
			continue
		}
		if property == FindName && elementName(e.Object, c.describers) != value {
			continue
		}
		return owner, key, true
	}
	return owner, "", true
}

func (t *Tree) findStart(c *collectionState, owner fyne.CanvasObject, parent string, after uint32) (int, bool) {
	if after == 0 {
		return 0, true
	}
	ref, ok := t.elementRefs[after]
	if !ok || ref.owner != owner || t.elements[owner][ref.key].id != after {
		return 0, false
	}
	e, exists := c.source.Element(ref.key)
	p, index, indexed := c.source.Index(ref.key)
	return index + 1, exists && !e.Hidden && e.Generation == ref.generation && indexed && p == parent
}

func (t *Tree) findAutomationID(c *collectionState, owner fyne.CanvasObject, parent string, start int, value string) string {
	id, err := strconv.ParseUint(strings.TrimPrefix(value, "fyne_"), 10, 32)
	if err != nil || !strings.HasPrefix(value, "fyne_") {
		return ""
	}
	ref, ok := t.elementRefs[uint32(id)]
	if !ok || ref.owner != owner {
		return ""
	}
	p, index, ok := c.source.Index(ref.key)
	if !ok || p != parent || index < start || !t.Realize(uint32(id)) {
		return ""
	}
	return ref.key
}

func (c *collectionState) findSelected(parent string, start int, selected []string) string {
	best, key := int(^uint(0)>>1), ""
	for _, candidate := range selected {
		p, index, ok := c.source.Index(candidate)
		if ok && p == parent && index >= start && index < best {
			best, key = index, candidate
		}
	}
	return key
}

func elementName(obj fyne.CanvasObject, describers []fyne.AccessibleChildDescriber) string {
	a, ok := obj.(fyne.Accessible)
	if !ok || !validObject(obj) {
		return ""
	}
	if d, ok := obj.(fyne.AccessibleDescribed); ok {
		if info := d.AccessibilityInfo(); info.NameSet || info.Name != "" {
			return info.Name
		}
	}
	for i := len(describers) - 1; i >= 0; i-- {
		if info := describers[i].AccessibilityChildInfo(obj); info.NameSet || info.Name != "" {
			return info.Name
		}
	}
	return a.AccessibilityLabel()
}
