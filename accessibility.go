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
	AccessibleRoleComboBox  AccessibleRole = "combobox"
	AccessibleRoleRadio     AccessibleRole = "radio"
	AccessibleRoleList      AccessibleRole = "list"
	AccessibleRoleListItem  AccessibleRole = "listitem"
	AccessibleRoleTree      AccessibleRole = "tree"
	AccessibleRoleTreeItem  AccessibleRole = "treeitem"
)

// Accessible interface should be implemented for a widget that should be accessible
//
// Since: 2.8
type Accessible interface {
	AccessibilityLabel() string
	AccessibilityRole() AccessibleRole
}

// AccessibilityInfo supplies optional metadata, independently of rendering.
// Nonzero fields override inherited metadata. Set the corresponding Set flag to
// override with an empty string or false. A zero struct inherits all fields.
// Password values must never be exported.
//
// Since: 2.9
type AccessibilityInfo struct {
	Name, Description                                string
	Required, Invalid                                bool
	NameSet, DescriptionSet, RequiredSet, InvalidSet bool
}

// AccessibleDescribed provides an explicit name and description.
//
// Since: 2.9
type AccessibleDescribed interface {
	AccessibilityInfo() AccessibilityInfo
}

// AccessibilityMode controls how an object composes its accessibility subtree.
// It does not change rendering, keyboard traversal or pointer handling.
//
// Since: 2.9
type AccessibilityMode uint8

const (
	// AccessibilityAuto preserves Accessible nodes and explicit logical children.
	// Containers expose their contents. Other widgets without Accessible are
	// transparent: their renderer is traversed until semantic boundaries are met.
	AccessibilityAuto AccessibilityMode = iota
	// AccessibilityTransparent omits the object's own semantics, retaining children.
	AccessibilityTransparent
	// AccessibilityGroup exposes a node and its children. Without Accessible, the
	// node has the container role and uses AccessibilityInfo for its metadata.
	AccessibilityGroup
	// AccessibilitySingle exposes only this object's own Accessible semantics.
	// It never merges names, states or commands from descendants. Without Accessible
	// it exposes no node, and can guard renderer internals awaiting semantic support.
	AccessibilitySingle
	// AccessibilityExclude omits this object and its entire subtree.
	AccessibilityExclude
)

// AccessibleComposition optionally overrides automatic semantic composition.
// Single and Exclude stop child traversal, even when explicit children exist.
// Transparent and Group use explicit children when supplied, otherwise renderer
// children. AccessibleCollection takes precedence over AccessibleElements, which
// takes precedence over AccessibleChildren. Auto
// treats an Accessible without explicit children as a leaf.
//
// Since: 2.9
type AccessibleComposition interface {
	AccessibilityMode() AccessibilityMode
}

// AccessibleChildren defines logical children in reading order. Positions are
// relative to this object, as with Container.Objects. Implementations must keep
// child objects stable across refreshes, or use AccessibleCollection for model keys.
// Decorative renderer objects are omitted.
// This list replaces automatic children; even a nil or empty list is definitive.
// An accessible object without this interface is a semantic leaf in Auto mode.
//
// Since: 2.9
type AccessibleChildren interface {
	AccessibilityChildren() []CanvasObject
}

// AccessibilityElement describes a logical element independently of recycled
// renderer cells. Key is nonempty and unique within its owner. Parent is the key
// of its logical parent, or empty for a direct child of the owner. Object supplies
// Accessible semantics and capabilities; its position is relative to the owner,
// even for nested elements. It need not be rendered or stable across snapshots.
// Its children and composition mode are ignored: Parent defines this hierarchy.
// Hidden retains identity but excludes the element and its descendants from
// navigation and commands. Object may be nil while Hidden. Offscreen elements
// remain exposed with empty bounds, so they can be scrolled into view.
//
// Since: 2.9
type AccessibilityElement struct {
	Key, Parent string
	Object      CanvasObject
	Hidden      bool
	// Generation distinguishes successive lifetimes of the same key. Change it
	// when an item is removed and replaced between adapter snapshots. Zero is
	// valid when the owner never reuses keys or every removal is observed.
	Generation uint64
}

// AccessibilityCollectionView optionally limits eagerly published semantics to
// the viewport and selection. Adapters can request any other non-hidden element
// through Element, without scrolling, selecting or creating a renderer cell.
// Index returns the logical parent and zero-based sibling index for a live key.
// ViewportKeys and SelectedKeys must contain only live, non-hidden keys; parents
// are included by the adapter. The source is used only on the Fyne event thread.
// Model changes must be followed by Refresh before querying the collection.
//
// Since: 2.9
type AccessibilityCollectionView interface {
	AccessibilityCollection
	// Revision changes whenever keys, hierarchy or generations change.
	Revision() uint64
	ViewportKeys() []string
	SelectedKeys() []string
	Index(key string) (parent string, index int, ok bool)
}

// AccessibilityCollection separates model topology from per-item semantics.
// The empty parent denotes the owner; all other keys are nonempty and unique
// within it. ChildKey returns the child at a zero-based index, or empty for an
// invalid index. Element returns false for a missing key. Its Key and Parent
// must match the topology. Include hidden descendants to retain their identities.
//
// A source is used synchronously on the Fyne event thread, until the next model
// mutation. Querying it must not render cells, move focus or change selection.
// Adapters may request individual items or enumerate the whole source. The
// AccessibilityCollectionView enables bounded, on-demand publication.
//
// Since: 2.9
type AccessibilityCollection interface {
	ChildCount(parent string) int
	ChildKey(parent string, index int) string
	Element(key string) (AccessibilityElement, bool)
}

// AccessibleCollection provides indexed logical children and lookup by model key.
// It takes precedence over AccessibleElements and AccessibleChildren, including
// when it returns nil. Single and Exclude still stop traversal. Objects returned
// by the source follow the same rules as AccessibilityElement.
//
// Since: 2.9
type AccessibleCollection interface {
	AccessibilityCollection() AccessibilityCollection
}

// AccessibleElements supplies a complete preorder snapshot of keyed logical
// elements, including hidden elements whose IDs should be retained. Keys belong
// to this owner, not the current Object or its renderer. Omitted keys are removed;
// if returned after an observed removal they get new IDs. To replace an item
// without an intervening snapshot, use a new key or Generation. Prefer
// AccessibleCollection for indexed access without constructing a complete slice.
// This interface takes precedence over AccessibleChildren and renderer traversal.
// Single and Exclude still stop traversal. Methods run on the Fyne event thread.
// No renderers are created for the supplied objects. Bounds are clipped to owner.
//
// Since: 2.9
type AccessibleElements interface {
	AccessibilityElements() []AccessibilityElement
}

// AccessibleActiveElement maps an owner's actual keyboard focus to an exposed
// key from AccessibleCollection or AccessibleElements. Empty means the owner
// itself. This takes precedence over AccessibleActiveDescendant and must not
// represent a separate reading cursor.
//
// Since: 2.9
type AccessibleActiveElement interface {
	AccessibilityActiveElement() string
}

// AccessibleHierarchy describes an item's one-based level, position and count
// among siblings. Zero means unknown. This is independent of selection support.
//
// Since: 2.9
type AccessibleHierarchy interface {
	AccessibilityHierarchy() (level, position, count int)
}

// AccessibleScrollItem reveals an item without changing selection or focus.
// Return false if it is detached or cannot be revealed.
//
// Since: 2.9
type AccessibleScrollItem interface {
	AccessibilityScrollIntoView() bool
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

// AccessibilityTextPosition describes an insertion point in widget coordinates.
// Positions include the end of the document, so there is one more than the
// number of runes. Line identifies the rendered line, including soft wrapping.
//
// Since: 2.9
type AccessibilityTextPosition struct {
	Position Position
	Height   float32
	Line     int
}

// AccessibilityTextInfo is a snapshot of editable plain text. All offsets count
// runes, not bytes or UTF-16 units. A collapsed selection describes the caret.
// Protected controls must supply only masking characters, never their contents.
// Revision changes on text edits, including replacement with identical text.
// Positions may be omitted when text geometry is unavailable.
//
// Since: 2.9
type AccessibilityTextInfo struct {
	Text                                string
	Caret, SelectionStart, SelectionEnd int
	Revision                            uint64
	Positions                           []AccessibilityTextPosition
	ViewportPosition                    Position
	ViewportSize                        Size

	// WordBoundaries supplies ordered rune offsets including zero and the end
	// of the text. Adapters use these instead of independently guessing word
	// breaks. Nil permits a platform fallback. Protected text must not reveal
	// the word boundaries of the original value.
	WordBoundaries []int
}

// AccessibleText exposes editable text, its caret and a single selection.
// Methods run on the Fyne event thread. Selection offsets are rune offsets and
// must be clamped to the current text; selecting must not change the text.
//
// Since: 2.9
type AccessibleText interface {
	AccessibilityText() AccessibilityTextInfo
	AccessibilitySelectText(start, end int)
}

// AccessibleTextScroller can reveal a text range without moving the caret.
//
// Since: 2.9
type AccessibleTextScroller interface {
	AccessibilityScrollText(start, end int, alignTop bool)
}

// AccessibleSelection describes a container whose logical children can be selected.
// Required means the last selected item cannot be removed through selection actions.
//
// Since: 2.9
type AccessibleSelection interface {
	AccessibilitySelection() (multiple, required bool)
}

// AccessibleSelectable exposes an item's selection owner, state and one-based
// position in its set. Select replaces the selection; deselect removes this item.
// Commands return false when the item is detached, disabled or cannot be removed.
//
// Since: 2.9
type AccessibleSelectable interface {
	AccessibilitySelectionItem() (owner CanvasObject, selected bool, position, count int)
	AccessibilitySelect(mode AccessibilitySelectionMode) bool
}

// AccessibleExpandable exposes disclosure without simulating pointer input.
//
// Since: 2.9
type AccessibleExpandable interface {
	AccessibilityExpanded() bool
	AccessibilitySetExpanded(bool)
}

// AccessibleActiveDescendant maps a compound control's actual keyboard focus to
// the child currently receiving its keyboard commands. It is not a reading cursor.
// Return nil when the compound control itself should be reported as focused.
//
// Since: 2.9
type AccessibleActiveDescendant interface {
	AccessibilityActiveDescendant() CanvasObject
}

// AccessibleFocusHandler routes focus for a logical child through its keyboard
// owner. It must use the real canvas focus, never maintain an independent cursor.
//
// Since: 2.9
type AccessibleFocusHandler interface {
	AccessibilityFocusable() bool
	AccessibilityFocus() bool
}

// AccessibleOverlayOwner identifies a control whose popup is represented by its
// own semantic children. While this overlay captures input, adapters expose only
// that owner and its descendants, preserving the owner's identity and metadata.
// Return nil for an ordinary overlay.
//
// Since: 2.9
type AccessibleOverlayOwner interface {
	AccessibilityOverlayOwner() CanvasObject
}

// AccessibilitySelectionMode identifies a selection command.
//
// Since: 2.9
type AccessibilitySelectionMode int

// Selection commands distinguish replacing a selection from extending it.
const (
	AccessibilitySelectReplace AccessibilitySelectionMode = iota
	AccessibilitySelectAdd
	AccessibilitySelectRemove
)
