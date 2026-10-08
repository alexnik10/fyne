package dialog

import (
	"image/color"

	"fyne.io/fyne/v2"
	col "fyne.io/fyne/v2/internal/color"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
)

func (b *colorButton) AccessibilityLabel() string           { return accessibleColorName(b.color) }
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
		return accessibleColorName(c)
	}
	return lang.X("accessibility.color.comparison", "Previous {{.Previous}}, current {{.Current}}", map[string]any{"Previous": text(p.previous), "Current": text(p.current)}), true, false
}
func (*colorPreview) AccessibilitySetValue(string) {}

// Known palette colours keep their authored names. Custom colours use exact
// channel values rather than an unreliable nearest named colour.
func accessibleColorName(c color.Color) string {
	r, g, b, a := col.ToNRGBA(c)
	hex := colorToString(c)
	name := ""
	switch hex {
	case "#f44336":
		name = lang.X("accessibility.color.red", "Red")
	case "#ff9800":
		name = lang.X("accessibility.color.orange", "Orange")
	case "#ffeb3b":
		name = lang.X("accessibility.color.yellow", "Yellow")
	case "#8bc34a":
		name = lang.X("accessibility.color.green", "Green")
	case "#296ff6":
		name = lang.X("accessibility.color.blue", "Blue")
	case "#9c27b0":
		name = lang.X("accessibility.color.purple", "Purple")
	case "#795548":
		name = lang.X("accessibility.color.brown", "Brown")
	case "#ffffff":
		name = lang.X("accessibility.color.white", "White")
	case "#cccccc":
		name = lang.X("accessibility.color.veryLightGrey", "Very light grey")
	case "#aaaaaa":
		name = lang.X("accessibility.color.lightGrey", "Light grey")
	case "#808080":
		name = lang.X("accessibility.color.grey", "Grey")
	case "#555555":
		name = lang.X("accessibility.color.darkGrey", "Dark grey")
	case "#333333":
		name = lang.X("accessibility.color.veryDarkGrey", "Very dark grey")
	case "#000000":
		name = lang.X("accessibility.color.black", "Black")
	}
	if name == "" {
		name = lang.X("accessibility.color.rgb", "Red {{.Red}}, green {{.Green}}, blue {{.Blue}}", map[string]any{"Red": r, "Green": g, "Blue": b})
	}
	if a != 255 {
		name = lang.X("accessibility.color.withOpacity", "{{.Colour}}, opacity {{.Opacity}}%", map[string]any{"Colour": name, "Opacity": (int(a)*100 + 127) / 255})
	}
	return lang.X("accessibility.color.namedHex", "{{.Colour}}, {{.Hex}}", map[string]any{"Colour": name, "Hex": hex})
}

// List view is a toggle with a stable name: pressed means list, released grid.
type fileViewButton struct {
	widget.Button
	picker *fileDialog
}

func (b *fileViewButton) AccessibilityChecked() bool { return b.picker.view == ListView }
func (b *fileViewButton) AccessibilityToggle()       { b.AccessibilityActivate() }

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
