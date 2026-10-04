package dialog

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
)

func (b *colorButton) AccessibilityLabel() string           { return colorToString(b.color) }
func (*colorButton) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleButton }
func (b *colorButton) AccessibilityActivate()               { b.Tapped(nil) }
func (b *colorButton) FocusGained()                         { b.focused = true; b.Refresh() }
func (b *colorButton) FocusLost()                           { b.focused = false; b.Refresh() }
func (*colorButton) TypedRune(rune)                         {}
func (b *colorButton) TypedKey(event *fyne.KeyEvent) {
	if event.Name == fyne.KeySpace || event.Name == fyne.KeyEnter || event.Name == fyne.KeyReturn {
		b.AccessibilityActivate()
	}
}

func (*colorPreview) AccessibilityLabel() string {
	return lang.X("accessibility.color.preview", "Colour preview")
}
func (*colorPreview) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleText }
func (p *colorPreview) AccessibilityValue() (value string, readOnly, protected bool) {
	text := func(c color.Color) string {
		if c == nil {
			return ""
		}
		return colorToString(c)
	}
	return lang.X("accessibility.color.comparison", "Previous {{.Previous}}, current {{.Current}}", map[string]any{"Previous": text(p.previous), "Current": text(p.current)}), true, false
}
func (*colorPreview) AccessibilitySetValue(string) {}

func colorChannelName(short string) string {
	switch short {
	case "R":
		return lang.X("accessibility.color.red", "Red")
	case "G":
		return lang.X("accessibility.color.green", "Green")
	case "B":
		return lang.X("accessibility.color.blue", "Blue")
	case "H":
		return lang.X("accessibility.color.hue", "Hue")
	case "S":
		return lang.X("accessibility.color.saturation", "Saturation")
	case "L":
		return lang.X("accessibility.color.lightness", "Lightness")
	case "A":
		return lang.X("accessibility.color.alpha", "Opacity")
	}
	return short
}

// UIA replacement is user input, just like typing, and must update the model.
func (e *userChangeEntry) AccessibilitySetValue(value string) {
	if e.Disabled() {
		return
	}
	e.userTyped = true
	e.SetText(value)
	e.userTyped = false
}

func (f *fileDialog) accessibilityFileKey(id int) string {
	if uri, ok := f.getDataItem(id); ok {
		return uri.String()
	}
	return ""
}

func (f *fileDialog) accessibilityFileInfo(id int) fyne.AccessibilityInfo {
	uri, ok := f.getDataItem(id)
	if !ok {
		return fyne.AccessibilityInfo{}
	}
	name := uri.Name()
	if f.dir != nil && id == 0 && len(uri.Path()) < len(f.dir.Path()) {
		name = lang.X("file.parent", "Parent")
	}
	description := lang.X("accessibility.file", "File")
	if _, folder := uri.(fyne.ListableURI); folder {
		description = lang.X("accessibility.folder", "Folder")
	}
	return fyne.AccessibilityInfo{Name: name, Description: description}
}

func (e *userChangeEntry) AccessibilitySetValueChecked(value string) bool {
	if e.Disabled() || (e.accessibilityValidate != nil && e.accessibilityValidate(value) != nil) {
		return false
	}
	e.AccessibilitySetValue(value)
	return true
}
