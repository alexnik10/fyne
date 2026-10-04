# Windows accessibility development

See [Accessibility for custom components](accessibility-authoring.md) for automatic
composition, explicit modes, boundaries, stable identity and public test helpers.

See [Large collections](accessibility-collections.md) for current demand-driven
List/Tree publication, Windows discovery/realization, measurements and acceptance.
Historical full-snapshot notes below describe earlier increments.

See [Tabs, menus and tables](accessibility-navigation.md) for the increment accepted
with NVDA on 2026-10-04, and [Scroll containers](accessibility-scroll.md) for the
current ScrollPattern contract, demo and acceptance checklist.

Build and run the acceptance example:

```sh
go run -tags accessibility ./cmd/accessibility_demo
```

The feature remains opt-in. See [the implementation plan](accessibility-plan.md)
for release gates. This branch is an implementation of the shared foundation and
basic Windows controls, not a claim of complete screen-reader support.

## Current coverage

| Area | Implemented | Remaining acceptance work |
| --- | --- | --- |
| Semantic tree | Logical children, names, descriptions, scoped form metadata, stable live-object IDs, hidden ancestor filtering | Logical IDs for recycled collection items, explicit label relationships, reading-order tooling |
| Button / Hyperlink | Invoke, normal widget command, disabled guard for Button | NVDA and Narrator activation |
| Check | Toggle, checked property and changes; confirmed with NVDA 2026.2 | Narrator acceptance |
| Entry | Value, protected masked Text/Text2, caret and single selection, rune-based ranges, plain text navigation, edit/selection events | Multiline editing acceptance; Narrator, IME, complex scripts, rich text formatting |
| Slider | RangeValue plus string Value, both change events, bounds, step, readonly and numeric validation | Broader boundary/disabled checks; Narrator acceptance |
| Popups | Logical content, dialog marker, top-overlay scope, initial modal focus and restoration; confirmed with NVDA 2026.2 | Narrator acceptance |
| Windows | Per-window context, fragment hierarchy, stable runtime IDs, snapshot queries, reference-counted detached providers, property/structure/focus events | Actual UIA client and screen-reader acceptance, DPI/multi-monitor coverage |
| Select / RadioGroup | Selection/SelectionItem, ExpandCollapse, set position, stable options, actual keyboard focus, popup scope | Demo 4 NVDA acceptance received; demo 5 follow-up and Narrator pending; large collection patterns remain separate |
| Other platforms | Public interfaces remain compatible; shared model has no Windows dependency | Migrate each adapter to the shared model |

Use `SetAccessibilityInfo(fyne.AccessibilityInfo{Name: "...", Description: "..."})`
for an icon-only button, an unlabelled slider or other custom name. In a Form,
labels, hints, required state and validation errors are supplied contextually;
an explicit widget name takes precedence. The entry placeholder is only a fallback.
Custom compound widgets implement `fyne.AccessibleChildren`; an accessible leaf
is a single logical control, irrespective of its drawing objects.

Snapshot construction and widget actions run on the Fyne event thread. COM
queries read a copied native snapshot under a lock. Invoke is queued
asynchronously and revalidated when the event thread executes it. Focus, Toggle,
Value and RangeValue commands are synchronously marshalled to the window's owner
thread through its window procedure, then revalidated and applied before the UIA
method returns. No provider lock is held during dispatch or application code.
Snapshots and providers stay referenced throughout notification publication,
including nested requests that update or close the window. A generation check
stops publishing obsolete events after a nested update. UIA client tests must
verify this thread and reentrancy contract on Windows.

Nodes keep identity while attached, including suppressed background roots.
Detached nodes return `UIA_E_ELEMENTNOTAVAILABLE`. Hiding a parent hides its
semantic descendants. Containers used only for layout stay in the raw hierarchy
but are excluded from the control/content view. Current bounds are canvas client
pixels translated by the native provider into screen coordinates; scroll viewport
clipping and virtualized offscreen items belong to the collection milestone.

## Automated checks

```sh
go test -race -tags ci ./internal/accessibility ./widget ./internal/widget ./internal/app
go build -tags accessibility ./cmd/accessibility_demo
```

On Windows with MinGW:

```sh
gcc -std=c11 -Wall -Werror=incompatible-pointer-types internal/driver/glfw/testdata/accessibility/provider_windows.c -o provider-test.exe -loleaut32 -lole32 -luuid
./provider-test.exe
```

The native regression test holds COM references across snapshot updates,
removal and window closure, queries concurrently with updates, checks the
fragment interface ABI, stable IDs, independent windows, unsupported patterns,
password redaction, state-change notifications, and updates/closure during notification reentrancy. `.github/workflows/accessibility.yml`
compiles the real GLFW driver, runs tests and uploads the demo executable.

## Manual Windows acceptance

Record Windows version, display scaling, Fyne commit, assistive technology/version
and the actual spoken output. Run with NVDA and Narrator separately.

1. Open the demo. Inspect the UIA tree with Accessibility Insights for Windows.
   Confirm field labels, required/invalid state, available patterns, bounds,
   and that static text does not advertise keyboard focusability.
2. Traverse with Tab and Shift+Tab. Match each spoken focus change to the control
   that receives input. A screen reader's reading cursor must remain independent.
3. Fill Email with an invalid address and correct it. Confirm the error and
   enabled Save state. Type a password; confirm UIA Value reads cannot retrieve it.
   Check Left/Right, Ctrl+Left/Right, Home/End, Shift selection and deletion.
   Confirm character echo does not repeat the whole value and password review
   reveals only masking characters, including after a field becomes protected.
4. Toggle Remember through the screen reader, then keyboard. Adjust Volume in
   both ways; confirm changes and boundary behaviour, including disabled controls.
5. Save. Confirm the dialog name and initial focus, inability to reach or invoke
   background controls, Close confirmation activation, and restoration to Save.
6. Change a label while reading it. Verify runtime ID and reading position remain
   stable. Remove the control while retaining its UIA reference; query it again.
7. Open the second window, switch between both, then close one. Confirm the other
   retains its own tree, focus and actions. The demo explicitly focuses its
   second window's Close button on opening. Close the final window with clients
   still attached; watch for hangs, crashes and leaked processes.
8. Repeat under 100%, 150% and 200% scaling and across monitors. Check hit testing
   at every control's edges. Probe immediate GetFocus/Value after synchronous
   requests; the returned state must already match the applied command.

Record failures against the plan. Automated success and a correct inspector tree
are necessary but insufficient evidence of usable screen-reader interaction.

## Platform references

- [UI Automation provider interfaces](https://learn.microsoft.com/en-us/windows/win32/winauto/uiauto-cpinterfaces)
- [Value provider](https://learn.microsoft.com/en-us/windows/win32/api/uiautomationcore/nn-uiautomationcore-ivalueprovider)
- [Provider options](https://learn.microsoft.com/en-us/windows/win32/api/uiautomationcore/ne-uiautomationcore-provideroptions)

## Feedback and follow-up: Windows 11 25H2 / NVDA 2026.2

The user tested build `4ee9269`: Tab/Shift+Tab, Email name/required/hint/error,
validation recovery, password protection, Check, modal focus and focus restoration
worked. Entry caret navigation was silent and typing repeated the entire value.
Slider changes were silent until refocusing. The second demo window required Tab
before its Close button received focus.

The follow-up adds `AccessibleText` and `AccessibleTextScroller` capabilities to
the common model and Text/Text2/TextRange providers to Windows. Rune offsets are
mapped to UTF-16 at the native boundary; snapshots include caret, selection,
text revision and Entry layout. Text edits and selection movements emit distinct
events. Password text is masked in the widget, shared snapshot and native copy;
retained ranges resolve against the current protected snapshot. Selection and
scroll commands return to the Fyne thread and revalidate the current input scope.
Rich formatting/embedded objects are not exported as plain-text attributes.

For Slider, the native RangeValue event was already present. The follow-up also
exposes its string Value and emits that property event. NVDA registers RangeValue
notifications globally and Value notifications on the focused element. This is a
compatibility fix to verify with the reported NVDA setup, not a confirmed diagnosis
of why the global notification was missed. No synthetic speech or focus events
are generated to announce a value.

The Email hint is a persistent form description, not placeholder text. It remains
available after valid input. Validation errors temporarily take precedence over
that description. This policy is covered by a regression test.

The user's repeat NVDA runs reported that almost everything worked, then identified
the word-navigation mismatch described below. This is not complete acceptance of
every editing scenario or another screen reader.
Native tests cover the text provider ABI, Unicode/surrogates, caret events,
selection, retained ranges, privacy, geometry, and both slider value events.

- [Text and TextRange contracts](https://learn.microsoft.com/en-us/windows/win32/winauto/uiauto-implementingtextandtextrange)
- [NVDA UIA event subscriptions](https://github.com/nvaccess/nvda/blob/master/source/UIAHandler/__init__.py)
- [NVDA TextInfo and value events](https://github.com/nvaccess/nvda/blob/master/source/NVDAObjects/UIA/__init__.py)

## Word navigation follow-up (demo 3)

The user's second NVDA 2026.2 run reported that most behavior now worked, but
Ctrl+Right skipped `example` in `user@example.org`. The original Entry shortcut
stopped at rune offsets 0, 4, 12 and 16, while UIA split the address at 0, 4, 5,
12, 13 and 16. Expanding the ranges at 4 and 12 returned only `@` and `.`.

Windows Ctrl+Left/Right and Ctrl+Shift+Left/Right now use the same word boundaries
as `AccessibilityTextInfo.WordBoundaries`. Word and punctuation runs are separate
units; spaces are attached to the preceding unit and newlines remain explicit
stops. Soft wrapping does not create extra word stops. Existing keyboard word
navigation on other platforms is preserved. The optional snapshot field lets
other adapters consume the control's boundaries without reimplementing them.

The Windows provider copies and validates these offsets for ExpandToEnclosingUnit,
Move and MoveEndpointByUnit. Expanding a word at end-of-document returns an empty
range instead of repeating the final word. Password snapshots and native copies
replace any original word boundaries with a single masked word.

Regressions cover the exact address in both directions, Ctrl+Shift selection,
trailing punctuation, spaces, hard/soft line breaks, Cyrillic, supplementary
characters, underscores, native range expansion/movement, ownership of the copied
boundary data and protected text. The user subsequently reported that almost all
examples worked and confirmed that `foo_bar` was read as one word, as intended.


## Selection follow-up (demo 4)

The platform-independent model now includes selection containers/items,
expand/collapse, set position/size, keyboard active descendants and popup owners.
Windows maps these to Selection, SelectionItem and ExpandCollapse providers and
raises selected/removed and property-change events after committing the snapshot.
Selection commands are synchronous on the Fyne event thread, with current scope,
disabled, single-selection and required-selection checks on both sides of the bridge.

Select exposes a read-only string value and stable option objects. Options remain
queryable with empty bounds while collapsed, so GetSelection still identifies the
choice. Opening preserves the combo box's ID and contextual form name; only that
control and its options remain in the active input scope. The popup continues to
own real keyboard input, with its highlighted item exposed as the active descendant.
This is separate from the committed selection: Escape cancels the preview; Enter
commits and returns focus. Arrow navigation reveals clipped popup items; snapshot
bounds are clipped to the popup viewport. General Scroll/ScrollItem and virtualized
collections are still a later gate.

RadioGroup exposes its actual keyboard-focusable radio items. Arrows move focus
and selection together; Enter selects without clearing an already selected item.
Space retains Fyne's existing behavior: an optional group can clear its selection,
a Required group cannot. Existing Tab traversal through radio items is retained.
Items in both controls retain identity by label and occurrence across reordering;
renaming/removing an option detaches it. Select's string-based API retains its
existing ambiguity for duplicate option labels (SelectedIndex chooses the first).

Demo 4 adds **Open selection demo** and **Open multiline text demo**. Keep the
previous form as a regression scenario. On Windows 11 25H2 / NVDA 2026.2 check:

1. Language announces its name, role and current choice. On Windows, all four
   arrows change the collapsed choice, stopping at the ends. Space, Enter or
   Alt+Up/Down opens the popup at the current choice. Up/Down announce each
   highlighted option and its position. Escape preserves the old choice; Enter
   commits. Both return focus to Language. Tab commits, closes and moves to the
   next control; Shift+Tab moves to the previous one. Background form controls
   must be inaccessible while open. Other platforms keep their existing keys.
2. Notifications announces each radio label, state and position in its group.
   Arrows change selection; Enter/Space cannot clear the required selection.
   Optional appearance starts empty; Space can clear the selected option.
3. Disable choices removes these controls from keyboard traversal and exposes
   their disabled state to object navigation. Re-enable and verify normal use.
   Compare the speed of checked/unchecked speech with other checkboxes.
4. Notes supports Up/Down, Home/End, Ctrl+arrows, Shift selection across lines,
   Backspace/Delete, replacement and undo. Check both Cyrillic and the email text.
   Add a newline after the last line, then two more: each final empty line must
   be reported as blank, without repeating the preceding nonempty line.
   Confirm the earlier password and slider behavior still works in the first window.

Automated tests cover selection commands, scope, focus restoration, required and
optional selection, item identity, native COM contracts/events, disabled controls,
empty selections and retained interfaces after removal/closure. Actual speech in
the new scenarios remains a manual acceptance gate.

- [SelectionItem provider](https://learn.microsoft.com/en-us/windows/win32/api/uiautomationcore/nn-uiautomationcore-iselectionitemprovider)
- [ExpandCollapse provider](https://learn.microsoft.com/en-us/windows/win32/api/uiautomationcore/nn-uiautomationcore-iexpandcollapseprovider)
- [ComboBox control](https://learn.microsoft.com/en-us/windows/win32/winauto/uiauto-supportcomboboxcontroltype)


## User feedback and corrections (demo 5)

The user confirmed selection naming, option positions, popup scope and focus
restoration; required/optional RadioGroup behavior; disabled-state exposure;
window switching; and no regressions in the original form. Multiline navigation,
selection and editing mostly worked. Three follow-ups are addressed here:

- Windows Select now uses the requested closed-arrow and disclosure policy above.
  Tab commits the popup highlight and delegates traversal to the canvas focus
  manager, respecting callbacks that deliberately redirect focus. Canvas shortcuts
  continue working after Select/PopUpMenu gain shortcut handling.
- Expanding a caret at the end of text into Line/Paragraph now preserves a final
  empty line instead of selecting its predecessor. Movement can reach that empty
  final unit. Nonempty last lines and whole-document expansion retain their meaning.
- Snapshot property notifications for the focused control are published before
  background controls. This puts Disable choices' Toggle event ahead of the batch
  of enabled-state changes. The complete snapshot is still committed first; native
  tests verify event ordering, reentrant queries and no duplicate events. This
  addresses a plausible source of the reported ~1.5 s speech delay; the latency
  improvement itself still needs a real NVDA repeat test.

Regressions also cover closed-arrow boundaries, initial popup highlight, commit
versus cancellation, forward/backward focus traversal, ordinary-menu behavior,
multiple final newlines and geometry-free text providers. Existing desktop-only
focus assertions are corrected for mobile CI without changing mobile focus policy.


### Leaving a multiline Entry

A further user report identified a keyboard trap in the Notes demo: Tab inserts
a tab character, leaving no documented way to reach Close text demo. Multiline
Entry now uses Ctrl+Tab for the next control and Ctrl+Shift+Tab for the previous
control, without changing the text or caret. Traversal uses the canvas focus
manager, skips disabled controls and stays within modal scope. Single-line
entries and other modifier combinations keep their existing behavior.

Notes exposes this instruction in both its accessible description and a visible
hint. Verify Tab insertion, Ctrl+Tab to Close text demo, and Shift+Tab back into
Notes. Ctrl+Shift+Tab moves backwards; in this two-control window it also reaches
Close text demo because traversal wraps.

This matches the documented Windows Forms behavior for a multiline TextBox with
[AcceptsTab enabled](https://learn.microsoft.com/en-us/dotnet/api/system.windows.forms.textboxbase.acceptstab).

## Responsiveness diagnosis (demo 6)

The user accepted demo 5's combo keyboard behavior, blank final line and multiline
focus escape. Disable choices now announces its checked state immediately, but
rapid repeated activation still seems slower than Remember this account. This
does not yet establish lost input: intermediate speech may be omitted, or widget,
rendering or synchronous UIA event work may delay the next input dispatch.

Build `cmd/accessibility_demo` with `accessibility,accessibilitydiagnostics` to
enable bounded in-memory tracing and a **Save diagnostic report** button in the
main and selection windows. Ordinary builds do not record diagnostics. No event
ordering, threading or selection behavior is changed by this increment.

Record one fresh session: ten separate Space presses on Remember, then ten slow
and ten rapid separate presses on Disable choices, with a pause between groups.
Do not hold Space. Note the intended counts and perceived delay, save the report,
then close the app. The JSON file is beside the executable; it is also saved on
normal application exit. A save failure is reported by the button's dialog.

Trace data contains fixed control identifiers (`remember`, `disable`), monotonic
timestamps, numeric IDs/counts and durations, never Entry contents, passwords,
accessible labels, arbitrary typed text or window titles. It stays in memory
during input; saving performs disk I/O after taking an independent snapshot.
The first 20,000 samples are retained; summary counters continue after that limit
and `dropped_samples` makes truncation explicit. Restart for another test session.

Interpretation:

- `space_press`, `space_repeat`, `space_release`: keyboard callbacks dispatched
  by Fyne while one of the two tracked checks is focused. These cannot count a
  physical key press that never reaches GLFW; compare against the user's count.
- `space_char`: Space delivered to the focused check. `toggle_on`/`toggle_off`
  count actual state changes. `uia_toggle_request` identifies UIA activation;
  pointer activation is separately marked `check_tap`.
- `widget_callback`: work in the check's OnChanged callback, including disabling
  or enabling the choice widgets. `check_space_dispatch` also includes the check's
  own refresh. These are nested measurements and must not be added together.
- `render`, `semantic_tree`, `marshal_snapshot`, `native_update_total`: successive
  frame/update stages. Native total includes the following two nested measures.
- `native_snapshot`: copy/commit of the native snapshot. `uia_events`: time spent
  inside the UIA notification calls plus their call count. `uia_slowest_event`
  identifies the longest call (kind 1 property, 2 automation, 3 structure).
  The remaining native time includes property comparisons and snapshot cleanup.
- `slow_focus_poll`: focus polling only when the call takes at least 1 ms.

The report measures application/provider work, not NVDA's speech queue or the
instant when sound is produced. Many state changes with fewer spoken states
therefore require checking speech separately. A long UIA phase points to the
bridge/client notification path; a long callback/render phase points to widget
or rendering work. Diagnose before selecting an optimization or changing threads.

Native regression tests inject a slow notification callback and verify that it
appears in the event timings, that measurements reset on the next update, and
that the provider's state/event contract is unchanged. Recorder/demo tests cover
bounded storage, counters, saving, I/O errors and normal Check behavior.


### Paired Fyne/NVDA trace findings

The paired demo 6 run on Windows 11 / NVDA 2026.2 records 10 Remember presses
and 11 Disable choices presses, with exactly the same number of state changes.
Every Disable choices notification batch returns within 4.8 ms of the received
key press. There is no evidence of a seconds-long widget or event-raising stall.

NVDA logs all 10 Remember ToggleState callbacks, but only 5 Disable choices
ToggleState callbacks during the rapid series and its subsequent drain. The
second callback arrives 1.05 s after the first, although the next application
state change occurred only 0.217 s after the first. Only the first state is spoken. Between callbacks, HWND normalization for background
choice elements takes 62–129 ms per call (50 calls, 4.45 s in total). This localizes
the backlog to UIA client queries/event processing, not the rendering callback.
NVDA reads the live state after the delayed callback; that state can already have
changed again. Event callbacks and current state queries must therefore be tested
together. These observations do not establish where unobserved callbacks were
coalesced/dropped or attribute the problem to an NVDA defect.

The separate-process Windows query fixture exercises the real UIAutomationCore
HWND-normalization/cache path with a 60 Hz host message pump. It complements the
existing direct-provider tests, which cannot expose cross-process message delays.
Only aggregate findings are kept here; user logs are not repository fixtures.


### Window query correction (demo 7)

The Windows provider now answers HWND identity, process/framework/class, root
control type and root name directly. Root metadata is captured on the HWND
thread; successful WM_SETTEXT updates its owned name under the context lock.
Virtual descendants expose NativeWindowHandle=0, keeping their fragment identity
and the owning window's identity distinct. This removes repeated legacy-host
fallback while NVDA normalizes background event senders to their containing HWND.
State events and their order are unchanged. Other platform adapters continue to
use the shared semantic model without Windows metadata.

The first real UIA probe showed that metadata alone was insufficient: roughly
211 ms/query fell to 99 ms/query, with synchronous HWND messages still waiting
for the next frame. In accessibility-enabled Windows builds the main thread now
waits on sent Windows messages as well as Go work/tick/shutdown wakeups. It
services synchronous queries between frames, without consuming posted keyboard
or pointer input; GLFW still dispatches those. Rendering/animations keep the
existing 60 Hz ticker. The default loop on other builds is unchanged. There is
no polling at 1 kHz and no UIA event worker; an auto-reset event wakes the native
wait when Go work or the frame ticker is ready.

Native tests cover root/child identity, title changes (including Unicode and an
empty title), detached queries and existing provider lifetime/privacy contracts.
The real UIA test compares production queries against a test-only reconstruction
of the old VT_EMPTY metadata fallback, from another process with a 60 Hz host.
The probe also retains a metadata-only control to distinguish the two changes.
It verifies the ancestor HWND and rejects a large relative performance regression
when the baseline reproduces the message-pump delay. Actual GLFW-loop tests cover
concurrent queued work and FIFO asynchronous work followed by a synchronous barrier.

Repeat rapid Space activation of Remember and Disable choices with demo 7; after
Disable choices, wait five seconds on the same control and press Space once more.
Save the diagnostic JSON. Real NVDA speech acceptance is still a separate gate;
faster synthetic queries do not guarantee a spoken announcement per key press.

### Queue readiness and native wakeup

Windows accessibility builds now notify the native wait from the unbounded
queue, after a task is published to its output channel. Waking immediately after
enqueueing at its input could race the queue's relay: the main loop could consume
the signal before the task became readable, leaving execution until a frame tick.
Every publication, including a refill from a backlog and the Close drain, now
signals readiness. Signals may coalesce; the consumer checks the output before
waiting again. Other builds keep the ordinary queue without a native callback.

The same queue handles asynchronous UIA Invoke for Button/Hyperlink and ordinary
main-thread work. Synchronous focus/value/text commands retain their window-message
dispatch. No separate per-widget wake mechanism is needed.

Regression tests check publication-before-notification, FIFO delivery with a full
output buffer and during Close, and a real Windows wait in an isolated process
with no frame ticker. A cross-process UIA client invokes a button that opens a
modal popup, its Close button, and a hyperlink, including a further enqueue from
the opening callback. These checks complement manual NVDA/Narrator acceptance.


## Keyed Tree acceptance

Open **Open tree demo** in the current Windows demo. The Project files tree
contains Documents with 200 reports and Archive with Read me.

1. Tab into the tree. Check its name, item name, level, sibling position/count,
   branch expanded/collapsed state and leaf state. Arrow keys move actual focus;
   Space selects. Selection and focus should remain distinguishable.
2. Press Right on collapsed Documents: it expands and keeps focus. Press Right
   again (or Down) to enter Report 1. Left returns to Documents; another Left
   collapses it without moving focus. The tree has no default spoken keyboard
   instructions. Navigate to Report 200 or use Go to Report 200. It should
   scroll into view without losing its identity. UIA ScrollItem alone must not
   select it or move keyboard focus.
3. Collapse Documents while a report is highlighted. Focus should move to the
   branch; hidden reports must not remain actionable. Re-expand and navigate again.
4. Reverse reports. Check updated positions and retained identity. Remove Report
   200, then restore it: a retained old UIA element must not control the restored
   item. Repeat removal while Documents is collapsed.
5. Repeat with NVDA and Narrator, recording actual speech and keyboard behavior.
   Automated native/provider tests do not replace this acceptance.

Model keys and names are supplied independently of renderer cells. The current
snapshot enumerates all model nodes; it does not claim lazy paging of large data.
Table support and migration of other native adapters remain separate work.

## Keyed List and indexed Tree acceptance

Open **Open list demo** in the Windows demo. The Reports list contains 200 model
items with names supplied independently of recycled visual labels.

1. Check List/ListItem roles, name, position/count and optional single selection.
2. Arrows move real keyboard focus; Space selects. UIA ScrollIntoView on Report 200
   reveals it without selecting or focusing it. UIA SetFocus must not select it.
3. Select and focus Report 200, then reverse the rows. Its runtime ID and selection
   must stay with the report. Keyboard commands must use its new position.
4. Remove the report and restore it. A retained old UIA interface must reject
   commands; the restored record gets a new runtime ID and no inherited selection.
5. Repeat with NVDA and Narrator. Automated provider checks do not establish spoken
   output quality; manual acceptance for this increment remains pending.

List and Tree now expose indexed sources with per-key lookup. Tree also records
closed-branch removals at Refresh, including removal and restoration between
adapter snapshots. Tests cover reparenting, offscreen rows, custom heights,
ambiguous keys, empty collections and external use of the public source contract.
A separate-process UIA List test uses the real GLFW driver and retains providers
through reorder and replacement. The existing native Tree scenario remains active.

This changes the public collection contract, not the native snapshot strategy:
the current backend still enumerates the full model. Native demand paging and
remotely loaded data need a further measured implementation.
