# Accessibility implementation plan (Windows first)

Baseline: `312219185e6aebb7ac7728abc7044089b657ef27`. The existing
`Accessible` interface and the `accessibility` build tag remain compatible.

## Architecture and acceptance gates

1. **Shared semantics.** Optional capability interfaces, a platform-independent
   tree builder, stable identity for live objects, logical children, immutable
   snapshots, and no dependence on Windows types in widgets. Unit tests cover
   identity, ordering, hidden ancestors, removal, and protected values.
2. **Interactive controls.** Button/link activation, Check toggle, Entry value,
   Slider range, explicit names/descriptions, enabled/read-only/required/invalid
   state. Commands use widget behaviour and run on the Fyne event thread.
3. **Windows provider.** One provider context per HWND; COM lifetime independent
   of the widget lifetime; hierarchical fragment navigation; UIA Invoke, Toggle,
   Value and RangeValue; actual Canvas focus; state/property/structure events.
   Removed objects must return UIA_E_ELEMENTNOTAVAILABLE, never point at another
   object. Windows compilation and native provider regression tests are gates.
4. **Complete basic scenario.** Named form controls, popup scope matching Fyne's
   input scope, focus restoration, sample app, Windows CI and a repeatable NVDA /
   Narrator checklist. Passing automated tests does not replace this checklist.
5. **Text (separate release gate).** ITextProvider/ITextRangeProvider: Unicode
   offsets, selection, caret, line/word boundaries, bounding rectangles, editing
   notifications and password restrictions. ValuePattern alone is not complete
   screen-reader support for an editor.
6. **Collections (separate release gate).** Selection/SelectionItem,
   ExpandCollapse, Scroll/ScrollItem, Grid/Table, logical item IDs independent of
   recycled render cells and ItemContainer/VirtualizedItem where appropriate.
   Cover Select, RadioGroup, List, Table, Tree, tabs, menus and scrolling.
7. **Other platforms.** Consume the shared snapshot/capability model in NSAX,
   Android, iOS and AT-SPI adapters; write mapping and conformance tests before
   claiming parity. Existing adapters remain compatible with the small API.

## Invariants

- Widget state is authoritative; platform queries use snapshots, not widget reads
  from arbitrary COM threads. Platform actions are marshalled and revalidated.
- Reading focus is distinct from keyboard focus and selection. No shadow Tab
  handler: only the Fyne canvas determines the keyboard focus.
- Identifiers survive changes to name/value/bounds/order while objects remain
  attached. Removed IDs are never reused. Recycled collection cells need logical
  item identity before collection support is claimed.
- Commit the whole tree before raising events; never hold provider locks while
  calling UIA, application callbacks or waiting for the event thread.
- Background content is excluded while an input-capturing overlay is active.
- No password text is stored in native snapshots, diagnostics or events.

## Definition of done

Each milestone is reported separately. Full Fyne accessibility requires the text
and collection gates as well as successful real screen-reader scenario tests on
Windows. Windows-first work must not be described as full platform parity.


## Text follow-up after first NVDA acceptance

Implement the plain Entry portion of gate 5 now: shared rune-based text/selection
snapshots, Windows Text and Text2 ranges, protected masks, caret and text events,
selection commands and Entry geometry. Keep IME, complex-script shaping, rich
formatting and broader screen-reader acceptance as explicit remaining gates.
Add a regression for each reported failure before distributing the second demo.
The first user run confirms Tab traversal, form metadata, Check and modal focus;
the revised text and Slider behavior require another real NVDA run.


## Current increment: Select and RadioGroup (demo 4)

The NVDA follow-up confirmed most plain Entry examples, including the intended
single-word underscore behavior. Keep multiline editing, Narrator, IME and complex
scripts as explicit remaining text acceptance work.

Start collection gate 6 with Select and RadioGroup: shared selection and disclosure
capabilities, stable logical options, actual focus mapping, Windows Selection /
SelectionItem / ExpandCollapse, and event-thread commands. A dropdown must preserve
its owner's name/identity while excluding unrelated background controls. Selection
and keyboard highlight must remain distinct until the user commits a choice.

Deliver a Windows executable and a focused acceptance checklist. Automated gates
include selection/required/disabled behavior, focus restoration and scope, retained
COM interfaces, event publication and option removal/reordering. Refactor semantic
snapshot construction into capability helpers to satisfy the existing complexity
limit without weakening static analysis.

Next increments remain tabs/menus, then List/Tree/Table and virtualization. Other
platform adapters can consume these capabilities without depending on Windows types.


## Current follow-up: demo 5

Demo 4 acceptance confirmed Select/RadioGroup semantics, popup isolation and
restoration, disabled-state exposure and the original form regressions. Most
multiline editing scenarios also passed. Before advancing to tabs/menus:

1. Match the requested Windows combo keys, including Alt+arrows and commit on
   Tab/Shift+Tab through the existing canvas focus manager.
2. Correct Line/Paragraph ranges at a final empty line, with native regressions.
3. Publish focused-control feedback before background property changes, and
   ask the user to repeat the Disable choices timing check.
4. Provide Ctrl+Tab / Ctrl+Shift+Tab to leave a multiline Entry without changing
   text, with visible and accessible instructions in Notes and modal-scope tests.
5. Ship a standalone Windows demo and rerun automated desktop/mobile/native gates.

Narrator, IME/complex scripts, the remaining collection patterns and other platform
implementations remain separate acceptance work.

## Diagnostic follow-up: demo 6

Demo 5 behavior is accepted with NVDA. Rapid repeated Disable choices activation
still needs investigation. Before choosing a performance fix, count framework
Space events and actual state changes, and measure widget callbacks, rendering,
semantic snapshot construction, marshaling, native snapshot commit and UIA calls.
Ship an opt-in diagnostic executable with local report saving. Keep speech timing
separate from state/input processing; do not infer lost input from omitted speech.
User-side Windows/NVDA measurements are required to identify the slow stage.


## Query responsiveness follow-up: demo 7

Paired Fyne/NVDA traces confirm actual state changes while HWND normalization
for background UIA events takes tens of milliseconds per element. Return owned
window metadata directly from the Windows provider, retain current title and
correct root/virtual-child HWND identities, and validate through a real external
UIA client with the same cache/normalization operation as NVDA. Keep all state
events and the focused-first order; do not introduce an asynchronous event worker
or alter general Fyne rendering. Metadata alone halves the measured cost but
leaves native messages waiting for frames: add a Windows/accessibility-only native
wait that services synchronous requests and wakes for Go work, frames and shutdown.
Retain the 60 Hz rendering/posted-input path and test real GLFW work dispatch.
Ship a diagnostic Windows build for NVDA speech
acceptance before proceeding to the remaining collection controls.


## Current increment: keyed Tree and unambiguous metadata

Introduce owner/key logical snapshots independently of renderer object identity,
using TreeNodeID as the first stable model key. Retain hidden descriptors, prune
observed removals, reject stale commands, and use fresh semantic objects for
commands. Add model-based Tree descriptions, hierarchy and sibling metadata,
selection, expansion, actual keyboard highlight and offscreen item scrolling.
No visual cells are created for the complete semantic model. Model enumeration
is currently O(N); lazy/paged semantics and List/Table keys remain later work.

Before freezing the API, distinguish unset metadata from explicit empty strings
and false via per-field Set flags. Specify fallback/context/own precedence, Entry
validation defaults and resetting to inheritance. Metadata overrides do not
modify validation or form submission.

Gates: public helper regressions (including 1,000 model items without cell
materialization), independent external-module tests, native provider lifetime and
LeafNode/ScrollItem checks, and a real separate-process UIA Tree scenario through
GLFW. The demo adds a Tree window with 200 reports and reorder/remove/restore.
Manual NVDA/Narrator speech acceptance remains required; automated UIA does not
establish speech quality or acceptance of demo 7 rapid-toggle behavior.


## Current increment: indexed collections, stable List keys and Tree lifetimes

Completed the common source contract (indexed children and per-key semantic
lookup), retaining the existing complete-slice fallback. Model generations retire
stale identities even when deletion and reinsertion occur between snapshots.
Tree uses this contract and preserves descendants across collapse and reparenting.
List adds ItemKey, DescribeItem, model-following selection/highlight/custom heights,
List/ListItem roles, selection, real focus and ScrollItem. No semantic query creates
visual cells. Positional lists remain supported with explicitly positional identity.

Validation includes public API and external-module tests, 1000-row/offscreen and
custom-height checks, source precedence/validation, model replacement without an
intervening snapshot, and a real Windows UIA List scenario alongside Tree.
The demo includes a separate keyed List window. NVDA/Narrator acceptance remains
pending. The shared and native adapters still publish full snapshots; demand
paging, ItemContainer/VirtualizedItem, Table and other platform adapters remain
follow-up work. A per-item source is the extension point, not a paging claim.

## Current increment: demand-driven large collections

Prioritize performance before tabs, menus and Table. The earlier Tree Right-arrow
correction has been accepted with NVDA. See the
[collection design and measurements](accessibility-collections.md).

List and Tree cache the model and publish viewport/selection/highlight plus a
bounded request cache. Windows uses ItemContainer and VirtualizedItem for
offscreen discovery and semantic realization. Compact identities survive eviction;
removed generations stay retired. Native indexes replace quadratic matching.
Initial indexing and structural Refresh remain O(N); renderer costs are separate.

Gates include bounded work on 100,000 records, eviction/realization, stale IDs,
modal/collapse scope, external-module tests, native C lifetime/index tests and
cross-process UIA discovery. Ship a size-selectable demo and request a separate
NVDA/Narrator run; warm semantic benchmarks do not prove speech responsiveness.

## Accepted increment: tabs, menus and Table

AppTabs/DocTabs, menu navigation and model-based Table/Grid/TableItem support
are implemented. On 2026-10-04 the user reported checking the increment with NVDA
and finding no critical problems. This is the acceptance to proceed with
ScrollPattern; it does not establish Narrator or other platform acceptance.
See [navigation coverage](accessibility-navigation.md).

## Current increment: ScrollPattern

Add a shared viewport capability and Windows IScrollProvider for Scroll,
List, Tree and Table, including both axes, relative steps, absolute percentages,
view sizes and property changes. Preserve focus and selection; reject invalid,
disabled, stale and out-of-scope commands. Generic scroll descendants also receive
ScrollItem, with nested viewport reveal. See [scroll semantics and acceptance](accessibility-scroll.md).

Gates: shared/public API tests, native COM lifetime/event tests, a real separate
process UIA scenario, normal/diagnostic demo builds, and the existing full CI
matrix. Wait for every check before reporting completion. Request the next NVDA
acceptance after publishing the demo. Incremental List/Tree model notifications
remain deferred; Table editing/interactive cells require a separate contract.
