# Editable Table cells and formatted text

## Text cells

`Table.CellValue(id) (value, readOnly)` reads the model without creating a render
cell. `OnCellChanged(id, value) error` validates and stores the value. An error
must leave the model unchanged. Both are opt-in; existing Table renderers retain
their behaviour. Arbitrary controls supplied by CreateCell are not automatically
converted into text editors.

Configured cells expose Value, GridItem and TableItem together. Writable cells
also expose Invoke. Value commands update the model directly, including offscreen
cells, without selecting or focusing them. Validation failures reject the command.
`AccessibleValueSetter` lets custom controls report the same acceptance result.

F2/Enter, `EditCell`, and Invoke open the same modal text editor. It is a separate
Entry with Value/Text/Text2, not a reused renderer template. Enter or Save commits;
Escape or Cancel discards the draft. Tab/Shift+Tab traverse Entry, Save and Cancel.
Dismissal restores the table's real focus. Validation keeps the editor open and
publishes its error through the Entry's Invalid/Description metadata.

The editor captures the cell key and generation. Sorting keyed axes preserves
the draft's record; removing/reinserting a key or making it read-only cancels the
editor. `Refresh` is required after model changes. Without RowKey/ColumnKey,
identity is positional. Modal input scope excludes background Table commands.
The cell retains its logical ID when the editor closes; a retained editor ID
cannot write to a subsequent editor. Applications should cancel an editor before
detaching a table; hiding the table cancels it automatically.

## Text and formatting

RichText exposes a read-only Document with Text/Text2, geometry, word/line/
paragraph ranges, and selection when Selectable is enabled. Its actual selection
widget delegates focus to the document. Embedded links/controls remain in the
semantic tree. Paragraph breaks and list markers enter the readable text; soft
wrapping does not add paragraph breaks. Synthetic characters map selection back
to the corresponding content boundary.

RichTextEntry exposes formatting on its existing Entry text and selection.
Value replacement uses RichTextEntry.SetText, so the rendered segments, callbacks,
and plain value stay consistent. Password snapshots omit formatting and contain
only masks in both shared and native layers.

`AccessibilityTextInfo.Runs` contains copied, contiguous rune ranges. Supported
Windows attributes are font size (converted to points), weight, italic, underline,
strikethrough, foreground colour, alignment, heading style, and read-only state.
GetAttributeValue returns the UIA reserved Mixed value where an attribute differs;
unknown attributes return NotSupported. FindAttribute searches contiguous matches
forward or backward. TextUnit_Format traverses changes in the exposed formatting.
Formatting-only changes raise TextChanged without moving retained range endpoints.

Actual font-family names, background decoration attributes, embedded-object
RangeFromChild, IME and complex-script boundary/shaping semantics are outside this
increment. Supplying Document/Text patterns does not promise NVDA browse mode.

## Acceptance

Open **Open editable cells and formatted text** in accessibility_demo.

1. In Editable Table, navigate with arrows. ID is read-only; Title and Count edit
   with F2/Enter. Check selection, typing, Enter/Save, Escape/Cancel, and Tab order.
2. Empty Title or negative/non-numeric Count must report an error and keep the
   draft open. Correct it and save. Reorder rows and verify the saved record.
3. In Formatted text, read/edit the editor, select text, apply Bold/Italic and
   inspect formatting with the screen reader. Check Cyrillic and a supplementary
   Unicode character, selection/copy, and Ctrl+Tab to leave the editor.
4. Update the read-only document and check headings, paragraphs, list markers,
   selection/copy and links if present. Repeat ordinary Entry/password regressions.

Automated gates cover model identity, validation, cancellation, stale commands,
modal scope, public external-module APIs, text/selection mapping, immutable runs,
native mixed attributes/search/format movement/privacy/events, and a separate
Windows UIA client using a real GLFW window. NVDA/Narrator speech acceptance is a
separate manual gate.

References: [Text and TextRange requirements](https://learn.microsoft.com/en-us/windows/win32/winauto/uiauto-implementingtextandtextrange),
[text attributes](https://learn.microsoft.com/en-us/windows/win32/winauto/uiauto-textattribute-ids).
