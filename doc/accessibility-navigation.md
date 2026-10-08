# Tabs, menus and tables

This increment adds shared semantics and Windows UI Automation support. It does
not change the deferred plan for incremental List/Tree model notifications.

On 2026-10-04 the user completed NVDA testing and reported no critical problems.
This increment is accepted for proceeding to [ScrollPattern](accessibility-scroll.md).
Narrator and the remaining Windows environment checks are still separate gates.

## Tabs

AppTabs and DocTabs expose Tab and TabItem with single, required selection.
TabItem objects retain their identity across reorder. Removing a tab retires its
semantic object, including when the same TabItem is appended again before the
next adapter snapshot. Only the selected page is exposed.

The strip owns one keyboard stop. Left/Right select enabled tabs in a horizontal
strip; Up/Down do so in a vertical strip. Home/End choose the first/last enabled
tab. Tab proceeds to page controls. DocTabs supports Delete to close the selected
document and exposes an independent Close command for each tab, honoring
CloseIntercept and OnClosed. Removing or hiding a page that owns keyboard focus
moves focus to the strip. Overflow tabs remain available to selection APIs.

## Menus

Menu bars, popups and submenus have explicit menu roles. The active menu item is
the active descendant of the real keyboard owner. A submenu is a semantic child
of its opener, with independent clipping because it is rendered outside the
parent menu viewport. Opening a popup preserves the existing canvas scope and
focus restoration behavior; menu-bar dismissal restores the previous focus.

Commands expose Invoke; submenu openers expose ExpandCollapse. Checked menu
items expose Toggle. Set MenuItem.Checkable for a two-state command that must
expose Toggle while unchecked. The existing Action callback changes Checked;
accessibility never changes application state independently of that callback.
Keyboard shortcuts are exposed through AcceleratorKey, not appended to names or
help text. Disabled commands cannot be invoked and keyboard traversal skips them.
Enter on a submenu opener opens it. Escape closes the deepest submenu first,
then dismisses the root menu.

## Table

Table exposes Grid/Table, Selection and ItemContainer. Data cells expose
GridItem/TableItem, SelectionItem, ScrollItem and VirtualizedItem. Grid.GetItem
requests a cell by zero-based row/column without rendering, selecting or scrolling
it. Header relationships return the corresponding row/column header providers.
Only visible cells, selection, keyboard highlight and the shared bounded request
cache are published eagerly.

Use DescribeCell to provide model-based names and metadata. A coordinate of -1
identifies a header, matching UpdateHeader. Custom renderers are not interrogated
for text and their pooled controls are not exposed as independent logical cells.
Editable or independently actionable controls inside cells need a separate
composition contract; this increment does not implement a cell editor.

Optional RowKey and ColumnKey bind identities, selection, active cell and custom
sizes to records across reorder. Keys are nonempty and unique within each axis.
Without a key callback, that axis has positional identity. Call Refresh after
model mutation; observed removal/reinsertion retires old identities. Mutations
that remove and restore a key without any intervening Refresh cannot be detected.

The model index is O(rows + columns), not O(rows * columns). Structural Refresh
still rebuilds those axes. Warm snapshots inspect the viewport and requested
cells. Arbitrary name searches can enumerate cells. Explicitly requesting all
row/column headers necessarily enumerates that axis and can return virtualized
header providers; callers can Realize retained providers if evicted. These costs
are not claimed to be constant time.

The uniform-size ScrollTo coordinate calculation now uses multiplication rather
than repeated float32 addition, avoiding accumulated errors on large tables.
General custom-size geometry costs are otherwise unchanged.

## Validation and manual acceptance

Shared tests cover keyboard navigation, hidden page scope, reorder/removal,
disabled commands, submenu focus/clipping, shortcuts, model cell/header names,
selection/focus retention, and a 100,000 by 100 table whose semantic reads do not
create visual cells. Native tests cover roles, patterns, grid coordinates,
containing grid, headers, invalid coordinates and stale providers. The separate
Windows UIA process exercises actual tab selection, virtual cells, scrolling,
reorder, replacement and popup/submenu scope.

In the Windows demo choose **Open tabs, menus and table demo**. With NVDA/Narrator:

1. Tab into Sections and switch between Documents and Reports using arrows.
   The unavailable tab is skipped; the inactive page must not be spoken.
2. In Documents, switch tabs, use New tab and Delete/Close, and check where focus
   lands after closing the selected document.
3. Open Actions or the context menu. Check the unchecked/checked command,
   disabled command, shortcut and submenu. Escape should return one level at a
   time and ultimately restore focus.
4. In Reports, move through cells with arrows and select with Space. Verify names,
   row/column coordinates and headers. Select Report 200, reverse rows, then
   return focus to the table. The same record should remain selected/active.
5. Remove and restore Report 200; an old UIA reference must not command its new
   lifetime. Move through viewport boundaries and check announcements.

Automated UIA checks do not establish manual screen-reader speech acceptance.

## Sources

- [Microsoft: TabItem control](https://learn.microsoft.com/en-us/windows/win32/winauto/uiauto-supporttabitemcontroltype)
- [Microsoft: MenuItem control](https://learn.microsoft.com/en-us/windows/win32/winauto/uiauto-supportmenuitemcontroltype)
- [Microsoft: Table control](https://learn.microsoft.com/en-us/windows/win32/winauto/uiauto-supporttablecontroltype)
- [Microsoft: control patterns](https://learn.microsoft.com/en-us/windows/win32/winauto/uiauto-controlpatternmapping)

The implementation follows the role/pattern separation in these contracts;
keyboard focus and commands remain owned by Fyne's existing canvas and widgets.
