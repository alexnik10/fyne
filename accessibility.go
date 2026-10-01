package fyne

// AccessibleRole describes the different roles an accessible element can take.
//
// Since: 2.8
type AccessibleRole string

// Known values for [AccessibleRole].
const (
	AccessibleRoleButton    AccessibleRole = "button"
	AccessibleRoleContainer AccessibleRole = "container"
	AccessibleRoleLink      AccessibleRole = "link"
	AccessibleRoleText      AccessibleRole = "text"
	AccessibleRoleCheck     AccessibleRole = "check"
	AccessibleRoleEntry     AccessibleRole = "entry"
	AccessibleRoleSlider    AccessibleRole = "slider"
	AccessibleRoleDialog    AccessibleRole = "dialog"
)

// Accessible interface should be implemented for a widget that should be accessible
//
// Since: 2.8
type Accessible interface {
	AccessibilityLabel() string
	AccessibilityRole() AccessibleRole
}

// AccessibilityInfo supplies optional metadata, independently of rendering.
// Empty Name uses AccessibilityLabel. Password values must never be exported.
//
// Since: 2.9
type AccessibilityInfo struct {
	Name, Description string
	Required, Invalid bool
}

// AccessibleDescribed provides an explicit name and description.
//
// Since: 2.9
type AccessibleDescribed interface {
	AccessibilityInfo() AccessibilityInfo
}

// AccessibleChildren defines logical children in reading order. Positions are
// relative to this object, as with Container.Objects. Implementations must keep
// child objects stable across refreshes. Decorative renderer objects are omitted.
// An accessible object without this interface is a semantic leaf.
//
// Since: 2.9
type AccessibleChildren interface {
	AccessibilityChildren() []CanvasObject
}

// AccessibleChildDescriber supplies contextual metadata for descendants (for
// example labels, hints and validation errors belonging to form fields).
// Explicit metadata on the child takes precedence over a contextual name.
//
// Since: 2.9
type AccessibleChildDescriber interface {
	AccessibilityChildInfo(CanvasObject) AccessibilityInfo
}

// AccessibleActionable supports activation without simulating pointer input.
// It must use the same command as keyboard/pointer activation and honour Disabled.
//
// Since: 2.9
type AccessibleActionable interface {
	AccessibilityActivate()
}

// AccessibleToggler exposes a two-state toggle and its command.
//
// Since: 2.9
type AccessibleToggler interface {
	AccessibilityChecked() bool
	AccessibilityToggle()
}

// AccessibleValue exposes a string value. A protected value returns an empty
// string. Setters run on the Fyne event thread and must honour read-only state.
//
// Since: 2.9
type AccessibleValue interface {
	AccessibilityValue() (value string, readOnly, protected bool)
	AccessibilitySetValue(string)
}

// AccessibleRange exposes a numeric value, bounds and smallest increment.
//
// Since: 2.9
type AccessibleRange interface {
	AccessibilityRange() (value, minimum, maximum, step float64)
	AccessibilitySetRangeValue(float64)
}
