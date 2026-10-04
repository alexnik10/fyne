package widget

import (
	"sort"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/internal/cache"
	"fyne.io/fyne/v2/internal/driver"
	"fyne.io/fyne/v2/lang"
)

// AccessibilityLabel returns the explicit metadata fallback.
// Since: 2.9
func (*Table) AccessibilityLabel() string { return "" }

// AccessibilityRole identifies a table.
// Since: 2.9
func (*Table) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleTable }

// AccessibilitySelection describes optional single-cell selection.
// Since: 2.9
func (*Table) AccessibilitySelection() (multiple, required bool) { return false, false }

// AccessibilityGrid returns data dimensions, excluding headers.
// Since: 2.9
func (t *Table) AccessibilityGrid() (rows, columns int) {
	if t.Length == nil {
		return 0, 0
	}
	r, c := t.Length()
	return max(0, r), max(0, c)
}

// AccessibilityTable reports available header axes.
// Since: 2.9
func (t *Table) AccessibilityTable() (rowHeaders, columnHeaders bool) {
	return t.ShowHeaderColumn, t.ShowHeaderRow
}

// AccessibilityCellKey returns a stable collection key for a cell or header.
// Since: 2.9
func (t *Table) AccessibilityCellKey(row, column int) string {
	return t.ensureAccessibilitySource().key(TableCellID{row, column})
}

// AccessibilityActiveElement follows the keyboard highlight.
// Since: 2.9
func (t *Table) AccessibilityActiveElement() string {
	return t.ensureAccessibilitySource().key(t.currentHighlight)
}

// AccessibilityCollection describes only requested and visible cells. Indexing
// costs O(rows+columns), not O(rows*columns), and never renders a cell.
// Since: 2.9
func (t *Table) AccessibilityCollection() fyne.AccessibilityCollection {
	return t.ensureAccessibilitySource()
}

// Refresh reconciles keyed row/column state before refreshing the visual cells.
func (t *Table) Refresh() {
	old := t.accessibilityCache
	if old != nil {
		if old.rowKeyed != (t.RowKey != nil) {
			t.rowLifetimes.generations = nil
		}
		if old.columnKeyed != (t.ColumnKey != nil) {
			t.columnLifetimes.generations = nil
		}
	}
	t.accessibilityCache = nil
	if old != nil || t.RowKey != nil || t.ColumnKey != nil {
		s := t.ensureAccessibilitySource()
		if old != nil {
			if s.rowKeyed && old.rowKeyed {
				t.rowHeights = remapTableSizes(t.rowHeights, old.rows, s.rowIndex)
			}
			if s.columnKeyed && old.columnKeyed {
				t.columnWidths = remapTableSizes(t.columnWidths, old.columns, s.columnIndex)
			}
		}
		remap := func(id TableCellID) (TableCellID, bool) { return s.remap(old, id) }
		if t.selectedCell != nil {
			previous := *t.selectedCell
			if id, ok := remap(previous); ok {
				t.selectedCell = &id
			} else {
				t.selectedCell = nil
				if t.OnUnselected != nil {
					t.OnUnselected(previous)
				}
			}
		}
		if id, ok := remap(t.currentHighlight); ok {
			t.currentHighlight = id
		} else {
			t.currentHighlight = TableCellID{}
		}
	}
	t.reconcileCellEdit()
	t.BaseWidget.Refresh()
}

func (s *tableAccessibilitySource) remap(old *tableAccessibilitySource, id TableCellID) (TableCellID, bool) {
	if old == nil {
		return id, s.valid(id)
	}
	if s.rowKeyed && old.rowKeyed {
		id.Row = remapTableAxis(id.Row, old.rows, s.rowIndex)
	}
	if s.columnKeyed && old.columnKeyed {
		id.Col = remapTableAxis(id.Col, old.columns, s.columnIndex)
	}
	return id, s.valid(id)
}

func remapTableAxis(index int, old []string, next map[string]int) int {
	if index < 0 || index >= len(old) {
		return -1
	}
	if result, ok := next[old[index]]; ok {
		return result
	}
	return -1
}

func remapTableSizes(sizes map[int]float32, old []string, next map[string]int) map[int]float32 {
	if sizes == nil {
		return nil
	}
	result := make(map[int]float32, len(sizes))
	for index, size := range sizes {
		if index == -1 {
			result[index] = size
			continue
		}
		if mapped := remapTableAxis(index, old, next); mapped >= 0 {
			result[mapped] = size
		}
	}
	return result
}

type tableAccessibilitySource struct {
	owner                           *Table
	rows, columns                   []string
	rowIndex, columnIndex           map[string]int
	rowGeneration, columnGeneration map[string]uint64
	revision                        uint64
	rowKeyed, columnKeyed           bool
	geometry                        map[TableCellID]tableSemanticBounds
}
type tableSemanticBounds struct {
	position fyne.Position
	size     fyne.Size
}

func tableAxis(count int, key func(int) string) ([]string, map[string]int) {
	keys, index := make([]string, count), make(map[string]int, count)
	for n := range keys {
		k := strconv.Itoa(n)
		if key != nil {
			k = key(n)
		}
		keys[n] = k
		if _, exists := index[k]; exists || k == "" {
			index[k] = -1
		} else {
			index[k] = n
		}
	}
	return keys, index
}

func (t *Table) ensureAccessibilitySource() *tableAccessibilitySource {
	if t.accessibilityCache != nil {
		return t.accessibilityCache
	}
	r, c := t.AccessibilityGrid()
	rows, ri := tableAxis(r, t.RowKey)
	cols, ci := tableAxis(c, t.ColumnKey)
	t.rowLifetimes.update(rows)
	t.columnLifetimes.update(cols)
	t.accessibilityRevision++
	s := &tableAccessibilitySource{owner: t, rows: rows, columns: cols, rowIndex: ri, columnIndex: ci, rowGeneration: t.rowLifetimes.generations, columnGeneration: t.columnLifetimes.generations, revision: t.accessibilityRevision, rowKeyed: t.RowKey != nil, columnKeyed: t.ColumnKey != nil}
	t.accessibilityCache = s
	return s
}

func (s *tableAccessibilitySource) valid(id TableCellID) bool {
	return id.Row >= 0 && id.Col >= 0 && id.Row < len(s.rows) && id.Col < len(s.columns)
}

func (s *tableAccessibilitySource) key(id TableCellID) string {
	if id.Row == -1 && id.Col >= 0 && id.Col < len(s.columns) && s.owner.ShowHeaderRow {
		return "c:" + s.columns[id.Col]
	}
	if id.Col == -1 && id.Row >= 0 && id.Row < len(s.rows) && s.owner.ShowHeaderColumn {
		return "r:" + s.rows[id.Row]
	}
	if !s.valid(id) {
		return ""
	}
	row := s.rows[id.Row]
	return "d:" + strconv.Itoa(len(row)) + ":" + row + s.columns[id.Col]
}

func (s *tableAccessibilitySource) resolve(key string) (TableCellID, bool) {
	id := TableCellID{-1, -1}
	var ok bool
	switch {
	case strings.HasPrefix(key, "c:"):
		id.Col, ok = s.columnIndex[key[2:]]
		ok = ok && s.owner.ShowHeaderRow && id.Col >= 0
	case strings.HasPrefix(key, "r:"):
		id.Row, ok = s.rowIndex[key[2:]]
		ok = ok && s.owner.ShowHeaderColumn && id.Row >= 0
	case strings.HasPrefix(key, "d:"):
		parts := strings.SplitN(key[2:], ":", 2)
		if len(parts) != 2 {
			return id, false
		}
		n, err := strconv.Atoi(parts[0])
		if err != nil || n < 0 || n > len(parts[1]) {
			return id, false
		}
		var rowOK, colOK bool
		id.Row, rowOK = s.rowIndex[parts[1][:n]]
		id.Col, colOK = s.columnIndex[parts[1][n:]]
		ok = rowOK && colOK && s.valid(id)
	}
	return id, ok && s.key(id) == key
}
func (s *tableAccessibilitySource) Revision() uint64 { return s.revision }
func (s *tableAccessibilitySource) ChildCount(parent string) int {
	if parent != "" {
		return 0
	}
	n := len(s.rows) * len(s.columns)
	if s.owner.ShowHeaderRow {
		n += len(s.columns)
	}
	if s.owner.ShowHeaderColumn {
		n += len(s.rows)
	}
	return n
}

func (s *tableAccessibilitySource) ChildKey(parent string, n int) string {
	if parent != "" || n < 0 || n >= s.ChildCount("") {
		return ""
	}
	if s.owner.ShowHeaderRow {
		if n < len(s.columns) {
			return s.key(TableCellID{-1, n})
		}
		n -= len(s.columns)
	}
	if s.owner.ShowHeaderColumn {
		if n < len(s.rows) {
			return s.key(TableCellID{n, -1})
		}
		n -= len(s.rows)
	}
	if len(s.columns) == 0 {
		return ""
	}
	return s.key(TableCellID{n / len(s.columns), n % len(s.columns)})
}

func (s *tableAccessibilitySource) Index(key string) (string, int, bool) {
	id, ok := s.resolve(key)
	if !ok {
		return "", 0, false
	}
	n := 0
	if id.Row == -1 {
		return "", id.Col, true
	}
	if s.owner.ShowHeaderRow {
		n += len(s.columns)
	}
	if id.Col == -1 {
		return "", n + id.Row, true
	}
	if s.owner.ShowHeaderColumn {
		n += len(s.rows)
	}
	return "", n + id.Row*len(s.columns) + id.Col, true
}

func (s *tableAccessibilitySource) SelectedKeys() []string {
	if id := s.owner.selectedCell; id != nil {
		if key := s.key(*id); key != "" {
			return []string{key}
		}
	}
	return nil
}

func (s *tableAccessibilitySource) ViewportKeys() []string {
	s.geometry = make(map[TableCellID]tableSemanticBounds)
	t := s.owner
	if t.cells == nil {
		return nil
	}
	r, ok := cache.CachedRenderer(t.cells)
	if !ok {
		return nil
	}
	cells, ok := r.(*tableCellsRenderer)
	if !ok {
		return nil
	}
	objects := make(map[fyne.CanvasObject]TableCellID, len(cells.visible)+len(cells.headers))
	for id, obj := range cells.visible {
		objects[obj] = id
	}
	for id, obj := range cells.headers {
		objects[obj] = id
	}
	var keys []string
	driver.WalkVisibleObjectTree(t.super(), func(obj fyne.CanvasObject, p, clip fyne.Position, size fyne.Size) bool {
		id, ok := objects[obj]
		if !ok {
			return false
		}
		left, top := max(p.X, clip.X), max(p.Y, clip.Y)
		right, bottom := min(p.X+obj.Size().Width, clip.X+size.Width), min(p.Y+obj.Size().Height, clip.Y+size.Height)
		left, top = left-t.Position().X, top-t.Position().Y
		right, bottom = right-t.Position().X, bottom-t.Position().Y
		s.geometry[id] = tableSemanticBounds{fyne.NewPos(left, top), fyne.NewSize(max(0, right-left), max(0, bottom-top))}
		if key := s.key(id); key != "" {
			keys = append(keys, key)
		}
		return false
	}, nil)
	sort.Slice(keys, func(i, j int) bool { _, a, _ := s.Index(keys[i]); _, b, _ := s.Index(keys[j]); return a < b })
	return keys
}

func (s *tableAccessibilitySource) Element(key string) (fyne.AccessibilityElement, bool) {
	id, ok := s.resolve(key)
	if !ok {
		return fyne.AccessibilityElement{}, false
	}
	// Axis generations are encoded in the logical lifetime token, without a
	// per-cell index. The same live pair survives both sorting and eviction.
	var row, col uint64
	if id.Row >= 0 {
		row = s.rowGeneration[s.rows[id.Row]]
	}
	if id.Col >= 0 {
		col = s.columnGeneration[s.columns[id.Col]]
	}
	const axisGenerationBits = 32
	generation := row<<axisGenerationBits | col
	base := tableAccessibilityObject{owner: s.owner, key: key, id: id, generation: generation, bounds: s.geometry[id]}
	var obj fyne.CanvasObject = &tableAccessibilityCell{tableAccessibilityObject: base}
	if id.Row < 0 || id.Col < 0 {
		obj = &tableAccessibilityHeader{tableAccessibilityObject: base}
	} else if s.owner.CellValue != nil {
		cell := tableAccessibilityValueCell{tableAccessibilityCell{tableAccessibilityObject: base}}
		obj = &cell
		if _, readOnly := s.owner.CellValue(id); !readOnly && s.owner.OnCellChanged != nil {
			obj = &tableAccessibilityEditableCell{cell}
		}
	}
	return fyne.AccessibilityElement{Key: key, Generation: generation, Object: obj}, true
}

type tableAccessibilityObject struct {
	owner      *Table
	key        string
	id         TableCellID
	generation uint64
	bounds     tableSemanticBounds
}

func (i *tableAccessibilityObject) AccessibilityInfo() fyne.AccessibilityInfo {
	if f := i.owner.DescribeCell; f != nil {
		return f(i.id)
	}
	return fyne.AccessibilityInfo{}
}

func (i *tableAccessibilityObject) resolve() (TableCellID, bool) {
	s := i.owner.ensureAccessibilitySource()
	e, ok := s.Element(i.key)
	if !ok || e.Generation != i.generation {
		return TableCellID{}, false
	}
	return s.resolve(i.key)
}
func (i *tableAccessibilityObject) Position() fyne.Position { return i.bounds.position }
func (i *tableAccessibilityObject) Size() fyne.Size         { return i.bounds.size }
func (*tableAccessibilityObject) MinSize() fyne.Size        { return fyne.Size{} }
func (*tableAccessibilityObject) Move(fyne.Position)        {}
func (*tableAccessibilityObject) Resize(fyne.Size)          {}
func (*tableAccessibilityObject) Hide()                     {}
func (*tableAccessibilityObject) Show()                     {}
func (*tableAccessibilityObject) Visible() bool             { return true }
func (*tableAccessibilityObject) Refresh()                  {}

type tableAccessibilityHeader struct{ tableAccessibilityObject }

func (*tableAccessibilityHeader) AccessibilityRole() fyne.AccessibleRole {
	return fyne.AccessibleRoleHeader
}

func (i *tableAccessibilityHeader) AccessibilityLabel() string {
	if i.id.Col < 0 {
		return strconv.Itoa(i.id.Row + 1)
	}
	n := i.id.Col + 1
	label := ""
	for n > 0 {
		n--
		label = string(rune('A'+n%columnLetterCount)) + label
		n /= columnLetterCount
	}
	return label
}

type tableAccessibilityCell struct{ tableAccessibilityObject }

func (*tableAccessibilityCell) AccessibilityRole() fyne.AccessibleRole {
	return fyne.AccessibleRoleCell
}

func (i *tableAccessibilityCell) AccessibilityLabel() string {
	return lang.X("accessibility.table.cell", "Row {{.Row}}, column {{.Column}}", map[string]any{"Row": i.id.Row + 1, "Column": i.id.Col + 1})
}

func (i *tableAccessibilityCell) AccessibilityGridItem() (owner fyne.CanvasObject, row, column, rowSpan, columnSpan int) {
	return i.owner.super(), i.id.Row, i.id.Col, 1, 1
}

func (i *tableAccessibilityCell) AccessibilitySelectionItem() (owner fyne.CanvasObject, selected bool, position, count int) {
	return i.owner.super(), i.owner.selectedCell != nil && *i.owner.selectedCell == i.id, 0, 0
}

func (i *tableAccessibilityCell) AccessibilitySelect(mode fyne.AccessibilitySelectionMode) bool {
	id, ok := i.resolve()
	if !ok {
		return false
	}
	switch mode {
	case fyne.AccessibilitySelectRemove:
		i.owner.Unselect(id)
	case fyne.AccessibilitySelectAdd:
		if i.owner.selectedCell != nil && *i.owner.selectedCell != id {
			return false
		}
		i.owner.Select(id)
	case fyne.AccessibilitySelectReplace:
		i.owner.Select(id)
	default:
		return false
	}
	return true
}
func (*tableAccessibilityCell) AccessibilityFocusable() bool { return true }
func (i *tableAccessibilityCell) AccessibilityFocus() bool {
	id, ok := i.resolve()
	if !ok {
		return false
	}
	c := fyne.CurrentApp().Driver().CanvasForObject(i.owner.super())
	f, ok := i.owner.super().(fyne.Focusable)
	if c == nil || !ok {
		return false
	}
	i.owner.Highlight(id)
	c.Focus(f)
	return c.Focused() == f
}

func (i *tableAccessibilityCell) AccessibilityScrollIntoView() bool {
	id, ok := i.resolve()
	if !ok || i.owner.content == nil {
		return false
	}
	i.owner.ScrollTo(id)
	return true
}
