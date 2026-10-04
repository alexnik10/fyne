package widget

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/mattn/go-runewidth"
)

// AccessibilityLabel leaves the document name to application metadata.
// Since: 2.9
func (*TextGrid) AccessibilityLabel() string { return "" }

// AccessibilityRole identifies a read-only text document.
// Since: 2.9
func (*TextGrid) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleDocument }

// AccessibilityText exposes model text, styles and positions without renderer cells.
// Line numbers and whitespace decorations are not part of the document.
// Since: 2.9
func (t *TextGrid) AccessibilityText() fyne.AccessibilityTextInfo {
	info := fyne.AccessibilityTextInfo{ReadOnly: true, SelectionDisabled: true, ViewportSize: t.Size()}
	if t.scroller != nil {
		info.ViewportPosition, info.ViewportSize = t.scroller.Position(), t.scroller.Size()
	}
	var text []rune
	cellSize := fyne.Size{}
	if t.content != nil {
		cellSize = t.content.cellSize
	}
	position := func(row, column int) fyne.AccessibilityTextPosition {
		if t.ShowLineNumbers && t.content != nil {
			column += t.content.lineNumberWidth() + 1
		}
		p := fyne.NewPos(float32(column)*cellSize.Width, float32(row)*cellSize.Height)
		if t.scroller != nil {
			p = p.Subtract(t.scroller.Offset).Add(t.scroller.Position())
		}
		return fyne.AccessibilityTextPosition{Position: p, Height: cellSize.Height, Line: row}
	}
	for row, data := range t.Rows {
		next := 0
		for column, cell := range data.Cells {
			if column < next {
				continue
			}
			info.Positions = append(info.Positions, position(row, column))
			style := cell.Style
			if style == nil {
				style = data.Style
			}
			run := t.accessibilityTextGridRun(style, len(text))
			info.Runs = append(info.Runs, run)
			text = append(text, cell.Rune)
			if cell.Rune == '\t' {
				next = nextTab(column, t.tabWidth())
			} else {
				next = column + max(1, runewidth.RuneWidth(cell.Rune))
			}
		}
		if row < len(t.Rows)-1 {
			info.Positions = append(info.Positions, position(row, len(data.Cells)))
			info.Runs = append(info.Runs, t.accessibilityTextGridRun(data.Style, len(text)))
			text = append(text, '\n')
		}
	}
	row, column := max(0, len(t.Rows)-1), 0
	if len(t.Rows) > 0 {
		column = len(t.Rows[row].Cells)
	}
	info.Positions = append(info.Positions, position(row, column))
	info.Text = string(text)
	info.WordBoundaries = entryWordBoundaries(info.Text)
	return info
}

func (t *TextGrid) accessibilityTextGridRun(style TextGridStyle, start int) fyne.AccessibilityTextRun {
	th := t.Theme()
	run := fyne.AccessibilityTextRun{Start: start, End: start + 1, Size: th.Size(theme.SizeNameText), Style: fyne.TextStyle{Monospace: true}}
	foreground := th.Color(theme.ColorNameForeground, fyne.CurrentApp().Settings().ThemeVariant())
	if style != nil {
		run.Style = style.Style()
		run.Style.Monospace = true
		if style.TextColor() != nil {
			foreground = style.TextColor()
		}
	}
	run.Foreground, _ = color.NRGBAModel.Convert(foreground).(color.NRGBA)
	return run
}

// AccessibilitySelectText does not create editing or selection behavior in TextGrid.
// Since: 2.9
func (*TextGrid) AccessibilitySelectText(int, int) {}

// AccessibilityScrollText reveals a range without creating a keyboard caret.
// Since: 2.9
func (t *TextGrid) AccessibilityScrollText(start, end int, alignTop bool) {
	if t.scroller == nil {
		return
	}
	info := t.AccessibilityText()
	offset := end
	if alignTop {
		offset = start
	}
	p := info.Positions[min(max(0, offset), len(info.Positions)-1)]
	target := p.Position.Subtract(t.scroller.Position()).Add(t.scroller.Offset)
	if !alignTop {
		target.Y -= t.scroller.Size().Height - p.Height
	}
	t.scroller.AccessibilityScrollTo(target)
}

// AccessibilityScroll describes the text viewport.
// Since: 2.9
func (t *TextGrid) AccessibilityScroll() fyne.AccessibilityScrollInfo {
	if t.scroller == nil {
		return fyne.AccessibilityScrollInfo{}
	}
	return t.scroller.AccessibilityScroll()
}

// AccessibilityScrollTo scrolls without changing the text.
// Since: 2.9
func (t *TextGrid) AccessibilityScrollTo(offset fyne.Position) bool {
	if t.scroller == nil {
		return false
	}
	return t.scroller.AccessibilityScrollTo(offset)
}
