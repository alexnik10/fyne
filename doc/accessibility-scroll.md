# Scroll containers

Scroll, HScroll, VScroll, List, Tree and Table expose Windows UIA ScrollPattern.
ScrollItem remains a separate command for revealing an individual element.
Ordinary semantic descendants of a Scroll now gain ScrollItem automatically;
nested viewports are revealed from the innermost to the outermost container.
Collection items retain their existing model-based reveal behavior.

## Shared contract

Custom controls implement `fyne.AccessibleScroll`. `AccessibilityScroll` returns
logical canvas units: current offset, content extent, viewport position and size,
enabled directions, and preferred small steps. `AccessibilityScrollTo` changes
the viewport on the event thread and preserves selection and keyboard focus.
Offsets follow Fyne's existing left-to-right, top-to-bottom coordinates.

The shared tree converts these units to percentages. An axis that fits, is
disabled by Direction, has zero viewport size, or has nonfinite geometry reports
percent -1 and view size 100. Other axes report percent and view size in [0,100].
Being disabled does not change a control's scroll metrics; it rejects commands.
For Table, both content and viewport exclude fixed headers and sticky rows/columns.

`SetScrollPercent` accepts [0,100], with -1 leaving that axis unchanged. `Scroll`
accepts None, SmallDecrement/Increment and LargeDecrement/Increment independently
for both axes. Small steps default to the themed icon size plus padding for built-in
scroll containers; pages move 95% of the viewport, matching scrollbar page clicks.
Both requested axes are validated before either changes. Invalid numbers/enums or
unsupported requested axes reject the entire command. At an edge, scrolling is a
successful no-op. Unchanged offsets do not issue another OnScrolled callback.

List/Tree/Table delegate to their existing viewport and OnScrolled behavior.
No scroll command selects, highlights, expands, edits or focuses a model item.
Semantic queries create no offscreen renderer cells. Scrolling still only
publishes the current viewport, selection/highlight and bounded request cache.
The deferred incremental model-notification work is unchanged.

Name a generic viewport with `scroll.SetAccessibilityInfo(...)`. Its logical
children are its Content; scrollbars and shadows are excluded. The viewport is
available in UIA's control/content views even without an explicit name.

## Windows adapter

IScrollProvider implements all six properties, Scroll and SetScrollPercent.
Queries use copied native snapshots. Commands synchronously reach the HWND thread,
rebuild the current scope, and publish the result before returning. No provider
lock is held during dispatch. Hidden, detached and modal-background nodes reject
retained commands. COM interfaces remain safe after replacement and window close.

Changed percent, view-size and scrollability properties issue ordinary property
change events; duplicate snapshots do not repeat them. There is no separate
Scroll event. The provider remains available when content fits, with both axes
non-scrollable, allowing an already retained pattern to observe later resizing.

Native invalid arguments return E_INVALIDARG; an unsupported requested axis returns
UIA_E_INVALIDOPERATION; a disabled owner returns UIA_E_ELEMENTNOTENABLED. Removed
or out-of-scope owners return UIA_E_ELEMENTNOTAVAILABLE.

Reference: [Microsoft Scroll control pattern](https://learn.microsoft.com/en-us/windows/win32/winauto/uiauto-implementingscroll).

## Verification and manual acceptance

Automated coverage includes two axes, relative steps and boundaries, atomic
rejection, disabled/hidden/modal/detached owners, resize-to-fit, nested ScrollItem,
custom public implementations, nonfinite metrics, Table fixed regions, bounded
collection publication, and unchanged focus/selection. Native C tests cover COM
interfaces, exact property values/errors, changed-property events and lifetime.
A separate-process UIA client exercises real GLFW Scroll/List/Tree/Table windows.

Build with `go run -tags accessibility ./cmd/accessibility_demo`. Open **Open
scrolling demo** for a button grid and a table with fixed first row/column. The
short-content checkbox lets the same viewport stop/start scrolling. Resize the
window to check view size changes. The existing List/Tree and large-collection
demos now expose ScrollPattern as well.

Manual NVDA acceptance for this increment remains pending. Check both axes and
endpoints with the available UIA tools, then use NVDA to navigate controls before
and after scrolling. Verify visible content, unchanged selection/focus, no extra
selection announcements, child ScrollIntoView, short/empty content, resizing and
modal scope. Narrator and DPI/multiple-monitor acceptance remain separate gates.
