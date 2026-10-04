# Built-in widget coverage

This matrix describes the shared semantics and the opt-in Windows UI Automation
adapter on this branch. The remaining-widget increment still requires manual
NVDA/Narrator speech acceptance. It does not enable the other platform adapters.

| Component | Semantics and interaction |
| --- | --- |
| Button, Hyperlink | Named Invoke with the normal application command |
| Label, canvas.Text | Readable static text; `Decorative` omits canvas text already represented by its owner |
| Entry, PasswordEntry, RichTextEntry | Value and Text/Text2, actual editing/selection/focus; password redaction; formatting for rich text |
| RichText, TextGrid | Read-only text with ranges, formatting, geometry and scrolling; TextGrid exposes no unsupported selection |
| Check, CheckGroup | Toggle, checked state, named group, real checkbox focus |
| RadioGroup, Select | Single selection, stable options, keyboard focus; Select disclosure and popup scope |
| SelectEntry | Editable ComboBox with Value/Text, option selection, disclosure, Alt+Up/Down and popup keyboard navigation |
| DateEntry, Calendar | Validated date replacement, calendar disclosure, named month controls, date selection, Grid/Table weekday headers; arrows/Home/End and Escape in the date popup |
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

## Manual acceptance

Build `go run -tags accessibility ./cmd/accessibility_demo`, then open **Open
remaining widgets demo**. Repeat with NVDA and Narrator; record versions and the
commit.

1. Read progress, start/stop activity, invoke the named toolbar action, expand and
   collapse sections. Check checkbox states and absence of duplicate drawing text.
2. Type a date and a custom choice. Use Alt+Down, date arrows across a month boundary,
   Enter to choose and Escape to cancel. Check actual focus after closing.
3. Discover Record 499 in the wrapping grid, reveal it and toggle its nested flag.
   Swap first/last and verify commands still affect the same record.
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
