package widget

import (
	"encoding/base64"
	"strings"

	"fyne.io/fyne/v2"
)

// CollectionControlKey returns the key of a model-backed control below an item.
// Use it with the public accessibility inspector. The NUL prefix is reserved for
// this encoding when ItemElements, NodeElements or CellElements is configured.
// Since: 2.9
func CollectionControlKey(item, control string) string {
	return "\x00" + base64.RawURLEncoding.EncodeToString([]byte(item)) + ":" + base64.RawURLEncoding.EncodeToString([]byte(control))
}

func decodeCollectionControl(key string) (root, local string, childKey bool) {
	if !strings.HasPrefix(key, "\x00") {
		return "", "", false
	}
	parts := strings.SplitN(key[1:], ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	parent, e1 := base64.RawURLEncoding.DecodeString(parts[0])
	child, e2 := base64.RawURLEncoding.DecodeString(parts[1])
	return string(parent), string(child), e1 == nil && e2 == nil && len(parent) > 0 && len(child) > 0
}

type (
	collectionChildIdentity struct{ parent, child, id uint64 }
	collectionChildIDs      struct {
		next uint64
		ids  map[string]collectionChildIdentity
	}
)

func (c *collectionChildIDs) generation(key string, parent, child uint64) uint64 {
	if c.ids == nil {
		c.ids = make(map[string]collectionChildIdentity)
	}
	id := c.ids[key]
	if id.id == 0 || id.parent != parent || id.child != child {
		c.next++
		id = collectionChildIdentity{parent, child, c.next}
		c.ids[key] = id
	}
	return id.id
}

type collectionWithChildren struct {
	fyne.AccessibilityCollectionView
	elements func(string) []fyne.AccessibilityElement
	ids      *collectionChildIDs
	cache    map[string][]fyne.AccessibilityElement
}

func (s *collectionWithChildren) descriptors(root string) []fyne.AccessibilityElement {
	parent, ok := s.AccessibilityCollectionView.Element(root)
	if !ok || parent.Hidden {
		return nil
	}
	if s.cache == nil {
		s.cache = make(map[string][]fyne.AccessibilityElement)
	}
	if elements, ok := s.cache[root]; ok {
		return elements
	}
	elements := s.elements(root)
	seen := make(map[string]bool, len(elements))
	for _, e := range elements {
		if e.Key == "" || seen[e.Key] || (e.Parent != "" && !seen[e.Parent]) {
			s.cache[root] = nil
			return nil
		}
		seen[e.Key] = true
	}
	s.cache[root] = elements
	return elements
}

func (s *collectionWithChildren) childKeys(parent string) []string {
	root, local, child := decodeCollectionControl(parent)
	if !child {
		root, local = parent, ""
	}
	if root == "" {
		return nil
	}
	var keys []string
	for _, e := range s.descriptors(root) {
		if e.Parent == local {
			keys = append(keys, CollectionControlKey(root, e.Key))
		}
	}
	return keys
}

func (s *collectionWithChildren) ChildCount(parent string) int {
	return s.AccessibilityCollectionView.ChildCount(parent) + len(s.childKeys(parent))
}

func (s *collectionWithChildren) ChildKey(parent string, index int) string {
	count := s.AccessibilityCollectionView.ChildCount(parent)
	if index < count {
		return s.AccessibilityCollectionView.ChildKey(parent, index)
	}
	keys := s.childKeys(parent)
	index -= count
	if index < 0 || index >= len(keys) {
		return ""
	}
	return keys[index]
}

func (s *collectionWithChildren) Index(key string) (string, int, bool) {
	_, _, child := decodeCollectionControl(key)
	if !child {
		return s.AccessibilityCollectionView.Index(key)
	}
	e, ok := s.Element(key)
	if !ok {
		return "", 0, false
	}
	parent := e.Parent
	for i, k := range s.childKeys(parent) {
		if k == key {
			return parent, s.AccessibilityCollectionView.ChildCount(parent) + i, true
		}
	}
	return "", 0, false
}

func (s *collectionWithChildren) Element(key string) (fyne.AccessibilityElement, bool) {
	root, local, child := decodeCollectionControl(key)
	if !child {
		return s.AccessibilityCollectionView.Element(key)
	}
	parent, ok := s.AccessibilityCollectionView.Element(root)
	if !ok {
		return fyne.AccessibilityElement{}, false
	}
	for _, e := range s.descriptors(root) {
		if e.Key != local {
			continue
		}
		e.Key = key
		if e.Parent == "" {
			e.Parent = root
		} else {
			e.Parent = CollectionControlKey(root, e.Parent)
		}
		e.Hidden = e.Hidden || parent.Hidden
		e.Generation = s.ids.generation(key, parent.Generation, e.Generation)
		if e.Object != nil {
			position := e.Object.Position()
			if e.Position != nil {
				position = *e.Position
			}
			if parent.Object != nil {
				position = position.Add(parent.Object.Position())
			}
			e.Position = &position
		}
		return e, true
	}
	delete(s.ids.ids, key)
	return fyne.AccessibilityElement{}, false
}

func (s *collectionWithChildren) ViewportKeys() []string {
	keys := append([]string(nil), s.AccessibilityCollectionView.ViewportKeys()...)
	for _, root := range append([]string(nil), keys...) {
		for _, e := range s.descriptors(root) {
			keys = append(keys, CollectionControlKey(root, e.Key))
		}
	}
	return keys
}

func (l *List) accessibilityWithChildren(base fyne.AccessibilityCollectionView) fyne.AccessibilityCollection {
	if l.ItemElements == nil {
		return base
	}
	return &collectionWithChildren{AccessibilityCollectionView: base, elements: func(key string) []fyne.AccessibilityElement {
		_, index, ok := base.Index(key)
		if !ok {
			return nil
		}
		return l.ItemElements(index)
	}, ids: &l.accessibilityChildIDs}
}

func (l *GridWrap) accessibilityWithChildren(base fyne.AccessibilityCollectionView) fyne.AccessibilityCollection {
	if l.ItemElements == nil {
		return base
	}
	return &collectionWithChildren{AccessibilityCollectionView: base, elements: func(key string) []fyne.AccessibilityElement {
		_, index, ok := base.Index(key)
		if !ok {
			return nil
		}
		return l.ItemElements(index)
	}, ids: &l.accessibilityChildIDs}
}

func (t *Tree) accessibilityWithChildren(base fyne.AccessibilityCollectionView) fyne.AccessibilityCollection {
	if t.NodeElements == nil {
		return base
	}
	return &collectionWithChildren{AccessibilityCollectionView: base, elements: t.NodeElements, ids: &t.accessibilityChildIDs}
}

func (t *Table) accessibilityWithChildren(base *tableAccessibilitySource) fyne.AccessibilityCollection {
	if t.CellElements == nil {
		return base
	}
	return &collectionWithChildren{AccessibilityCollectionView: base, elements: func(key string) []fyne.AccessibilityElement {
		id, ok := base.resolve(key)
		if !ok || id.Row < 0 || id.Col < 0 {
			return nil
		}
		return t.CellElements(id)
	}, ids: &t.accessibilityChildIDs}
}
