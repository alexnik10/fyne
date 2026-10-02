# Accessibility for custom components

This document describes the shared semantic model in the accessibility draft.
Windows UI Automation consumes it today. Other native adapters still need to be
migrated; passing these tests does not establish screen-reader support on every
Fyne platform.

## Composition defaults

A custom `BaseWidget` that composes standard controls does not need to repeat
their accessibility implementations. If it has no `fyne.Accessible` interface,
the builder traverses its renderer, using the same cached renderer as layout and
input. Buttons, entries, labels and other semantic descendants keep their actual
objects, actions, state and focus. Drawing primitives do not acquire semantics
merely because the builder visits them.

An `Accessible` object is a semantic boundary by default. Its name, role and
capabilities describe the whole element; its renderer internals are not exposed.
Implement capability interfaces such as `AccessibleActionable` or
`AccessibleToggler` using the same commands as keyboard and pointer interaction.
Semantic support does not implement keyboard interaction for a custom widget.

`AccessibleChildren` replaces automatic child discovery. Its order is the reading
order; a nil or empty result means there are no semantic children. Logical child
positions are relative to their owner. Keep their objects stable across refreshes
and reordering. Do not add the renderer children to this list automatically.

## Explicit composition

`BaseWidget.SetAccessibilityMode` provides these modes. Other CanvasObjects can
implement `AccessibleComposition` directly.

| Mode | Own node | Children |
| --- | --- | --- |
| Auto | If Accessible | Explicit children first; Container.Objects; renderer only for widgets without Accessible |
| Transparent | Omitted | Explicit children, otherwise ordinary container/renderer children |
| Group | Accessible semantics, or a container node with AccessibilityInfo | Explicit children, otherwise ordinary container/renderer children |
| Single | Only if Accessible | None, including when AccessibleChildren exists |
| Exclude | Omitted | Entire subtree excluded |

AccessibleElements takes precedence over AccessibleChildren; both take precedence
over renderer discovery. Single and Exclude take precedence over either child list. Group and Transparent do
not override an explicit empty child list. No mode infers actions, combines text
values or merges the state of several controls. Existing Accessible objects keep
their Auto boundary, including when a custom type embeds a standard widget.

For example, a named group preserving independent child commands:

```go
group := container.NewAccessibilityGroup("Choose file", filePicker)
```

An author of a composed BaseWidget can use:

```go
component.SetAccessibilityMode(fyne.AccessibilityGroup)
component.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Connection settings"})
```

For an ordinary CanvasObject use `container.NewAccessibility(mode, content)`.
Its public Content can be replaced followed by Refresh. The wrapper retains the
original child objects and does not impersonate their focus or action interfaces.
Its layout fills the wrapper with the content, like a single-child stack.

Exclude changes only accessibility. It does not hide rendering or remove keyboard
interaction. Use it for decoration, not as a way to disable a control. A keyboard
target hidden behind Exclude or Single needs an equivalent accessible command or
an explicitly designed focus relationship.

SetAccessibilityInfo supplies metadata; it does not invent a role or an action.
Use Group mode to name a component containing other controls, or implement
Accessible for an element with its own meaning. Names never incorporate entry
values automatically, and protected text remains masked in snapshots.

## Metadata: inherit, override and clear

A zero `AccessibilityInfo` inherits all metadata. Nonempty strings and `true`
values are explicit overrides. To supply an empty string or `false`, set the
corresponding `NameSet`, `DescriptionSet`, `RequiredSet` or `InvalidSet` flag:

```go
entry.SetAccessibilityInfo(fyne.AccessibilityInfo{
    Description: "", DescriptionSet: true,
    Required: false, RequiredSet: true,
    Invalid: false, InvalidSet: true,
})
// Restore form labels, hints and widget validation metadata:
entry.SetAccessibilityInfo(fyne.AccessibilityInfo{})
```

Precedence is fallback label, outer-to-inner contextual metadata, then the
widget's own metadata. Only supplied fields override earlier values. A form with
an empty label leaves the control's own label intact (for example a Check).
Entry supplies its validation error by default; its explicit description or
InvalidSet overrides that metadata. These overrides do not change validation,
required-field checks, visible labels or submission behavior. NameSet also permits
an explicitly empty accessible name; use a meaningful name for interactive controls.

## Identity, geometry and scope

Within one tree, attached child objects retain IDs when reordered or wrapped in
another component. Hidden or excluded subtrees retain the last known attachment
structure without constructing hidden renderers. A detached object loses its ID
once the builder observes the removal. A different object never inherits it.
Renderer recreation cannot preserve an object that the component itself replaces;
keep interactive child objects on the component, not only in a renderer instance.

Automatic order follows container/renderer order, not a geometric guess at visual
reading order. Supply AccessibleChildren when these orders differ. Cycles and
duplicate references are ignored; a child must have one logical parent.

Ancestor clipping affects exposed bounds, including text viewports. Layout
positions remain separate so clipping cannot shift caret geometry. Offscreen
content remains attached. The current input-capturing overlay determines which
nodes can receive accessibility commands; background nodes retain their identity
but cannot be invoked. All commands revalidate the current tree.

## Keyed logical elements

`AccessibleElements` returns a complete preorder slice of `AccessibilityElement`.
Each nonempty Key is unique within its owner. Parent names an earlier key, or is
empty for a direct child of the owner. Object supplies Accessible semantics and
capabilities, with geometry relative to the owner. It need not be rendered, and
can be replaced on every snapshot: identity is **owner + key**. Commands dispatch
to the current Object after rebuilding. Its renderer, child interfaces and
composition mode are not traversed. Bounds are clipped to the owner and ancestors.
The owner should expose its own Accessible semantics or use Group mode.

* Keep a key for the same model item across edits, reordering and cell recycling.
* Return Hidden descriptors, including descendants, to retain identities while
  suppressing navigation and commands. Hidden descriptors may omit Object. An
  exposed child of a hidden parent is also suppressed.
* Omit a deleted key. Returning it after an observed removal creates a new ID.
  A remove-and-replace between snapshots is indistinguishable without a new key;
  include a generation in the key when these are different model items.
* Offscreen is different from hidden: return its semantic object with empty or
  clipped bounds and AccessibleScrollItem to reveal it without selecting/focusing.
* AccessibleActiveElement maps the owner's **actual canvas focus** to a key.
  It does not establish a separate keyboard or screen-reader focus model.

Invalid keys, duplicates, forward/missing parents and missing Accessible objects
are skipped and reported as `invalid-element`. A hidden or excluded owner retains
its last known keys without calling its provider. Deletions while the owner itself
is hidden are therefore observed when it is exposed again. Single removes the
child contract instead of retaining hidden children.

### Tree

Tree uses TreeNodeID as the logical key. Its semantics include Tree/TreeItem roles,
sibling position/count, hierarchy level, expansion, optional single selection,
actual keyboard highlight and scrolling. Collapsing an ancestor moves a hidden
keyboard highlight to that ancestor. Refresh reconciles removed selections and
highlights with the model. Scrolling alone does not change selection or focus.
A custom nonempty Root is exposed as the first item and is always expanded, in
accordance with Tree's existing Root behavior.

Provide names without creating visual cells:

```go
tree.DescribeNode = func(id widget.TreeNodeID) fyne.AccessibilityInfo {
    return fyne.AccessibilityInfo{Name: model[id].Title}
}
tree.SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "Project files"})
```

Without DescribeNode the name is TreeNodeID, matching NewTreeWithStrings. The
snapshot enumerates model IDs, including collapsed branches, but never calls
CreateNode/UpdateNode or allocates visual cells. ChildUIDs must therefore support
model enumeration without loading UI objects. Cost is linear in model size in
the shared builder; this is not a lazy/paged semantic collection API. UIA snapshot
publication has additional native costs. Very large or remotely loaded models
need a separate paging/realization design and performance measurements.

Windows maps leaves to ExpandCollapse LeafNode, branches to Expanded/Collapsed,
and exposes Selection/SelectionItem, Level, PositionInSet, SizeOfSet and ScrollItem.
Collapsed descendants are omitted; scrolled-off descendants remain discoverable.
This follows the [Microsoft TreeItem contract](https://learn.microsoft.com/en-us/windows/win32/winauto/uiauto-supporttreeitemcontroltype).
All model nodes have semantics, so UIA VirtualizedItem is not advertised merely
because renderer cells are recycled.

## Public tests

Use a laid-out test canvas and keep one inspector across updates:

```go
semantics := test.NewAccessibilityTree(window.Canvas())
node, present := semantics.Node(button)
if !present || !node.Invoke {
    t.Fatal("button is missing or cannot be activated")
}
if !semantics.Perform(node.ID, test.AccessibilityActivate, "", 0) {
    t.Fatal("activation was rejected")
}
```

Snapshot returns independent semantic data including IDs, parents, metadata,
states, capabilities, focus, selection and text. Perform rebuilds before dispatch;
an old ID cannot act on removed, excluded, disabled or background controls.
Use `semantics.Element(owner, key)` for keyed lookup and
`AccessibilityScrollIntoView` for item scrolling. Level, SetPosition and SetSize
are available on public snapshots. SelectText and ScrollText use rune offsets. Reading a snapshot does not move
keyboard focus. Run these helpers on the event thread, like other Fyne test APIs.

Issues reports these stable diagnostic codes, without copying UI text into its
messages:

* `unrepresented-metadata`: an Auto widget has metadata but no semantic node.
* `missing-semantics`: an Auto keyboard control has no Accessible interface.
* `unrepresented-focus`: current keyboard focus has no exposed semantic target.
* `invalid-element`: malformed keyed logical elements.

These checks do not prove accessibility. Test native interaction and speech too.

The independent module in `test/testdata/accessibility` demonstrates a custom
toggle, a file picker, a named group, keyed Tree commands and explicit metadata.
It imports only public packages, and CI tests it on Linux and Windows:

```sh
cd test/testdata/accessibility
go test -mod=readonly -race -tags ci ./...
```

## Built-in boundaries and remaining collection work

Entry exposes its ActionItem explicitly while retaining its text, selection,
placeholder and caret as internal rendering. The password revealer supplies its
own accessible command, keyboard focus and activation; it follows Entry's disabled
state and restores editing focus when activated.

List and Table still explicitly stop automatic renderer discovery. Their recycled
visual cells must not be confused with stable logical data items. Their positional
IDs are not stable model keys; their full collection semantics and paging remain
separate work. Tree is the first consumer of keyed logical elements.

This change does not implement arbitrary semantic merging or rich-text semantics.
