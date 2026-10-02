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

Single and Exclude take precedence over a child list. Group and Transparent do
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
SelectText and ScrollText use rune offsets. Reading a snapshot does not move
keyboard focus. Run these helpers on the event thread, like other Fyne test APIs.

Issues reports these stable diagnostic codes, without copying UI text into its
messages:

* `unrepresented-metadata`: an Auto widget has metadata but no semantic node.
* `missing-semantics`: an Auto keyboard control has no Accessible interface.
* `unrepresented-focus`: current keyboard focus has no exposed semantic target.

These checks do not prove accessibility. Test native interaction and speech too.

The independent module in `test/testdata/accessibility` demonstrates a custom
toggle, a file picker and a named group with independent actions and reordering.
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

List, Table and Tree explicitly stop automatic renderer discovery. Their recycled
visual cells must not be confused with stable logical data items. This boundary
does not implement collection accessibility: keyed virtual children, realization,
scrolling and collection patterns remain a separate feature. Authors implementing
logical children today must own stable child objects and use the existing focus
and selection interfaces; a reused visual cell alone is not a logical identity.

This change does not implement arbitrary semantic merging or rich-text semantics.
