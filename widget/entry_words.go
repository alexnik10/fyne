package widget

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// entryWordBoundaries is shared by Windows Ctrl+Arrow and accessibility text
// snapshots. Punctuation runs are separate units, trailing spaces belong to the
// preceding unit, and hard line breaks remain explicit navigation stops.
func entryWordBoundaries(text string) []int {
	runes := []rune(text)
	boundaries := []int{0}
	for i := 1; i < len(runes); i++ {
		before, after := runes[i-1], runes[i]
		if before == '\n' || after == '\n' ||
			(!unicode.IsSpace(after) && (unicode.IsSpace(before) || isWordSeparator(before) != isWordSeparator(after))) {
			boundaries = append(boundaries, i)
		}
	}
	if len(runes) != 0 {
		boundaries = append(boundaries, len(runes))
	}
	return boundaries
}

// A word movement must land on the same boundaries as TextUnit_Word. The old
// asymmetric movement stopped at word ends to the right and starts to the left,
// skipping whole words when a screen reader expanded the punctuation at a stop.
func (e *Entry) moveWordWindows(right bool) {
	text := e.Text
	if e.Password {
		text = strings.Repeat(passwordChar, utf8.RuneCountInString(text))
	}
	boundaries := entryWordBoundaries(text)
	current, target := e.CursorTextOffset(), 0
	if right {
		target = boundaries[len(boundaries)-1]
		for _, boundary := range boundaries {
			if boundary > current {
				target = boundary
				break
			}
		}
	} else {
		for _, boundary := range boundaries {
			if boundary >= current {
				break
			}
			target = boundary
		}
	}
	e.setFieldsAndRefresh(func() {
		e.CursorRow, e.CursorColumn = e.rowColFromTextPos(target)
		e.syncSelectable()
	})
}
