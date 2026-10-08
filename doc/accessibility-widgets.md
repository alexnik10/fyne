# Built-in widget coverage

This matrix describes the shared semantics and the opt-in Windows UI Automation
adapter on this branch. The remaining-widget increment still requires manual
NVDA/Narrator speech acceptance. It does not enable the other platform adapters.

| Component | Semantics and interaction |
| --- | --- |
| Button, Hyperlink | Named Invoke with the normal application command |
| Label, canvas.Text | Readable static text; `Decorative` omits canvas text already represented by its owner |
| Status labels | Opt-in `Label.SetAccessibilityLiveSetting` and `AccessibleLiveRegion`; Windows LiveSetting/LiveRegionChanged announce the current name without moving focus |
| Entry, PasswordEntry, RichTextEntry | Value and Text/Text2, actual editing/selection/focus; password redaction; formatting for rich text |
| RichText, TextGrid | Read-only text with ranges, formatting, geometry and scrolling; TextGrid exposes no unsupported selection |
| Check, CheckGroup | Toggle, checked state, named group, real checkbox focus |
| RadioGroup, Select | Single selection, stable options, keyboard focus; Select disclosure and popup scope |
| SelectEntry | Editable ComboBox with Value/Text, option selection, disclosure, Alt+Up/Down and popup keyboard navigation |
| DateEntry, Calendar | Validated date replacement, calendar disclosure, named month controls, Grid/Table weekday headers; one date in the Tab sequence, arrows/Home/End, Enter/Space selection and Escape in the date popup |
| Slider | RangeValue and string Value with bounds and step |
| ProgressBar | Read-only RangeValue and displayed Value; indeterminate ProgressBarInfinite/Activity expose activity state without a fictional percentage |
| Accordion | Stable section headers, Invoke and ExpandCollapse, Left/Right, content exposed only while open |
| Toolbar | Named toolbar and actual action buttons; use `ToolbarAction.SetAccessibilityInfo` for icon actions |
| Card | Title/subtitle group with content and described image |
| Icon, FileIcon, canvas.Image | Image role; name an Icon explicitly, use Image.AltText; unnamed images/icons stay decorative; FileIcon uses the file name |
| List, GridWrap, Tree, Table | Model keys, selection, focus, offscreen discovery/realization and scrolling; Tree hierarchy/disclosure and Table grid/header/editing contracts |
| Nested collection controls | `ItemElements`, `NodeElements`, `CellElements` provide model-backed controls below a row/node/cell; explicit focus routing required for custom keyboard interaction |
| AppTabs, DocTabs, menus | Selection, actions, actual keyboard navigation, close commands and popup scope |
| Scroll, Split | ScrollPattern for viewports; focusable split divider with bounded RangeValue and arrow/Home/End resizing |
| InnerWindow, MultipleWindows | Named inner groups, window buttons, keyboard movement/resizing through the existing callbacks and visible focus indicators |
| Navigation | Named Back/Forward commands and accessible page contents |
| Form, PopUp, built-in dialogs | Contextual labels/validation, named dialogs and existing overlay focus scope/restoration |
| File dialogs | Model-backed full names and folder descriptions in List/GridWrap, named locations and icon commands |
| Colour dialog | Keyboard/Invoke swatches, named RGB/HSL/opacity/hex fields, UIA replacement updates the colour model, read-only before/after preview; the wheel has equivalent HSL controls |
| Layout containers, padding, separators, drawing shapes, raster and gradients | Structural or decorative; expose semantic children rather than inventing controls. Custom meaningful drawings need an accessible wrapper |

## Model-backed children

The List/GridWrap `ItemElements`, Tree `NodeElements`, and Table `CellElements`
callbacks return lightweight descriptors for the current model record. They must
not inspect or create recycled renderer cells. Bind commands to the record's key,
not to a stored row index. Return fresh state on every query and call `Refresh`
after model changes.

Keys must be nonempty and unique within each record. Parents precede children;
`Parent` is a local child key, or empty for a direct child. Invalid duplicate or
cyclic descriptor sets are omitted. The NUL prefix is reserved in collection item
keys when these callbacks are used. `CollectionControlKey(recordKey, childKey)`
returns the full key for `test.AccessibilityTree.Element`.

Child positions (or the optional `AccessibilityElement.Position` override) are
relative to the row/node/cell, even for nested descriptors. Bounds are clipped by
the collection viewport. A replaced record receives new child lifetimes. Increment
a child's `Generation` when deleting and replacing its key between snapshots;
removal followed by reinsertion with no observed removal cannot be inferred from
identical descriptors. Positional item keys intentionally follow positions;
provide persistent model keys when rows can move.

A temporary `Button`, `Check` or `Entry` can supply commands/state, but its ordinary
`Focusable` methods do not make it a real keyboard target. Implement
`AccessibleFocusHandler` and the owner's active-element contract to route keyboard
interaction, or set `FocusTarget` to an actual stable rendered control. The toolkit
does not infer a custom cell's tab order or editor lifecycle. Standard Table text
editing continues to use the explicit `CellValue`/`OnCellChanged` contract.

The remaining demo's `internal/accessibilitydemo` grid shows one keyboard policy
for records containing a single checkbox. Canvas focus stays on the grid, whose
`AccessibilityActiveElement` points directly to the model checkbox. Its
`AccessibleFocusHandler` routes UIA focus back to the grid and the correct record.
Renderer checkboxes implement `TabStop() false`; pointer input also routes through
the model key. This keeps recycled cells out of the Tab sequence and preserves
the semantic focus identity while scrolling, toggling and reordering records.

## Manual acceptance

The remaining-widgets demo's bottom status label uses
`SetAccessibilityLiveSetting(fyne.AccessibilityLivePolite)`. Activate **Save
document**, wait for speech, then activate it again: both actions should announce
`Toolbar Save invoked`, with focus remaining on the button. Ordinary labels stay
silent on refresh. Initial exposure, reappearance after hiding and empty status
text do not trigger announcements. Multiple changes before one adapter snapshot
may be coalesced. Other native adapters do not yet implement live-region events.

With **Notification settings** open, Tab order is the section header, **Email**,
**Desktop**, then **More details**. Shift+Tab reverses that order. Collapsing the
section removes its controls from the focus chain. Static card/detail text and
images remain available through screen-reader reading/object navigation without
adding Tab stops.

On **Dates and choices**, Tab visits **Appointment date**, **Choose date**,
**Language or custom text**, **Show choices**, then the standalone calendar's
month buttons and one date. Shift+Tab reverses the order. Arrow keys move the
active date, including across month boundaries; Tab leaves the standalone date
grid in one step. Returning to the grid restores the last active day. In the date
popup, Tab cycles between the two month buttons and the active date. Main Enter,
keypad Enter and Space select a date; Escape cancels. Closing the keyboard-opened
popup restores focus to the date field. Check that the standalone calendar changes
only the bottom status, while the popup fills its date field.

**Choose date** and **Show choices** are accessible children of their compound
fields. The editable combo also retains model-backed options while collapsed;
they have empty bounds (Windows `IsOffscreen`) and cannot receive keyboard focus.
Their presence in object navigation does not mean that the popup is open. These
model children retain their identities across opening and closing.

On **Grid and nested controls**, entering the grid or using its arrow keys should
announce **Enable Record NNN**, the checkbox role and its checked state. Space
(also either Enter key) toggles that checkbox; it no longer selects a separate
record in this demo. Home/End reach the first/last record. Tab moves directly from
the grid to **Swap first and last**; Shift+Tab returns to the active checkbox.
Toggling with the keyboard, UIA Toggle or a pointer preserves its semantic focus.
The bottom status is a polite live region and reports `Record NNN enabled: true`
or `false` without moving focus. Reordering retains the active record's identity
and state and reveals its new position. Verify these announcements with NVDA;
native tests exercise the same demo component, not a separately recreated grid.

Build `go run -tags accessibility ./cmd/accessibility_demo`, then open **Open
remaining widgets demo**. Repeat with NVDA and Narrator; record versions and the
commit.

1. Read progress, start/stop activity, invoke the named toolbar action, expand and
   collapse sections. Check checkbox states and absence of duplicate drawing text.
2. Type a date and a custom choice. Use Alt+Down, date arrows across a month boundary,
   Enter to choose and Escape to cancel. Check actual focus after closing.
3. Enter the wrapping grid, press End and toggle **Enable Record 499** with Space.
   Check checkbox state speech, the live status and stable focus. Tab to **Swap
   first and last**, activate it, then Shift+Tab and toggle the same record again.
   Also check UIA focus/Toggle and pointer activation of the nested checkbox.
4. Read TextGrid text, including tabs/Cyrillic/emoji. Tab to the divider and resize
   the panes. Move/resize an inner window with the arrows and invoke its buttons.
5. In file/colour dialogs, check names, actual chosen file/colour, disabled controls,
   focus restoration and both light/dark themes.

Automated checks cover shared/public semantics, model identity and stale commands,
existing rendering regressions, native COM role/read-only/event behavior and a
separate-process UIA client against actual GLFW windows. Those checks establish
behavior, not the quality or timing of spoken output.

IME/complex-script editing, other native adapters, and incremental List/Tree model
notifications remain deferred. Application-specific names/alternative text and
custom semantic/keyboard contracts remain the application's responsibility.
