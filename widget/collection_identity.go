package widget

// Lifetimes belong to model keys, independently of visual cells and adapters.
// Refresh records removals even when no accessibility snapshot is taken between
// removal and reinsertion. Collapsing a branch does not remove its model keys.
type collectionLifetimes struct {
	next        uint64
	generations map[string]uint64
}

func (c *collectionLifetimes) update(keys []string) {
	current := make(map[string]uint64, len(keys))
	for _, key := range keys {
		if key == "" || current[key] != 0 {
			continue
		}
		generation := c.generations[key]
		if generation == 0 {
			c.next++
			generation = c.next
		}
		current[key] = generation
	}
	c.generations = current
}
