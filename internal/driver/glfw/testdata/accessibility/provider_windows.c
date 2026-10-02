// Native provider regression tests, compiled and executed on Windows by CI.
#define CINTERFACE
#define COBJMACROS
#define INITGUID
#include <windows.h>
#include <ole2.h>
#include <uiautomation.h>
#include <assert.h>
#include <stdio.h>
static int propertyEvents, structureEvents, focusEvents;
static int textEvents, selectionEvents, numericEvents, valueEvents;
static int itemSelectionEvents, itemRemovalEvents, expansionEvents;
static void (*duringProperty)(void);
static void (*observeProperty)(IRawElementProviderSimple *, PROPERTYID);
static HRESULT WINAPI propertyEvent(IRawElementProviderSimple *p, PROPERTYID id, VARIANT before, VARIANT after) {
    (void)p; (void)id; (void)before; (void)after; ++propertyEvents;
    if (id == UIA_RangeValueValuePropertyId) { assert(before.vt == VT_R8 && after.vt == VT_R8); ++numericEvents; }
    if (id == UIA_ValueValuePropertyId) { assert(before.vt == VT_BSTR && after.vt == VT_BSTR); ++valueEvents; }
    if (id == UIA_ExpandCollapseExpandCollapseStatePropertyId) ++expansionEvents;
    if (observeProperty) observeProperty(p, id);
    if (duringProperty) { void (*fn)(void) = duringProperty; duringProperty = NULL; fn(); }
    return S_OK;
}
static HRESULT WINAPI structureEvent(IRawElementProviderSimple *p, enum StructureChangeType type, int *ids, int count) {
    (void)p; (void)type; (void)ids; (void)count; ++structureEvents; return S_OK;
}
static HRESULT WINAPI automationEvent(IRawElementProviderSimple *p, EVENTID id) {
    (void)p; if (id == UIA_AutomationFocusChangedEventId) ++focusEvents;
    if (id == UIA_Text_TextChangedEventId) ++textEvents;
    if (id == UIA_Text_TextSelectionChangedEventId) ++selectionEvents;
    if (id == UIA_SelectionItem_ElementSelectedEventId) ++itemSelectionEvents;
    if (id == UIA_SelectionItem_ElementRemovedFromSelectionEventId) ++itemRemovalEvents;
    return S_OK;
}
#define UiaRaiseAutomationPropertyChangedEvent propertyEvent
#define UiaRaiseStructureChangedEvent structureEvent
#define UiaRaiseAutomationEvent automationEvent
#include "../../accessibility_windows.c"

static int actions, lastAction;
static uintptr_t actionWindow;
static uint32_t actionID;
static char actionValue[128];
static int textStart, textEnd, textScroll;
int goFyneAccessibilityTextAction(uintptr_t handle, uint32_t id, int start, int end, int scroll, int alignTop) {
    (void)alignTop; actionWindow = handle; actionID = id;
    textStart = start; textEnd = end; textScroll = scroll; return 1;
}
void goFyneAccessibilityAction(uintptr_t handle, uint32_t id, int act, char *value, double number) {
    lastAction = act; (void)number; ++actions; actionWindow = handle; actionID = id;
    if (value) lstrcpynA(actionValue, value, sizeof(actionValue));
}
int goFyneAccessibilityPerform(uintptr_t handle, uint32_t id, int act, char *value, double number) {
    goFyneAccessibilityAction(handle, id, act, value, number);
    return 1;
}
static HWND newWindow(void) {
    return CreateWindowExW(0, L"STATIC", L"Provider test", WS_OVERLAPPEDWINDOW, 0, 0, 400, 300, NULL, NULL, GetModuleHandleW(NULL), NULL);
}
static Element *retain(WinAccessibility *c, uint32_t id) {
    Element *e = elementFor(c, id); assert(e); addRef(e); return e;
}
static WinAccessibility *nestedContext;
static WinAccessibilityNode *nestedNodes;
static void nestedUpdate(void) {
    nestedNodes[1].name = "Changed during notification";
    assert(WinAccessibilityUpdate(nestedContext, nestedNodes, 5));
}
static void nestedClose(void) { WinAccessibilityCleanup(nestedContext); }
static LONG stopQueries;
static DWORD WINAPI queryWorker(void *arg) {
    Element *e = arg;
    while (!InterlockedCompareExchange(&stopQueries, 0, 0)) {
        VARIANT v; HRESULT hr = property(&e->simple, UIA_NamePropertyId, &v);
        assert(hr == S_OK || hr == UNAVAILABLE);
        if (hr == S_OK) { assert(v.vt == VT_BSTR); assert(v.bstrVal); }
        VariantClear(&v);
    }
    return 0;
}
static void expectText(ITextRangeProvider *range, const WCHAR *expected) {
    BSTR text = NULL;
    assert(ITextRangeProvider_GetText(range, -1, &text) == S_OK);
    assert(text && !wcscmp(text, expected)); SysFreeString(text);
}
static void testTextProvider(void) {
    HWND hwnd = newWindow(); assert(hwnd);
    WinAccessibility *c = WinAccessibilityCreate(hwnd, 44); assert(c);
    WinAccessibilityTextPosition positions[7] = {
        {0,0,20,0}, {10,0,20,0}, {30,0,20,0}, {40,0,20,0},
        {0,20,20,1}, {10,20,20,1}, {20,20,20,1}
    };
    WinAccessibilityNode node = {.id=1, .role=5, .flags=WinAccText|WinAccValue|WinAccFocusable,
        .text="A\xf0\x9f\x98\x80\xd0\x91\nxy", .value="A\xf0\x9f\x98\x80\xd0\x91\nxy",
        .caret=2, .selection_start=2, .selection_end=2, .text_revision=1,
        .positions=positions, .position_count=7, .viewport_width=100, .viewport_height=40};
    assert(WinAccessibilityUpdate(c, &node, 1));
    Element *entry = retain(c, 1);
    IUnknown *unknown = NULL;
    assert(pattern(&entry->simple, UIA_TextPatternId, &unknown) == S_OK && unknown);
    ITextProvider2 *provider = NULL;
    assert(IUnknown_QueryInterface(unknown, &IID_ITextProvider2, (void **)&provider) == S_OK);
    IUnknown_Release(unknown);
    ITextRangeProvider *document = NULL, *range = NULL, *clone = NULL;
    assert(ITextProvider2_get_DocumentRange(provider, &document) == S_OK);
    expectText(document, L"A\xd83d\xde00\u0411\nxy");
    SAFEARRAY *selection = NULL;
    assert(ITextProvider2_GetSelection(provider, &selection) == S_OK);
    LONG index = 0; assert(SafeArrayGetElement(selection, &index, &range) == S_OK);
    SafeArrayDestroy(selection);
    expectText(range, L"");
    assert(ITextRangeProvider_ExpandToEnclosingUnit(range, TextUnit_Character) == S_OK);
    expectText(range, L"\u0411"); // Cyrillic character at the caret, not UTF-16 offset 2
    int moved = 0;
    assert(ITextRangeProvider_Move(range, TextUnit_Character, -1, &moved) == S_OK && moved == -1);
    expectText(range, L"\xd83d\xde00"); // surrogate pair stays intact
    BSTR shortText = NULL;
    assert(ITextRangeProvider_GetText(range, 1, &shortText) == S_OK && SysStringLen(shortText) == 0);
    SysFreeString(shortText);
    assert(ITextRangeProvider_Clone(range, &clone) == S_OK);
    BOOL equal = FALSE; assert(ITextRangeProvider_Compare(range, clone, &equal) == S_OK && equal);
    assert(ITextRangeProvider_Select(range) == S_OK && textStart == 1 && textEnd == 2 && !textScroll);
    assert(ITextRangeProvider_ScrollIntoView(range, TRUE) == S_OK && textScroll);
    VARIANT value;
    assert(ITextRangeProvider_GetAttributeValue(range, UIA_IsReadOnlyAttributeId, &value) == S_OK && value.vt == VT_BOOL && !value.boolVal);
    VariantClear(&value);
    assert(ITextRangeProvider_GetAttributeValue(range, -1, &value) == S_OK && value.vt == VT_UNKNOWN && value.punkVal);
    VariantClear(&value);
    SAFEARRAY *rectangles = NULL;
    assert(ITextRangeProvider_GetBoundingRectangles(range, &rectangles) == S_OK);
    LONG upper = -1; SafeArrayGetUBound(rectangles, 1, &upper); assert(upper == 3); SafeArrayDestroy(rectangles);
    POINT origin = {0,0}; ClientToScreen(hwnd, &origin);
    ITextRangeProvider *point = NULL;
    struct UiaPoint hit = {origin.x+30, origin.y+5};
    assert(ITextProvider2_RangeFromPoint(provider, hit, &point) == S_OK);
    assert(TEXT_RANGE(point)->start == 2); ITextRangeProvider_Release(point);
    assert(ITextProvider2_GetVisibleRanges(provider, &rectangles) == S_OK); SafeArrayDestroy(rectangles);
    ITextRangeProvider *found = NULL; BSTR needle = SysAllocString(L"XY");
    assert(ITextRangeProvider_FindText(document, needle, FALSE, TRUE, &found) == S_OK && found);
    expectText(found, L"xy"); ITextRangeProvider_Release(found); SysFreeString(needle);
    int texts = textEvents, selections = selectionEvents;
    node.caret = node.selection_start = node.selection_end = 1;
    assert(WinAccessibilityUpdate(c, &node, 1));
    assert(textEvents == texts && selectionEvents == selections + 1);
    assert(WinAccessibilityUpdate(c, &node, 1));
    assert(textEvents == texts && selectionEvents == selections + 1); // unchanged snapshots are silent
    node.text_revision++;
    assert(WinAccessibilityUpdate(c, &node, 1));
    assert(textEvents == texts + 1); // replacing an equal string is still an edit
    node.text = "ZA\xf0\x9f\x98\x80\xd0\x91\nxy"; node.position_count=0;
    assert(WinAccessibilityUpdate(c, &node, 1));
    expectText(range, L"\xd83d\xde00"); // retained range follows insertion before it
    // Password queries and previously retained public ranges must only see masks.
    node.flags |= WinAccProtected; node.text = "secret"; node.value = "secret";
    assert(WinAccessibilityUpdate(c, &node, 1));
    ITextRangeProvider_Release(document);
    assert(ITextProvider2_get_DocumentRange(provider, &document) == S_OK);
    expectText(document, L"\u2022\u2022\u2022\u2022\u2022\u2022");
    assert(ITextRangeProvider_GetText(range, -1, &shortText) == S_OK);
    for (UINT i=0; i<SysStringLen(shortText); ++i) assert(shortText[i] == 0x2022);
    SysFreeString(shortText);
    needle = SysAllocString(L"secret"); found = NULL;
    assert(ITextRangeProvider_FindText(document, needle, FALSE, FALSE, &found) == S_OK && !found); SysFreeString(needle);
    assert(getValue(&entry->value, &shortText) == E_ACCESSDENIED && !shortText);
    node.flags |= WinAccDisabled;
    assert(WinAccessibilityUpdate(c, &node, 1));
    assert(ITextRangeProvider_Select(document) == (HRESULT)UIA_E_ELEMENTNOTENABLED);
    assert(WinAccessibilityUpdate(c, NULL, 0));
    assert(ITextRangeProvider_GetText(document, -1, &shortText) == UNAVAILABLE && !shortText);
    WinAccessibilityCleanup(c);
    assert(ITextProvider2_GetSelection(provider, &selection) == UNAVAILABLE && !selection);
    ITextRangeProvider_Release(document); ITextRangeProvider_Release(range); ITextRangeProvider_Release(clone);
    ITextProvider2_Release(provider); release(entry); DestroyWindow(hwnd);
}
static void testWordNavigation(void) {
    HWND hwnd = newWindow(); assert(hwnd);
    WinAccessibility *c = WinAccessibilityCreate(hwnd, 55); assert(c);
    int stops[] = {0, 4, 5, 12, 13, 16};
    const WCHAR *words[] = {L"user", L"@", L"example", L".", L"org", L""};
    WinAccessibilityNode node = {.id=1, .role=5, .flags=WinAccText|WinAccFocusable,
        .text="user@example.org", .word_boundaries=stops, .word_boundary_count=6};
    assert(WinAccessibilityUpdate(c, &node, 1));
    Element *entry = retain(c, 1);
    // Replay the caret stops delivered by Windows Entry shortcuts, then make
    // the same selection/word-expansion requests as NVDA, in both directions.
    for (int direction = 1; direction >= -1; direction -= 2) {
        for (int i = direction > 0 ? 0 : 5; i >= 0 && i < 6; i += direction) {
            node.caret = node.selection_start = node.selection_end = stops[i];
            assert(WinAccessibilityUpdate(c, &node, 1));
            SAFEARRAY *selection = NULL; ITextRangeProvider *range = NULL; LONG index = 0;
            assert(ITextProvider2_GetSelection(&entry->text, &selection) == S_OK);
            assert(SafeArrayGetElement(selection, &index, &range) == S_OK); SafeArrayDestroy(selection);
            assert(ITextRangeProvider_ExpandToEnclosingUnit(range, TextUnit_Word) == S_OK);
            expectText(range, words[i]); ITextRangeProvider_Release(range);
        }
    }
    // Word movement and endpoint movement must use the identical partition.
    BOOL active = FALSE; ITextRangeProvider *range = NULL;
    assert(ITextProvider2_GetCaretRange(&entry->text, &active, &range) == S_OK);
    for (int i=1; i<6; ++i) {
        int moved = 0;
        assert(ITextRangeProvider_Move(range, TextUnit_Word, 1, &moved) == S_OK && moved == 1);
        assert(TEXT_RANGE(range)->start == stops[i] && TEXT_RANGE(range)->end == stops[i]);
    }
    for (int i=4; i>=0; --i) {
        int moved = 0;
        assert(ITextRangeProvider_MoveEndpointByUnit(range, TextPatternRangeEndpoint_Start, TextUnit_Word, -1, &moved) == S_OK && moved == -1);
        assert(TEXT_RANGE(range)->start == stops[i]);
    }
    ITextRangeProvider_Release(range);

    // Underscores are part of an Entry word. A native character classifier must
    // not override the widget's boundaries, and no input array may be retained.
    int custom[] = {0, 7};
    node.text = "foo_bar"; node.word_boundaries = custom; node.word_boundary_count = 2;
    node.caret = node.selection_start = node.selection_end = 3;
    assert(WinAccessibilityUpdate(c, &node, 1)); custom[1] = 3;
    assert(ITextProvider2_GetCaretRange(&entry->text, &active, &range) == S_OK);
    assert(ITextRangeProvider_ExpandToEnclosingUnit(range, TextUnit_Word) == S_OK);
    expectText(range, L"foo_bar"); ITextRangeProvider_Release(range);

    // Protected controls expose one masked word, even if a custom control
    // incorrectly supplied the original password's word boundaries.
    node.text = "user@example.org"; node.flags |= WinAccProtected;
    node.word_boundaries = stops; node.word_boundary_count = 6;
    node.caret = node.selection_start = node.selection_end = 5;
    assert(WinAccessibilityUpdate(c, &node, 1));
    assert(ITextProvider2_GetCaretRange(&entry->text, &active, &range) == S_OK);
    assert(ITextRangeProvider_ExpandToEnclosingUnit(range, TextUnit_Word) == S_OK);
    BSTR masked = NULL;
    assert(ITextRangeProvider_GetText(range, -1, &masked) == S_OK && SysStringLen(masked) == 16);
    for (UINT i=0; i<SysStringLen(masked); ++i) assert(masked[i] == 0x2022);
    SysFreeString(masked); ITextRangeProvider_Release(range);
    WinAccessibilityCleanup(c); release(entry); DestroyWindow(hwnd);
}
#include "selection_windows.h"
#include "feedback_windows.h"

int main(void) {
    CoInitializeEx(NULL, COINIT_APARTMENTTHREADED);
    testWindowMetadata();
    testTextProvider();
    testWordNavigation();
    testSelectionProviders();
    testFinalEmptyLine();
    testFocusedFeedbackOrder();
    testDiagnosticTimings();
    HWND h1 = newWindow(), h2 = newWindow(); assert(h1 && h2);
    WinAccessibility *a = WinAccessibilityCreate(h1, 11), *b = WinAccessibilityCreate(h2, 22);
    assert(a && b && a != b);
    WinAccessibilityNode nodes[] = {
        {.id=1, .parent=0, .role=0, .name="", .width=400, .height=300},
        {.id=2, .parent=1, .role=1, .flags=WinAccInvoke|WinAccFocusable, .name="Save", .width=80, .height=25},
        {.id=3, .parent=1, .role=4, .flags=WinAccToggle|WinAccFocusable, .name="Remember", .y=30, .width=100, .height=25},
        {.id=4, .parent=1, .role=5, .flags=WinAccValue|WinAccFocusable, .name="Password", .value="public", .y=60, .width=100, .height=25},
        {.id=5, .parent=1, .role=6, .flags=WinAccRange|WinAccValue|WinAccFocusable, .name="Volume", .minimum=0, .maximum=10, .step=1, .number=3, .value="3"}
    };
    assert(WinAccessibilityUpdate(a, nodes, 5)); assert(WinAccessibilityUpdate(b, nodes, 5));
    Element *save = retain(a, 2), *check = retain(a, 3), *entry = retain(a, 4), *slider = retain(a, 5);
    Element *other = retain(b, 2), *root = retain(a, 0);
    IRawElementProviderFragment *fragment = NULL;
    assert(navigate(&save->fragment, NavigateDirection_Parent, &fragment) == S_OK);
    assert(OWNER(fragment, fragment)->id == 1); IRawElementProviderFragment_Release(fragment);
    assert(navigate(&save->fragment, NavigateDirection_NextSibling, &fragment) == S_OK);
    assert(OWNER(fragment, fragment) == check); IRawElementProviderFragment_Release(fragment);
    assert(navigate(&a->root->fragment, NavigateDirection_FirstChild, &fragment) == S_OK);
    assert(OWNER(fragment, fragment)->id == 1); IRawElementProviderFragment_Release(fragment);
    SAFEARRAY *before = NULL, *after = NULL;
    assert(runtimeId(&save->fragment, &before) == S_OK);
    IUnknown *patternObject = NULL;
    assert(pattern(&save->simple, UIA_InvokePatternId, &patternObject) == S_OK && patternObject);
    int actionsBefore = actions;
    assert(IInvokeProvider_Invoke((IInvokeProvider *)patternObject) == S_OK);
    assert(actions == actionsBefore + 1 && actionWindow == 11 && actionID == 2); IUnknown_Release(patternObject);
    assert(pattern(&save->simple, UIA_ValuePatternId, &patternObject) == S_OK && !patternObject);
    assert(invoke(&other->invoke) == S_OK && actionWindow == 22);
    assert(toggle(&check->toggle) == S_OK && actionID == 3);
    assert(setValue(&entry->value, L"hello") == S_OK && !strcmp(actionValue, "hello"));
    assert(setRange(&slider->range, -1) == E_INVALIDARG);
    assert(setRange(&slider->range, NAN) == E_INVALIDARG);
    assert(setRange(&slider->range, 6) == S_OK);
    int numericBefore = numericEvents, valueBefore = valueEvents;
    nodes[4].number = 4; nodes[4].value = "4";
    assert(WinAccessibilityUpdate(a, nodes, 5));
    assert(numericEvents == numericBefore + 1 && valueEvents == valueBefore + 1);
    int structures = structureEvents, properties = propertyEvents;
    nodes[1].name = "Saved"; nodes[2].flags |= WinAccChecked;
    assert(WinAccessibilityUpdate(a, nodes, 5));
    assert(structureEvents == structures && propertyEvents > properties);
    assert(elementFor(a, 2) == save);
    assert(runtimeId(&save->fragment, &after) == S_OK);
    LONG index=1, idBefore, idAfter;
    SafeArrayGetElement(before, &index, &idBefore); SafeArrayGetElement(after, &index, &idAfter);
    assert(idBefore == idAfter); SafeArrayDestroy(before); SafeArrayDestroy(after);
    enum ToggleState state;
    assert(toggleState(&check->toggle, &state) == S_OK && state == ToggleState_On);
    nodes[3].flags |= WinAccProtected; nodes[3].value = "secret";
    assert(WinAccessibilityUpdate(a, nodes, 5));
    assert(find(a, 4)->value[0] == 0);
    BSTR secret = NULL; assert(getValue(&entry->value, &secret) == E_ACCESSDENIED && !secret);
    nodes[1].flags |= WinAccDisabled;
    assert(WinAccessibilityUpdate(a, nodes, 5));
    assert(invoke(&save->invoke) == (HRESULT)UIA_E_ELEMENTNOTENABLED);
    nestedContext = a; nestedNodes = nodes;
    duringProperty = nestedUpdate;
    nodes[1].name = "Trigger nested update";
    assert(WinAccessibilityUpdate(a, nodes, 5));
    assert(!wcscmp(find(a, 2)->name, L"Changed during notification"));
    // A retained provider also survives closing during notification publication.
    HWND h3 = newWindow(); assert(h3);
    WinAccessibility *third = WinAccessibilityCreate(h3, 33); assert(third);
    assert(WinAccessibilityUpdate(third, nodes, 5));
    Element *thirdSave = retain(third, 2);
    nestedContext = third; duringProperty = nestedClose;
    nodes[1].name = "Trigger nested close";
    assert(WinAccessibilityUpdate(third, nodes, 5));
    assert(invoke(&thirdSave->invoke) == UNAVAILABLE); release(thirdSave); DestroyWindow(h3);
    // Queries race with snapshots and detachment; external COM references stay valid.
    HANDLE worker = CreateThread(NULL, 0, queryWorker, save, 0, NULL); assert(worker);
    for (int i=0; i<100; ++i) { nodes[1].name = (i%2) ? "A" : "B"; assert(WinAccessibilityUpdate(a, nodes, 5)); }
    assert(WinAccessibilityUpdate(a, NULL, 0));
    VARIANT v; assert(property(&save->simple, UIA_NamePropertyId, &v) == UNAVAILABLE);
    assert(invoke(&save->invoke) == UNAVAILABLE);
    assert(invoke(&other->invoke) == S_OK); // another HWND is unaffected
    WinAccessibilityCleanup(a);
    assert(property(&root->simple, UIA_NamePropertyId, &v) == UNAVAILABLE);
    InterlockedExchange(&stopQueries, 1); WaitForSingleObject(worker, INFINITE); CloseHandle(worker);
    release(save); release(check); release(entry); release(slider); release(root);
    WinAccessibilityCleanup(b); assert(invoke(&other->invoke) == UNAVAILABLE); release(other);
    DestroyWindow(h1); DestroyWindow(h2); CoUninitialize();
    puts("UIA lifetime, identity, hierarchy, patterns, privacy, events and multi-window tests passed");
    return 0;
}
