package widget

import (
	"image/color"
	"sort"
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// AccessibilityRole identifies a formatted document.
// Since: 2.9
func (*RichText) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleDocument }

// AccessibilityLabel leaves the document name to explicit metadata.
// Since: 2.9
func (*RichText) AccessibilityLabel() string { return "" }

// AccessibilityChildren preserves embedded links and controls.
// Since: 2.9
func (t *RichText) AccessibilityChildren() []fyne.CanvasObject {
	if t.scr != nil {
		return []fyne.CanvasObject{t.scr}
	}
	return t.visuals
}

// AccessibilityFocusable reports whether this document supports selection.
// Since: 2.9
func (t *RichText) AccessibilityFocusable() bool { return t.Selectable }

// AccessibilityFocus focuses the existing selection widget.
// Since: 2.9
func (t *RichText) AccessibilityFocus() bool {
	sel := t.selectionWidget()
	c := fyne.CurrentApp().Driver().CanvasForObject(t.super())
	if sel == nil || c == nil {
		return false
	}
	c.Focus(sel)
	return c.Focused() == sel
}

// AccessibilityActiveDescendant maps the real selection focus to its document.
// Since: 2.9
func (f *focusSelectable) AccessibilityActiveDescendant() fyne.CanvasObject {
	if f.accessibilityOwner != nil {
		return f.accessibilityOwner
	}
	return f.provider
}

// AccessibilityMode omits the selection overlay; its owner exposes the text.
// Since: 2.9
func (*focusSelectable) AccessibilityMode() fyne.AccessibilityMode { return fyne.AccessibilityExclude }

// AccessibilityText exposes paragraph separators, list markers and formatting.
// Since: 2.9
func (t *RichText) AccessibilityText() fyne.AccessibilityTextInfo {
	info, offsets, reverse := t.accessibilityDocument()
	info.ReadOnly, info.SelectionDisabled = true, !t.Selectable
	if sel := t.selection; t.Selectable && sel != nil {
		caret := min(max(textPosFromRowCol(sel.cursorRow, sel.cursorColumn, t), 0), len(reverse)-1)
		info.Caret = reverse[caret]
		info.SelectionStart, info.SelectionEnd = info.Caret, info.Caret
		if sel.selecting {
			anchor := min(max(textPosFromRowCol(sel.selectRow, sel.selectColumn, t), 0), len(reverse)-1)
			info.SelectionStart, info.SelectionEnd = min(reverse[anchor], info.Caret), max(reverse[anchor], info.Caret)
		}
	}
	info.ViewportSize = t.Size()
	origin := fyne.Position{}
	if t.scr != nil {
		origin = t.scr.Position().Subtract(t.scr.Offset)
		info.ViewportPosition, info.ViewportSize = t.scr.Position(), t.scr.Size()
	}
	pad, size := t.Theme().Size(theme.SizeNameInnerPadding), t.Theme().Size(theme.SizeNameText)
	info.Positions = make([]fyne.AccessibilityTextPosition, len(offsets))
	for i, raw := range offsets {
		row, col := t.accessibilityRowColumn(raw)
		y, height := t.rowGeometry(row)
		x := t.lineSizeToColumn(col, row, size, pad).Width
		info.Positions[i] = fyne.AccessibilityTextPosition{Position: origin.Add(fyne.NewPos(x, y+pad)), Height: height, Line: row}
	}
	return info
}

// AccessibilitySelectText uses the same selection as pointer selection and Copy.
// Synthetic paragraph separators and list markers map to their content boundary.
// Since: 2.9
func (t *RichText) AccessibilitySelectText(start, end int) {
	sel := t.selectionWidget()
	if sel == nil {
		return
	}
	_, offsets, _ := t.accessibilityDocument()
	start, end = min(max(start, 0), len(offsets)-1), min(max(end, 0), len(offsets)-1)
	sel.selectRow, sel.selectColumn = t.accessibilityRowColumn(offsets[start])
	sel.cursorRow, sel.cursorColumn = t.accessibilityRowColumn(offsets[end])
	sel.selecting = start != end
	t.highlight = &sel.selectable
	sel.Refresh()
	t.Refresh()
}

// AccessibilityScrollText reveals text without changing the selection.
// Since: 2.9
func (t *RichText) AccessibilityScrollText(start, end int, alignTop bool) {
	if t.scr == nil {
		return
	}
	info := t.AccessibilityText()
	offset := end
	if alignTop {
		offset = start
	}
	p := info.Positions[min(max(offset, 0), len(info.Positions)-1)]
	target := p.Position.Subtract(t.scr.Position()).Add(t.scr.Offset)
	if !alignTop {
		target.Y -= t.scr.Size().Height - p.Height
	}
	t.scr.ScrollToOffset(target)
}

func (t *RichText) accessibilityRowColumn(offset int) (line, column int) {
	row := max(0, sort.Search(len(t.rowBounds), func(i int) bool { return t.rowBounds[i].docBegin > offset })-1)
	if len(t.rowBounds) == 0 {
		return 0, 0
	}
	return row, min(max(offset-t.rowBounds[row].docBegin, 0), t.rowLength(row))
}

func (t *RichText) accessibilityDocument() (document fyne.AccessibilityTextInfo, sourceOffsets, documentOffsets []int) {
	raw := []rune(t.String())
	inserts := make(map[int]string)
	for row := 1; row < len(t.rowBounds); row++ {
		before, after := t.rowBounds[row-1], t.rowBounds[row]
		if len(before.segments) != 0 && len(after.segments) != 0 && before.docEnd == after.docBegin && before.segments[len(before.segments)-1] != after.segments[0] {
			inserts[after.docBegin] = "\n"
		}
	}
	for _, marker := range t.markersBetween(0, len(raw)) {
		inserts[marker.pos] += marker.text
	}
	var text strings.Builder
	var offsets []int
	reverse := make([]int, len(raw)+1)
	for i := 0; i <= len(raw); i++ {
		for _, r := range inserts[i] {
			offsets = append(offsets, i)
			text.WriteRune(r)
		}
		reverse[i] = len(offsets)
		if i < len(raw) {
			offsets = append(offsets, i)
			text.WriteRune(raw[i])
		}
	}
	offsets = append(offsets, len(raw))
	info := fyne.AccessibilityTextInfo{Text: text.String()}
	info.WordBoundaries = entryWordBoundaries(info.Text)
	for _, run := range t.accessibilityRuns() {
		run.Start, run.End = reverse[run.Start], reverse[run.End]
		if len(info.Runs) == 0 {
			run.Start = 0
		}
		info.Runs = append(info.Runs, run)
	}
	return info, offsets, reverse
}

func (t *RichText) accessibilityRuns() []fyne.AccessibilityTextRun {
	var runs []fyne.AccessibilityTextRun
	offset := 0
	for _, segment := range t.contentSegments() {
		length := utf8.RuneCountInString(segment.Textual())
		if length == 0 {
			continue
		}
		style := RichTextStyleInline
		switch s := segment.(type) {
		case *TextSegment:
			style = s.Style
		case *HyperlinkSegment:
			style.TextStyle, style.SizeName, style.ColorName = s.TextStyle, s.SizeName, theme.ColorNameHyperlink
		case *CodeBlockSegment:
			style = RichTextStyleCodeBlock
		}
		run := t.accessibilityStyle(style)
		run.Start, run.End = offset, offset+length
		offset += length
		if len(runs) != 0 {
			previous := runs[len(runs)-1]
			previous.Start, previous.End = run.Start, run.End
			if previous == run {
				runs[len(runs)-1].End = run.End
				continue
			}
		}
		runs = append(runs, run)
	}
	if len(runs) == 0 {
		runs = append(runs, t.accessibilityStyle(RichTextStyleInline))
	}
	return runs
}

func (t *RichText) accessibilityStyle(style RichTextStyle) fyne.AccessibilityTextRun {
	sizeName, colorName := style.SizeName, style.ColorName
	if sizeName == "" {
		sizeName = theme.SizeNameText
	}
	if colorName == "" {
		colorName = theme.ColorNameForeground
	}
	heading := style.headingLevel
	if sizeName == theme.SizeNameHeadingText {
		heading = 1
	}
	if sizeName == theme.SizeNameSubHeadingText {
		heading = 2
	}
	foreground, _ := color.NRGBAModel.Convert(t.Theme().Color(colorName, fyne.CurrentApp().Settings().ThemeVariant())).(color.NRGBA)
	return fyne.AccessibilityTextRun{Style: style.TextStyle, Size: t.Theme().Size(sizeName), Foreground: foreground, Alignment: style.Alignment, HeadingLevel: heading}
}

// AccessibilityText adds formatting from the same segments used by the editor.
// Since: 2.9
func (e *RichTextEntry) AccessibilityText() fyne.AccessibilityTextInfo {
	info := e.Entry.AccessibilityText()
	if !e.Password {
		info.Runs = e.richProvider().accessibilityRuns()
	}
	return info
}

// AccessibilitySetValue replaces rich content through RichTextEntry.SetText.
// Since: 2.9
func (e *RichTextEntry) AccessibilitySetValue(value string) {
	if !e.Disabled() {
		e.SetText(value)
	}
}
