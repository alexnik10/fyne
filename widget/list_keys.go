package widget

import "strconv"

func (l *List) modelKeys() ([]string, map[string]int) {
	count := 0
	if l.Length != nil {
		count = max(0, l.Length())
	}
	keys, indices := make([]string, count), make(map[string]int, count)
	for index := range keys {
		key := l.keyForIndex(index)
		keys[index] = key
		if _, exists := indices[key]; exists || key == "" {
			indices[key] = -1 // Ambiguous keys must never dispatch a command.
		} else {
			indices[key] = index
		}
	}
	return keys, indices
}

func (l *List) keyForIndex(index int) string {
	if l.ItemKey != nil {
		return l.ItemKey(index)
	}
	return strconv.Itoa(index)
}

func (l *List) ensureItemKeys() {
	if l.itemKeys == nil || l.keyed != (l.ItemKey != nil) {
		l.reconcileItems()
	}
}

func (l *List) reconcileItems() {
	if l.ItemKey == nil && !l.keyed {
		l.reconcilePositionalItems()
		return
	}
	keys, indices := l.modelKeys()
	old, keyed := l.itemKeys, l.ItemKey != nil
	l.itemKeys = keys
	if l.keyed != keyed {
		l.lifetimes.generations = nil
	}
	preserve := l.keyed && keyed && old != nil
	l.keyed = keyed
	l.lifetimes.update(keys)
	mapIndex := func(index int) int {
		if preserve {
			if index < 0 || index >= len(old) || old[index] == "" {
				return -1
			}
			if next, ok := indices[old[index]]; ok {
				return next
			}
			return -1
		}
		if index < 0 || index >= len(keys) {
			return -1
		}
		return index
	}
	if preserve {
		heights := make(map[ListItemID]float32, len(l.itemHeights))
		for index, height := range l.itemHeights {
			if next := mapIndex(index); next >= 0 {
				heights[next] = height
			}
		}
		l.itemHeights = heights
	}
	l.reconcileItemState(mapIndex, len(keys))
}

func (l *List) reconcileItemState(mapIndex func(int) int, count int) {
	removed := -1
	if len(l.selected) > 0 {
		previous := l.selected[0]
		if next := mapIndex(previous); next >= 0 {
			l.selected[0] = next
		} else {
			l.selected, removed = nil, previous
		}
	}
	previous := l.currentHighlight
	next := mapIndex(previous)
	l.currentHighlight = max(0, next)
	if removed >= 0 && l.OnUnselected != nil {
		l.OnUnselected(removed)
	}
	if (next < 0 || previous != l.currentHighlight) && count > 0 && l.OnHighlighted != nil {
		l.OnHighlighted(l.currentHighlight)
	}
}

// Preserve the O(1) refresh path of positional lists when no adapter has asked
// for logical identities. Only already-published positions need retirement.
func (l *List) reconcilePositionalItems() {
	count := 0
	if l.Length != nil {
		count = max(0, l.Length())
	}
	for key := range l.lifetimes.generations {
		index, err := strconv.Atoi(key)
		if err != nil || index < 0 || index >= count {
			delete(l.lifetimes.generations, key)
		}
	}
	l.reconcileItemState(func(index int) int {
		if index < 0 || index >= count {
			return -1
		}
		return index
	}, count)
}
