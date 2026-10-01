# Windows accessibility development

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
