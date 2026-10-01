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
| Entry | Value, protected masked Text/Text2, caret and single selection, rune-based ranges, plain text navigation, edit/selection events | Repeat NVDA editing acceptance; Narrator, IME, complex scripts, rich text formatting |
| Slider | RangeValue plus string Value, both change events, bounds, step, readonly and numeric validation | Repeat NVDA adjustment feedback; Narrator acceptance |
| Popups | Logical content, dialog marker, top-overlay scope, initial modal focus and restoration; confirmed with NVDA 2026.2 | Narrator acceptance |
| Windows | Per-window context, fragment hierarchy, stable runtime IDs, snapshot queries, reference-counted detached providers, property/structure/focus events | Actual UIA client and screen-reader acceptance, DPI/multi-monitor coverage |
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

These follow-up changes still need the user's repeat screen-reader test.
Native tests cover the text provider ABI, Unicode/surrogates, caret events,
selection, retained ranges, privacy, geometry, and both slider value events.

- [Text and TextRange contracts](https://learn.microsoft.com/en-us/windows/win32/winauto/uiauto-implementingtextandtextrange)
- [NVDA UIA event subscriptions](https://github.com/nvaccess/nvda/blob/master/source/UIAHandler/__init__.py)
- [NVDA TextInfo and value events](https://github.com/nvaccess/nvda/blob/master/source/NVDAObjects/UIA/__init__.py)
