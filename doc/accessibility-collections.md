# Large collections: demand-driven semantics

## Decision and prior art

List and Tree publish the viewport, selection, keyboard highlight, and at most
64 recently requested model keys per collection, plus their ancestors. A separate
model index preserves identity and supports lookup. Realizing semantics never
creates renderer cells, selects, focuses, expands a branch, or scrolls.

The design combines three established approaches:

* [Qt item views](https://github.com/qt/qtbase/blob/dev/src/widgets/accessible/itemviews.cpp)
  create accessible children by model index on request and maintain identity
  using persistent indexes. Fyne similarly separates model membership from
  renderer cells and cached semantic objects.
* [WPF ItemsControlAutomationPeer](https://github.com/dotnet/wpf/blob/main/src/Microsoft.DotNet.Wpf/src/PresentationFramework/System/Windows/Automation/Peers/ItemsControlAutomationPeer.cs)
  discovers data-item peers through ItemContainer, independently of visual
  containers. Fyne exposes documented discovery and realization patterns so
  offscreen records remain reachable.
* [AccessKit](https://github.com/AccessKit/accesskit/blob/main/ARCHITECTURE.md)
  gives adapters frozen semantic data and explicit updates. Ordinary Fyne UIA
  queries still read native-owned snapshots. Only explicit discovery/realization
  runs on the window thread, without holding a provider lock.

Bounded full snapshots preserve atomic publication and reentrant-event safety
without the extra invalidation machinery of incremental native patches. Legacy
sources still work; native ID indexes remove their quadratic snapshot matching.
No asynchronous semantics worker or second focus model is introduced.

## Source and lifetime contract

`AccessibilityCollection` remains the fallback API. The optional
`AccessibilityCollectionView` adds `ViewportKeys()`, `SelectedKeys()`, `Index(key)`
(parent and sibling position), and `Revision()` (changes in keys, hierarchy or
generations). `Element(key)` supplies a lightweight logical object without creating
a visual cell. Model changes must be followed by `Refresh()` before queries.

List and Tree cache topology and geometry until Refresh or geometry invalidation.
Scrolling changes only viewport lookup and relative positions. A compact ID record
survives semantic eviction. `Generation` distinguishes removal/replacement from
eviction, including deletion/reinsertion between platform snapshots. Collapse
retains IDs but rejects discovery/commands on descendants. Hidden owners and modal
scope also apply to realization.

`test.AccessibilityTree.Snapshot()` returns the materialized view for these sources.
`Element(owner, key)` explicitly requests semantics without rendering the row.
Full diagnostic enumeration can deliberately use the indexed source; it is no
longer the cost of every normal snapshot.

## Windows contract

[ItemContainer](https://learn.microsoft.com/en-us/windows/win32/winauto/uiauto-implementingitemcontainer)
supports sequential discovery (property zero), Name, AutomationId and IsSelected.
It returns complete semantics even for offscreen records, with no visual changes.
Tree branches expose discovery of direct children; closed descendants are excluded.
Arbitrary Name search can be linear, as in WPF. Sequential discovery uses the
source index instead of rescanning preceding rows.

[VirtualizedItem](https://learn.microsoft.com/en-us/windows/win32/winauto/uiauto-implementingvirtualizeditem)
`Realize()` repopulates an evicted semantic node with the same RuntimeId. Ordinary
queries on an evicted provider can return `UIA_E_ELEMENTNOTAVAILABLE` until
realization. A removed/replaced ID cannot be realized. `ScrollItem.ScrollIntoView`
is a separate action. Offscreen selected records remain in Selection queries;
actual canvas focus remains authoritative.

Native snapshots have ID-sorted indexes. Record lookup is logarithmic; copying
providers and matching events no longer scan every old record for every new one.
Retained COM interfaces remain safe through eviction, updates and window closure.

## Measurements and remaining costs

Command: `go test -tags ci ./internal/accessibility -run '^$' -bench
BenchmarkCollectionSnapshots -benchtime=3x -count=1`. Linux amd64, AMD EPYC 9V74,
Go 1.26.1, 300 x 300 viewport. These are warm semantic snapshots, excluding initial
indexing, rendering, native UIA and speech. Small-sample timings are indicative;
bounded operation counts are regression assertions.

| 100,000 records | Previous snapshot | Demand-driven snapshot |
| --- | ---: | ---: |
| List time | 899.7 ms | 0.055 ms |
| List allocated bytes | 625,834,485 | 43,757 |
| Tree time | 354.5 ms | 0.033 ms |
| Tree allocated bytes | 261,520,920 | 11,037 |
| Descriptions per snapshot | 100,000 | 8 |
| Published nodes including owner | 100,001 | 9 |

The same viewport describes eight records at 1,000 and 10,000 model sizes. The
benchmark and a native 10,000-record update measurement remain in the repository.
Timing is reported, not used as a flaky CI threshold.

Initial indexing and structural Refresh remain O(N), including closed Tree
membership: callback-based models provide neither mutation deltas nor a reverse
key lookup. The model index is O(N); compact IDs are O(discovered keys). Heavy
semantic/native records are bounded by viewport, selection, requests and ancestry.
Searching arbitrary names is O(N). Existing Tree renderer traversal and model
callback costs remain separate. This does not make creation, structural changes
or rendering constant-time, or fetch the application data itself lazily.

## Acceptance

Open **Open large collections demo**, choose 1,000, 10,000 or 100,000 reports, and
open List or Tree. Use the diagnostic executable when recording timings.

1. Navigate, scroll, leave and return with Tab. Check names, positions and focus.
   Tree Right must expand without entering the child.
2. Go to the last report, select it and reverse the model. Verify retained
   identity, selection and active model key.
3. Remove and restore it, also while the Tree branch is closed. Old providers
   must not control the replacement.
4. Use an external UIA client to discover an offscreen record, exhaust the request
   cache and Realize the retained VirtualizedItem. Check unchanged RuntimeId,
   viewport, selection and focus.
5. Repeat with NVDA and Narrator, especially object navigation at viewport
   boundaries. Automated provider tests do not establish speech acceptance.

The earlier Tree arrow correction was accepted with NVDA. This publication model
requires separate manual acceptance. Tabs, menus and Table remain the next stage.
