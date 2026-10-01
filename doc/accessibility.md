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
| Check | Toggle, checked property and changes | Screen-reader announcements |
| Entry | Value, password protection, readonly/disabled, validation information | Text/Text2, caret/selection/ranges and text editing events |
| Slider | RangeValue, bounds, step, readonly and numeric validation | Screen-reader adjustment and feedback |
| Popups | Logical content, dialog marker, top-overlay scope, initial modal focus, existing canvas focus restoration | Dialog opening/closing with NVDA and Narrator |
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

## Manual Windows acceptance (not yet executed)

Record Windows version, display scaling, Fyne commit, assistive technology/version
and the actual spoken output. Run with NVDA and Narrator separately.

1. Open the demo. Inspect the UIA tree with Accessibility Insights for Windows.
   Confirm field labels, required/invalid state, available patterns, bounds,
   and that static text does not advertise keyboard focusability.
2. Traverse with Tab and Shift+Tab. Match each spoken focus change to the control
   that receives input. A screen reader's reading cursor must remain independent.
3. Fill Email with an invalid address and correct it. Confirm the error and
   enabled Save state. Type a password; confirm UIA Value reads cannot retrieve it.
   Do not count this as complete Entry support: text navigation is still pending.
4. Toggle Remember through the screen reader, then keyboard. Adjust Volume in
   both ways; confirm changes and boundary behaviour, including disabled controls.
5. Save. Confirm the dialog name and initial focus, inability to reach or invoke
   background controls, Close confirmation activation, and restoration to Save.
6. Change a label while reading it. Verify runtime ID and reading position remain
   stable. Remove the control while retaining its UIA reference; query it again.
7. Open the second window, switch between both, then close one. Confirm the other
   retains its own tree, focus and actions. Close the final window with clients
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
