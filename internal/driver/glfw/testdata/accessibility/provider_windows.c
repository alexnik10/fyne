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
static void (*duringProperty)(void);
static HRESULT WINAPI propertyEvent(IRawElementProviderSimple *p, PROPERTYID id, VARIANT before, VARIANT after) {
    (void)p; (void)id; (void)before; (void)after; ++propertyEvents;
    if (duringProperty) { void (*fn)(void) = duringProperty; duringProperty = NULL; fn(); }
    return S_OK;
}
static HRESULT WINAPI structureEvent(IRawElementProviderSimple *p, enum StructureChangeType type, int *ids, int count) {
    (void)p; (void)type; (void)ids; (void)count; ++structureEvents; return S_OK;
}
static HRESULT WINAPI automationEvent(IRawElementProviderSimple *p, EVENTID id) {
    (void)p; if (id == UIA_AutomationFocusChangedEventId) ++focusEvents; return S_OK;
}
#define UiaRaiseAutomationPropertyChangedEvent propertyEvent
#define UiaRaiseStructureChangedEvent structureEvent
#define UiaRaiseAutomationEvent automationEvent
#include "../../accessibility_windows.c"

static int actions;
static uintptr_t actionWindow;
static uint32_t actionID;
static char actionValue[128];
void goFyneAccessibilityAction(uintptr_t handle, uint32_t id, int act, char *value, double number) {
    (void)act; (void)number; ++actions; actionWindow = handle; actionID = id;
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
int main(void) {
    CoInitializeEx(NULL, COINIT_APARTMENTTHREADED);
    HWND h1 = newWindow(), h2 = newWindow(); assert(h1 && h2);
    WinAccessibility *a = WinAccessibilityCreate(h1, 11), *b = WinAccessibilityCreate(h2, 22);
    assert(a && b && a != b);
    WinAccessibilityNode nodes[] = {
        {.id=1, .parent=0, .role=0, .name="", .width=400, .height=300},
        {.id=2, .parent=1, .role=1, .flags=WinAccInvoke|WinAccFocusable, .name="Save", .width=80, .height=25},
        {.id=3, .parent=1, .role=4, .flags=WinAccToggle|WinAccFocusable, .name="Remember", .y=30, .width=100, .height=25},
        {.id=4, .parent=1, .role=5, .flags=WinAccValue|WinAccFocusable, .name="Password", .value="public", .y=60, .width=100, .height=25},
        {.id=5, .parent=1, .role=6, .flags=WinAccRange|WinAccFocusable, .name="Volume", .minimum=0, .maximum=10, .step=1, .number=3}
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
    assert(IInvokeProvider_Invoke((IInvokeProvider *)patternObject) == S_OK);
    assert(actions == 1 && actionWindow == 11 && actionID == 2); IUnknown_Release(patternObject);
    assert(pattern(&save->simple, UIA_ValuePatternId, &patternObject) == S_OK && !patternObject);
    assert(invoke(&other->invoke) == S_OK && actionWindow == 22);
    assert(toggle(&check->toggle) == S_OK && actionID == 3);
    assert(setValue(&entry->value, L"hello") == S_OK && !strcmp(actionValue, "hello"));
    assert(setRange(&slider->range, -1) == E_INVALIDARG);
    assert(setRange(&slider->range, NAN) == E_INVALIDARG);
    assert(setRange(&slider->range, 6) == S_OK);
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
