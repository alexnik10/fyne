//go:build accessibility && windows

// UIA queries never read Go/widget memory. The context lock protects snapshots;
// COM references keep detached providers and their context alive. No lock is held
// while calling UIA or scheduling a Go command.
#define CINTERFACE
#define COBJMACROS
#define INITGUID
#include <windows.h>
#include <ole2.h>
#include <uiautomation.h>
#include <math.h>
#include <stddef.h>
#include <stdlib.h>
#include <wctype.h>
#include "accessibility_windows.h"

extern void goFyneAccessibilityAction(uintptr_t, uint32_t, int, char *, double);
extern int goFyneAccessibilityPerform(uintptr_t, uint32_t, int, char *, double);
extern int goFyneAccessibilityTextAction(uintptr_t, uint32_t, int, int, int, int);
typedef struct {
    uintptr_t handle;
    uint32_t id;
    int action;
    char *text;
    double number;
    HRESULT result;
    int start, end, scroll, alignTop;
} ActionRequest;
static UINT actionMessage;

// One process-lifetime event, shared by the GLFW main loop and Go work/tick
// producers. It must remain valid during concurrent wakeups and app shutdown.
static INIT_ONCE wakeOnce = INIT_ONCE_STATIC_INIT;
static HANDLE mainLoopWake;
static BOOL CALLBACK createWake(PINIT_ONCE once, PVOID parameter, PVOID *context) {
    (void)once; (void)parameter; (void)context;
    mainLoopWake = CreateEventW(NULL, FALSE, FALSE, NULL);
    return mainLoopWake != NULL;
}
void WinAccessibilityWake(void) {
    if (InitOnceExecuteOnce(&wakeOnce, createWake, NULL, NULL)) SetEvent(mainLoopWake);
}
int WinAccessibilityWaitForMessage(uint32_t timeout) {
    if (!InitOnceExecuteOnce(&wakeOnce, createWake, NULL, NULL)) return 0;
    DWORD result = MsgWaitForMultipleObjectsEx(1, &mainLoopWake, timeout, QS_SENDMESSAGE, MWMO_INPUTAVAILABLE);
    if (result == WAIT_FAILED) return 0;
    if (result == WAIT_OBJECT_0 + 1) {
        // PeekMessage dispatches sent messages even when it returns no posted
        // message. Do not remove keyboard/pointer messages from GLFW's queue.
        MSG msg;
        PeekMessageW(&msg, NULL, 0, 0, PM_NOREMOVE | PM_QS_SENDMESSAGE);
    }
    return 1;
}

// MinGW distributions do not consistently ship a UIAutomationCore import
// library. Resolve the documented entry points from the system DLL once.
static INIT_ONCE uiaOnce = INIT_ONCE_STATIC_INIT;
static LRESULT (WINAPI *uiaReturn)(HWND, WPARAM, LPARAM, IRawElementProviderSimple *);
static HRESULT (WINAPI *uiaHost)(HWND, IRawElementProviderSimple **);
static HRESULT (WINAPI *uiaEvent)(IRawElementProviderSimple *, EVENTID);
static HRESULT (WINAPI *uiaProperty)(IRawElementProviderSimple *, PROPERTYID, VARIANT, VARIANT);
static HRESULT (WINAPI *uiaStructure)(IRawElementProviderSimple *, enum StructureChangeType, int *, int);
static HRESULT (WINAPI *uiaDisconnect)(IRawElementProviderSimple *);
static HRESULT (WINAPI *uiaNotSupported)(IUnknown **);
static BOOL CALLBACK loadUIA(PINIT_ONCE once, PVOID parameter, PVOID *context) {
    (void)once; (void)parameter; (void)context;
    HMODULE dll = LoadLibraryExW(L"UIAutomationCore.dll", NULL, LOAD_LIBRARY_SEARCH_SYSTEM32);
    if (!dll) return FALSE;
#define LOAD(target, name) *(FARPROC *)&target = GetProcAddress(dll, name)
    LOAD(uiaReturn, "UiaReturnRawElementProvider");
    LOAD(uiaHost, "UiaHostProviderFromHwnd");
    LOAD(uiaEvent, "UiaRaiseAutomationEvent");
    LOAD(uiaProperty, "UiaRaiseAutomationPropertyChangedEvent");
    LOAD(uiaStructure, "UiaRaiseStructureChangedEvent");
    LOAD(uiaDisconnect, "UiaDisconnectProvider");
    LOAD(uiaNotSupported, "UiaGetReservedNotSupportedValue");
#undef LOAD
    if (!uiaReturn || !uiaHost || !uiaEvent || !uiaProperty || !uiaStructure || !uiaDisconnect || !uiaNotSupported) {
        FreeLibrary(dll); return FALSE;
    }
    actionMessage = RegisterWindowMessageW(L"Fyne.UIAutomation.PerformAction");
    if (!actionMessage) { FreeLibrary(dll); return FALSE; }
    // Kept loaded for the lifetime of externally retained COM providers.
    return TRUE;
}
#define UiaReturnRawElementProvider uiaReturn
#define UiaHostProviderFromHwnd uiaHost
#define UiaDisconnectProvider uiaDisconnect
#ifndef UiaRaiseAutomationEvent
#define UiaRaiseAutomationEvent uiaEvent
#endif
#ifndef UiaRaiseAutomationPropertyChangedEvent
#define UiaRaiseAutomationPropertyChangedEvent uiaProperty
#endif
#ifndef UiaRaiseStructureChangedEvent
#define UiaRaiseStructureChangedEvent uiaStructure
#endif


typedef struct Element Element;
typedef struct TextRange TextRange;
typedef struct {
    WinAccessibilityNode data;
    WCHAR *name, *description, *value;
    Element *element;
    WCHAR *text;
    int *offsets, length;
    unsigned char *wordBoundaries;
    WinAccessibilityTextPosition *positions;
} Record;
// A snapshot remains alive while events are raised, including nested native
// message dispatch that publishes a newer snapshot or closes the window.
typedef struct {
    LONG refs;
    Record records[];
} Snapshot;
static void holdRecords(Record *records) {
    if (records) InterlockedIncrement(&((Snapshot *)((char *)records - offsetof(Snapshot, records)))->refs);
}
struct WinAccessibility {
    SRWLOCK lock;
    LONG refs;
    HWND hwnd;
    WNDPROC original;
    uintptr_t handle;
    int closed;
    int comInitialized;
    WCHAR *windowName;
    WCHAR windowClass[256];
    int count;
    Record *records;
    Element *root;
    uint32_t focus;
    int foreground;
    uint64_t generation;
};
struct Element {
    IRawElementProviderSimple simple;
    IRawElementProviderFragment fragment;
    IRawElementProviderFragmentRoot root;
    IInvokeProvider invoke;
    IToggleProvider toggle;
    IValueProvider value;
    IRangeValueProvider range;
    ITextProvider2 text;
    ISelectionProvider selection;
    ISelectionItemProvider selectionItem;
    IExpandCollapseProvider expand;
    TextRange *textRanges;
    LONG refs;
    uint32_t id;
    WinAccessibility *context;
};
#define OWNER(p, member) ((Element *)((char *)(p) - offsetof(Element, member)))
#define UNAVAILABLE ((HRESULT)UIA_E_ELEMENTNOTAVAILABLE)
#define WINDOW_PROPERTY L"Fyne.UIAutomation.Context"

static IRawElementProviderSimpleVtbl simpleVtbl;
static IRawElementProviderFragmentVtbl fragmentVtbl;
static IRawElementProviderFragmentRootVtbl rootVtbl;
static IInvokeProviderVtbl invokeVtbl;
static IToggleProviderVtbl toggleVtbl;
static IValueProviderVtbl valueVtbl;
static IRangeValueProviderVtbl rangeVtbl;
static ITextProvider2Vtbl textVtbl;
static ISelectionProviderVtbl selectionVtbl;
static ISelectionItemProviderVtbl selectionItemVtbl;
static IExpandCollapseProviderVtbl expandVtbl;
static HRESULT selectionArray(Element *, SAFEARRAY **);
static WCHAR *windowName(HWND);

static void contextRelease(WinAccessibility *c) {
    if (!InterlockedDecrement(&c->refs)) { free(c->windowName); free(c); }
}
static ULONG addRef(Element *e) { return InterlockedIncrement(&e->refs); }
static ULONG release(Element *e) {
    ULONG refs = InterlockedDecrement(&e->refs);
    if (!refs) { WinAccessibility *c = e->context; free(e); contextRelease(c); }
    return refs;
}
static Record *find(WinAccessibility *c, uint32_t id) {
    for (int i = 0; i < c->count; ++i) if (c->records[i].data.id == id) return &c->records[i];
    return NULL;
}
static int alive(Element *e) { return !e->context->closed && (!e->id || find(e->context, e->id)); }
static Element *elementFor(WinAccessibility *c, uint32_t id) {
    if (!id) return c->root;
    Record *r = find(c, id);
    return r ? r->element : NULL;
}
static Element *newElement(WinAccessibility *c, uint32_t id) {
    Element *e = calloc(1, sizeof(*e));
    if (!e) return NULL;
    e->simple.lpVtbl = &simpleVtbl; e->fragment.lpVtbl = &fragmentVtbl;
    e->root.lpVtbl = &rootVtbl; e->invoke.lpVtbl = &invokeVtbl;
    e->toggle.lpVtbl = &toggleVtbl; e->value.lpVtbl = &valueVtbl;
    e->range.lpVtbl = &rangeVtbl; e->refs = 1; e->id = id; e->context = c;
    e->text.lpVtbl = &textVtbl;
    e->selection.lpVtbl = &selectionVtbl; e->selectionItem.lpVtbl = &selectionItemVtbl;
    e->expand.lpVtbl = &expandVtbl;
    InterlockedIncrement(&c->refs);
    return e;
}
static HRESULT query(Element *e, REFIID iid, void **out) {
    if (!out) return E_POINTER;
    *out = NULL;
    if (IsEqualIID(iid, &IID_IUnknown) || IsEqualIID(iid, &IID_IRawElementProviderSimple)) *out = &e->simple;
    else if (IsEqualIID(iid, &IID_IRawElementProviderFragment)) *out = &e->fragment;
    else if (!e->id && IsEqualIID(iid, &IID_IRawElementProviderFragmentRoot)) *out = &e->root;
    else {
        AcquireSRWLockShared(&e->context->lock);
        Record *r = find(e->context, e->id);
        int flags = r && !e->context->closed ? r->data.flags : 0;
        if ((flags & WinAccInvoke) && IsEqualIID(iid, &IID_IInvokeProvider)) *out = &e->invoke;
        if ((flags & WinAccToggle) && IsEqualIID(iid, &IID_IToggleProvider)) *out = &e->toggle;
        if ((flags & WinAccValue) && IsEqualIID(iid, &IID_IValueProvider)) *out = &e->value;
        if ((flags & WinAccRange) && IsEqualIID(iid, &IID_IRangeValueProvider)) *out = &e->range;
        if ((flags & WinAccText) && (IsEqualIID(iid, &IID_ITextProvider) || IsEqualIID(iid, &IID_ITextProvider2))) *out = &e->text;
        if ((flags & WinAccSelection) && IsEqualIID(iid, &IID_ISelectionProvider)) *out = &e->selection;
        if ((flags & WinAccSelectable) && IsEqualIID(iid, &IID_ISelectionItemProvider)) *out = &e->selectionItem;
        if ((flags & WinAccExpandable) && IsEqualIID(iid, &IID_IExpandCollapseProvider)) *out = &e->expand;
        ReleaseSRWLockShared(&e->context->lock);
    }
    if (!*out) return E_NOINTERFACE;
    addRef(e); return S_OK;
}
#define IUNKNOWN(prefix, Type, member) \
static HRESULT STDMETHODCALLTYPE prefix##_QI(Type *p, REFIID iid, void **out) { return query(OWNER(p, member), iid, out); } \
static ULONG STDMETHODCALLTYPE prefix##_Add(Type *p) { return addRef(OWNER(p, member)); } \
static ULONG STDMETHODCALLTYPE prefix##_Release(Type *p) { return release(OWNER(p, member)); }
IUNKNOWN(S, IRawElementProviderSimple, simple)
IUNKNOWN(F, IRawElementProviderFragment, fragment)
IUNKNOWN(R, IRawElementProviderFragmentRoot, root)
IUNKNOWN(I, IInvokeProvider, invoke)
IUNKNOWN(T, IToggleProvider, toggle)
IUNKNOWN(V, IValueProvider, value)
IUNKNOWN(N, IRangeValueProvider, range)
IUNKNOWN(X, ITextProvider2, text)
IUNKNOWN(SL, ISelectionProvider, selection)
IUNKNOWN(SI, ISelectionItemProvider, selectionItem)
IUNKNOWN(EC, IExpandCollapseProvider, expand)

static HRESULT STDMETHODCALLTYPE options(IRawElementProviderSimple *p, enum ProviderOptions *out) {
    if (!out) return E_POINTER;
    *out = ProviderOptions_ServerSideProvider; return S_OK;
}
static HRESULT STDMETHODCALLTYPE pattern(IRawElementProviderSimple *p, PATTERNID id, IUnknown **out) {
    if (!out) return E_POINTER;
    *out = NULL;
    Element *e = OWNER(p, simple); WinAccessibility *c = e->context;
    AcquireSRWLockShared(&c->lock);
    if (!alive(e)) { ReleaseSRWLockShared(&c->lock); return UNAVAILABLE; }
    Record *r = find(c, e->id); int flags = r ? r->data.flags : 0;
    if (id == UIA_InvokePatternId && (flags & WinAccInvoke)) *out = (IUnknown *)&e->invoke;
    if (id == UIA_TogglePatternId && (flags & WinAccToggle)) *out = (IUnknown *)&e->toggle;
    if (id == UIA_ValuePatternId && (flags & WinAccValue)) *out = (IUnknown *)&e->value;
    if (id == UIA_RangeValuePatternId && (flags & WinAccRange)) *out = (IUnknown *)&e->range;
    if ((id == UIA_TextPatternId || id == UIA_TextPattern2Id) && (flags & WinAccText)) *out = (IUnknown *)&e->text;
    if (id == UIA_SelectionPatternId && (flags & WinAccSelection)) *out = (IUnknown *)&e->selection;
    if (id == UIA_SelectionItemPatternId && (flags & WinAccSelectable)) *out = (IUnknown *)&e->selectionItem;
    if (id == UIA_ExpandCollapsePatternId && (flags & WinAccExpandable)) *out = (IUnknown *)&e->expand;
    if (*out) addRef(e);
    ReleaseSRWLockShared(&c->lock); return S_OK;
}
static void variantBool(VARIANT *v, int b) { v->vt = VT_BOOL; v->boolVal = b ? VARIANT_TRUE : VARIANT_FALSE; }
static void integer(VARIANT *v, int i) { v->vt = VT_I4; v->lVal = i; }
static void variantNumber(VARIANT *v, double d) { v->vt = VT_R8; v->dblVal = d; }
static void variantString(VARIANT *v, const WCHAR *s) { v->vt = VT_BSTR; v->bstrVal = SysAllocString(s); }
static int controlType(int role) {
    switch (role) {
    case 1: return UIA_ButtonControlTypeId;
    case 2: return UIA_TextControlTypeId;
    case 3: return UIA_HyperlinkControlTypeId;
    case 4: return UIA_CheckBoxControlTypeId;
    case 5: return UIA_EditControlTypeId;
    case 6: return UIA_SliderControlTypeId;
    case 7: return UIA_PaneControlTypeId;
    case 8: return UIA_ComboBoxControlTypeId;
    case 9: return UIA_RadioButtonControlTypeId;
    case 10: return UIA_ListItemControlTypeId;
    default: return UIA_GroupControlTypeId;
    }
}
// Caller holds the context lock; also used with old immutable records for events.
static void propertyValue(WinAccessibility *c, Record *r, PROPERTYID id, VARIANT *out) {
    VariantInit(out);
    WinAccessibilityNode *n = r ? &r->data : NULL;
    int f = n ? n->flags : 0;
    switch (id) {
    // HWND normalization is used for every background NVDA state event. Avoid
    // falling back to the host's legacy window queries: each can wait for the
    // next native message-pump tick even though our snapshot is already ready.
    // Only the fragment root owns the HWND; virtual controls must return zero.
    case UIA_NativeWindowHandlePropertyId: integer(out, n ? 0 : (LONG)(LONG_PTR)c->hwnd); break;
    case UIA_ProcessIdPropertyId: integer(out, GetCurrentProcessId()); break;
    case UIA_FrameworkIdPropertyId: variantString(out, L"Fyne"); break;
    case UIA_ClassNamePropertyId: variantString(out, n ? L"" : c->windowClass); break;
    case UIA_ControlTypePropertyId: integer(out, n ? controlType(n->role) : UIA_WindowControlTypeId); break;
    case UIA_NamePropertyId: variantString(out, r ? r->name : c->windowName); break;
    case UIA_BoundingRectanglePropertyId:
        if (n) {
            POINT origin = {0, 0}; ClientToScreen(c->hwnd, &origin);
            double values[4] = {origin.x + n->x, origin.y + n->y, n->width, n->height};
            SAFEARRAY *array = SafeArrayCreateVector(VT_R8, 0, 4);
            if (array) {
                for (LONG i = 0; i < 4; ++i) SafeArrayPutElement(array, &i, &values[i]);
                out->vt = VT_ARRAY | VT_R8; out->parray = array;
            }
        }
        break;
    case UIA_HelpTextPropertyId: if (r) variantString(out, r->description); break;
    case UIA_AutomationIdPropertyId:
        if (n) { WCHAR text[32]; wsprintfW(text, L"fyne_%u", n->id); variantString(out, text); } break;
    case UIA_IsKeyboardFocusablePropertyId: variantBool(out, n ? (f & WinAccFocusable) != 0 : 1); break;
    case UIA_HasKeyboardFocusPropertyId: variantBool(out, c->foreground && (n ? (f & WinAccFocused) != 0 : !c->focus)); break;
    case UIA_IsEnabledPropertyId: variantBool(out, !(f & WinAccDisabled)); break;
    case UIA_IsControlElementPropertyId:
    case UIA_IsContentElementPropertyId: variantBool(out, !n || n->role != 0 || (r->name && r->name[0])); break;
    case UIA_IsPasswordPropertyId: variantBool(out, f & WinAccProtected); break;
    case UIA_IsRequiredForFormPropertyId: variantBool(out, f & WinAccRequired); break;
    case UIA_IsDataValidForFormPropertyId: variantBool(out, !(f & WinAccInvalid)); break;
    case UIA_IsDialogPropertyId: variantBool(out, n && n->role == 7); break;
    case UIA_IsOffscreenPropertyId: {
        RECT rect = {0}; GetClientRect(c->hwnd, &rect);
        variantBool(out, n && (n->width <= 0 || n->height <= 0 || n->x + n->width <= 0 || n->y + n->height <= 0 || n->x >= rect.right || n->y >= rect.bottom)); break;
    }
    case UIA_IsInvokePatternAvailablePropertyId: variantBool(out, f & WinAccInvoke); break;
    case UIA_IsTogglePatternAvailablePropertyId: variantBool(out, f & WinAccToggle); break;
    case UIA_IsValuePatternAvailablePropertyId: variantBool(out, f & WinAccValue); break;
    case UIA_IsRangeValuePatternAvailablePropertyId: variantBool(out, f & WinAccRange); break;
    case UIA_IsTextPatternAvailablePropertyId:
    case UIA_IsTextPattern2AvailablePropertyId: variantBool(out, f & WinAccText); break;
    case UIA_ToggleToggleStatePropertyId: if (f & WinAccToggle) integer(out, (f & WinAccChecked) ? ToggleState_On : ToggleState_Off); break;
    case UIA_ValueValuePropertyId: if ((f & WinAccValue) && !(f & WinAccProtected)) variantString(out, r->value); break;
    case UIA_ValueIsReadOnlyPropertyId: if (f & WinAccValue) variantBool(out, f & WinAccReadOnly); break;
    case UIA_RangeValueValuePropertyId: if (f & WinAccRange) variantNumber(out, n->number); break;
    case UIA_RangeValueMinimumPropertyId: if (f & WinAccRange) variantNumber(out, n->minimum); break;
    case UIA_RangeValueMaximumPropertyId: if (f & WinAccRange) variantNumber(out, n->maximum); break;
    case UIA_RangeValueSmallChangePropertyId: if (f & WinAccRange) variantNumber(out, n->step); break;
    case UIA_RangeValueLargeChangePropertyId: if (f & WinAccRange) variantNumber(out, n->step); break;
    case UIA_RangeValueIsReadOnlyPropertyId: if (f & WinAccRange) variantBool(out, f & WinAccReadOnly); break;
    case UIA_IsSelectionPatternAvailablePropertyId: variantBool(out, f & WinAccSelection); break;
    case UIA_IsSelectionItemPatternAvailablePropertyId: variantBool(out, f & WinAccSelectable); break;
    case UIA_IsExpandCollapsePatternAvailablePropertyId: variantBool(out, f & WinAccExpandable); break;
    case UIA_SelectionCanSelectMultiplePropertyId: if (f & WinAccSelection) variantBool(out, f & WinAccMultiple); break;
    case UIA_SelectionIsSelectionRequiredPropertyId: if (f & WinAccSelection) variantBool(out, f & WinAccSelectionRequired); break;
    case UIA_SelectionItemIsSelectedPropertyId: if (f & WinAccSelectable) variantBool(out, f & WinAccSelected); break;
    case UIA_SelectionItemSelectionContainerPropertyId:
        if (f & WinAccSelectable) {
            Record *owner = find(c, n->selection_owner);
            if (owner && (owner->data.flags & WinAccSelection)) {
                out->vt = VT_UNKNOWN; out->punkVal = (IUnknown *)&owner->element->simple; addRef(owner->element);
            }
        }
        break;
    case UIA_SelectionSelectionPropertyId:
        if (f & WinAccSelection) {
            SAFEARRAY *array = NULL;
            if (SUCCEEDED(selectionArray(r->element, &array))) { out->vt = VT_ARRAY | VT_UNKNOWN; out->parray = array; }
        }
        break;
    case UIA_ExpandCollapseExpandCollapseStatePropertyId:
        if (f & WinAccExpandable) integer(out, (f & WinAccExpanded) ? ExpandCollapseState_Expanded : ExpandCollapseState_Collapsed);
        break;
    case UIA_PositionInSetPropertyId: if (n && n->set_position > 0) integer(out, n->set_position); break;
    case UIA_SizeOfSetPropertyId: if (n && n->set_size > 0) integer(out, n->set_size); break;
    case UIA_ProviderDescriptionPropertyId: variantString(out, L"Fyne semantic UI Automation provider"); break;
    }
}
static HRESULT STDMETHODCALLTYPE property(IRawElementProviderSimple *p, PROPERTYID id, VARIANT *out) {
    if (!out) return E_POINTER;
    VariantInit(out);
    Element *e = OWNER(p, simple); WinAccessibility *c = e->context;
    AcquireSRWLockShared(&c->lock);
    HRESULT hr = alive(e) ? S_OK : UNAVAILABLE;
    if (SUCCEEDED(hr)) propertyValue(c, find(c, e->id), id, out);
    ReleaseSRWLockShared(&c->lock); return hr;
}
static HRESULT STDMETHODCALLTYPE host(IRawElementProviderSimple *p, IRawElementProviderSimple **out) {
    if (!out) return E_POINTER;
    *out = NULL;
    Element *e = OWNER(p, simple); WinAccessibility *c = e->context;
    AcquireSRWLockShared(&c->lock); HWND hwnd = c->hwnd; int valid = alive(e);
    ReleaseSRWLockShared(&c->lock);
    if (!valid) return UNAVAILABLE;
    return e->id ? S_OK : UiaHostProviderFromHwnd(hwnd, out);
}
static HRESULT STDMETHODCALLTYPE navigate(IRawElementProviderFragment *p, enum NavigateDirection direction, IRawElementProviderFragment **out) {
    if (!out) return E_POINTER;
    *out = NULL;
    Element *e = OWNER(p, fragment); WinAccessibility *c = e->context;
    AcquireSRWLockShared(&c->lock);
    if (!alive(e)) { ReleaseSRWLockShared(&c->lock); return UNAVAILABLE; }
    Element *target = NULL; Record *r = find(c, e->id);
    if (direction == NavigateDirection_Parent && r) target = elementFor(c, r->data.parent);
    if (direction == NavigateDirection_FirstChild || direction == NavigateDirection_LastChild) {
        for (int i = 0; i < c->count; ++i) if (c->records[i].data.parent == e->id) {
            target = c->records[i].element;
            if (direction == NavigateDirection_FirstChild) break;
        }
    }
    if (r && (direction == NavigateDirection_NextSibling || direction == NavigateDirection_PreviousSibling)) {
        int i = (int)(r - c->records), step = direction == NavigateDirection_NextSibling ? 1 : -1;
        for (i += step; i >= 0 && i < c->count; i += step) if (c->records[i].data.parent == r->data.parent) { target = c->records[i].element; break; }
    }
    if (target) { addRef(target); *out = &target->fragment; }
    ReleaseSRWLockShared(&c->lock); return S_OK;
}
static HRESULT STDMETHODCALLTYPE runtimeId(IRawElementProviderFragment *p, SAFEARRAY **out) {
    if (!out) return E_POINTER;
    *out = NULL;
    Element *e = OWNER(p, fragment); WinAccessibility *c = e->context;
    AcquireSRWLockShared(&c->lock); int valid = alive(e); ReleaseSRWLockShared(&c->lock);
    if (!valid) return UNAVAILABLE;
    if (!e->id) return S_OK; // HWND host supplies the root runtime ID.
    *out = SafeArrayCreateVector(VT_I4, 0, 2);
    if (!*out) return E_OUTOFMEMORY;
    LONG i = 0, v = UiaAppendRuntimeId; SafeArrayPutElement(*out, &i, &v);
    i = 1; v = (LONG)e->id; SafeArrayPutElement(*out, &i, &v); return S_OK;
}
static HRESULT STDMETHODCALLTYPE bounds(IRawElementProviderFragment *p, struct UiaRect *out) {
    if (!out) return E_POINTER;
    Element *e = OWNER(p, fragment); WinAccessibility *c = e->context;
    AcquireSRWLockShared(&c->lock);
    if (!alive(e)) { ReleaseSRWLockShared(&c->lock); return UNAVAILABLE; }
    POINT origin = {0, 0}; ClientToScreen(c->hwnd, &origin);
    Record *r = find(c, e->id);
    if (r) { out->left = origin.x + r->data.x; out->top = origin.y + r->data.y; out->width = r->data.width; out->height = r->data.height; }
    else { RECT rc = {0}; GetClientRect(c->hwnd, &rc); out->left = origin.x; out->top = origin.y; out->width = rc.right; out->height = rc.bottom; }
    ReleaseSRWLockShared(&c->lock); return S_OK;
}
static HRESULT STDMETHODCALLTYPE embedded(IRawElementProviderFragment *p, SAFEARRAY **out) {
    if (!out) return E_POINTER;
    *out = NULL;
    Element *e = OWNER(p, fragment); AcquireSRWLockShared(&e->context->lock);
    int valid = alive(e); ReleaseSRWLockShared(&e->context->lock); return valid ? S_OK : UNAVAILABLE;
}
static HRESULT action(Element *e, int act, char *text, double value) {
    WinAccessibility *c = e->context;
    AcquireSRWLockShared(&c->lock);
    Record *r = find(c, e->id); HRESULT hr = S_OK;
    if (!alive(e)) hr = UNAVAILABLE;
    else if (!r && !(e->id == 0 && act == 0)) hr = (HRESULT)UIA_E_NOTSUPPORTED;
    else if (r && (r->data.flags & WinAccDisabled)) hr = (HRESULT)UIA_E_ELEMENTNOTENABLED;
    else if (r) {
        int required[] = {WinAccFocusable, WinAccInvoke, WinAccToggle, WinAccValue, WinAccRange, 0,
            WinAccSelectable, WinAccSelectable, WinAccSelectable, WinAccExpandable, WinAccExpandable};
        if (act < 0 || act > 10 || !(r->data.flags & required[act])) hr = (HRESULT)UIA_E_NOTSUPPORTED;
        else if ((act == 3 || act == 4) && (r->data.flags & WinAccReadOnly)) hr = (HRESULT)UIA_E_INVALIDOPERATION;
        else if (act == 4 && (!isfinite(value) || value < r->data.minimum || value > r->data.maximum)) hr = E_INVALIDARG;
    }
    if (SUCCEEDED(hr) && act >= 6 && act <= 8) {
        Record *owner = find(c, r->data.selection_owner);
        if (!owner || !(owner->data.flags & WinAccSelection)) hr = UNAVAILABLE;
        else if (owner->data.flags & WinAccDisabled) hr = (HRESULT)UIA_E_ELEMENTNOTENABLED;
        else {
            int others = 0;
            for (int i = 0; i < c->count; ++i) {
                WinAccessibilityNode *n = &c->records[i].data;
                if (n->id != e->id && n->selection_owner == owner->data.id && (n->flags & WinAccSelected)) ++others;
            }
            if (act == 7 && !(owner->data.flags & WinAccMultiple) && others) hr = (HRESULT)UIA_E_INVALIDOPERATION;
            if (act == 8 && (r->data.flags & WinAccSelected) && (owner->data.flags & WinAccSelectionRequired) && !others)
                hr = (HRESULT)UIA_E_INVALIDOPERATION;
        }
    }
    uintptr_t handle = c->handle; HWND hwnd = c->hwnd;
    ReleaseSRWLockShared(&c->lock);
    if (FAILED(hr)) return hr;
    if (act == 1) {
        // Invoke may open a modal UI; the UIA contract requires async dispatch.
        goFyneAccessibilityAction(handle, e->id, act, text, value);
        return S_OK;
    }
    // Synchronous pattern commands are marshalled to the HWND's owner thread.
    // SendMessage also dispatches directly when already on that thread. Never
    // hold a provider lock here: Windows can re-enter during COM/UIA calls.
    ActionRequest request = {.handle=handle, .id=e->id, .action=act, .text=text, .number=value, .result=UNAVAILABLE};
    SendMessageW(hwnd, actionMessage, 0, (LPARAM)&request);
    return request.result;
}
static HRESULT STDMETHODCALLTYPE focus(IRawElementProviderFragment *p) { return action(OWNER(p, fragment), 0, NULL, 0); }
static HRESULT STDMETHODCALLTYPE fragmentRoot(IRawElementProviderFragment *p, IRawElementProviderFragmentRoot **out) {
    if (!out) return E_POINTER;
    *out = NULL;
    Element *e = OWNER(p, fragment); WinAccessibility *c = e->context;
    AcquireSRWLockShared(&c->lock);
    HRESULT hr = alive(e) ? S_OK : UNAVAILABLE;
    if (SUCCEEDED(hr)) { addRef(c->root); *out = &c->root->root; }
    ReleaseSRWLockShared(&c->lock); return hr;
}
static HRESULT STDMETHODCALLTYPE fromPoint(IRawElementProviderFragmentRoot *p, double x, double y, IRawElementProviderFragment **out) {
    if (!out) return E_POINTER;
    *out = NULL;
    Element *e = OWNER(p, root); WinAccessibility *c = e->context;
    AcquireSRWLockShared(&c->lock);
    if (!alive(e)) { ReleaseSRWLockShared(&c->lock); return UNAVAILABLE; }
    POINT origin = {0, 0}; ClientToScreen(c->hwnd, &origin); x -= origin.x; y -= origin.y;
    RECT rc = {0}; GetClientRect(c->hwnd, &rc);
    Element *target = NULL;
    if (x >= 0 && y >= 0 && x < rc.right && y < rc.bottom) {
        target = c->root;
        // Preorder reversed: deepest, frontmost matching child wins.
        for (int i = c->count - 1; i >= 0; --i) {
            WinAccessibilityNode *n = &c->records[i].data;
            if (x >= n->x && y >= n->y && x < n->x + n->width && y < n->y + n->height) { target = c->records[i].element; break; }
        }
    }
    if (target) { addRef(target); *out = &target->fragment; }
    ReleaseSRWLockShared(&c->lock); return S_OK;
}
static HRESULT STDMETHODCALLTYPE getFocus(IRawElementProviderFragmentRoot *p, IRawElementProviderFragment **out) {
    if (!out) return E_POINTER;
    *out = NULL;
    Element *e = OWNER(p, root); WinAccessibility *c = e->context;
    AcquireSRWLockShared(&c->lock);
    if (!alive(e)) { ReleaseSRWLockShared(&c->lock); return UNAVAILABLE; }
    Element *target = c->foreground ? elementFor(c, c->focus) : NULL;
    if (target) { addRef(target); *out = &target->fragment; }
    ReleaseSRWLockShared(&c->lock); return S_OK;
}
static HRESULT STDMETHODCALLTYPE invoke(IInvokeProvider *p) { return action(OWNER(p, invoke), 1, NULL, 0); }
static HRESULT STDMETHODCALLTYPE toggle(IToggleProvider *p) { return action(OWNER(p, toggle), 2, NULL, 0); }
static HRESULT STDMETHODCALLTYPE toggleState(IToggleProvider *p, enum ToggleState *out) {
    if (!out) return E_POINTER;
    VARIANT v; HRESULT hr = property(&OWNER(p, toggle)->simple, UIA_ToggleToggleStatePropertyId, &v);
    if (SUCCEEDED(hr) && v.vt != VT_I4) return (HRESULT)UIA_E_NOTSUPPORTED;
    if (SUCCEEDED(hr)) { *out = (enum ToggleState)v.lVal; }
    return hr;
}
static HRESULT STDMETHODCALLTYPE setValue(IValueProvider *p, LPCWSTR text) {
    if (!text) return E_INVALIDARG;
    Element *e = OWNER(p, value); WinAccessibility *c = e->context;
    AcquireSRWLockShared(&c->lock); Record *r = find(c, e->id);
    int numeric = r && (r->data.flags & WinAccRange);
    ReleaseSRWLockShared(&c->lock);
    if (numeric) {
        WCHAR *end = NULL; double value = wcstod(text, &end);
        if (end == text || *end) return E_INVALIDARG;
        return action(e, 4, NULL, value);
    }
    int size = WideCharToMultiByte(CP_UTF8, 0, text, -1, NULL, 0, NULL, NULL);
    char *utf8 = malloc(size); if (!utf8) return E_OUTOFMEMORY;
    WideCharToMultiByte(CP_UTF8, 0, text, -1, utf8, size, NULL, NULL);
    HRESULT hr = action(OWNER(p, value), 3, utf8, 0); free(utf8); return hr;
}
static HRESULT STDMETHODCALLTYPE getValue(IValueProvider *p, BSTR *out) {
    if (!out) return E_POINTER;
    *out = NULL;
    Element *e = OWNER(p, value); WinAccessibility *c = e->context;
    AcquireSRWLockShared(&c->lock); Record *r = find(c, e->id);
    HRESULT hr = !alive(e) ? UNAVAILABLE : !r || !(r->data.flags & WinAccValue) ? (HRESULT)UIA_E_NOTSUPPORTED : S_OK;
    if (SUCCEEDED(hr) && (r->data.flags & WinAccProtected)) hr = E_ACCESSDENIED;
    if (SUCCEEDED(hr)) { *out = SysAllocString(r->value); if (!*out) hr = E_OUTOFMEMORY; }
    ReleaseSRWLockShared(&c->lock); return hr;
}
static HRESULT readOnly(Element *e, PROPERTYID id, BOOL *out) {
    if (!out) return E_POINTER;
    VARIANT v; HRESULT hr = property(&e->simple, id, &v);
    if (SUCCEEDED(hr) && v.vt != VT_BOOL) return (HRESULT)UIA_E_NOTSUPPORTED;
    if (SUCCEEDED(hr)) { *out = v.boolVal != VARIANT_FALSE; }
    return hr;
}
static HRESULT STDMETHODCALLTYPE valueReadOnly(IValueProvider *p, BOOL *out) { return readOnly(OWNER(p, value), UIA_ValueIsReadOnlyPropertyId, out); }
static HRESULT STDMETHODCALLTYPE setRange(IRangeValueProvider *p, double value) { return action(OWNER(p, range), 4, NULL, value); }
static HRESULT STDMETHODCALLTYPE rangeReadOnly(IRangeValueProvider *p, BOOL *out) { return readOnly(OWNER(p, range), UIA_RangeValueIsReadOnlyPropertyId, out); }
#define RANGE_GETTER(fn, pid) \
static HRESULT STDMETHODCALLTYPE fn(IRangeValueProvider *p, double *out) { \
    if (!out) { return E_POINTER; } VARIANT v; HRESULT hr = property(&OWNER(p, range)->simple, pid, &v); \
    if (SUCCEEDED(hr) && v.vt != VT_R8) return (HRESULT)UIA_E_NOTSUPPORTED; \
    if (SUCCEEDED(hr)) { *out = v.dblVal; } return hr; }
RANGE_GETTER(rangeValue, UIA_RangeValueValuePropertyId)
RANGE_GETTER(rangeMin, UIA_RangeValueMinimumPropertyId)
RANGE_GETTER(rangeMax, UIA_RangeValueMaximumPropertyId)
RANGE_GETTER(rangeLarge, UIA_RangeValueLargeChangePropertyId)
RANGE_GETTER(rangeSmall, UIA_RangeValueSmallChangePropertyId)

static IRawElementProviderSimpleVtbl simpleVtbl = {S_QI, S_Add, S_Release, options, pattern, property, host};
static IRawElementProviderFragmentVtbl fragmentVtbl = {F_QI, F_Add, F_Release, navigate, runtimeId, bounds, embedded, focus, fragmentRoot};
static IRawElementProviderFragmentRootVtbl rootVtbl = {R_QI, R_Add, R_Release, fromPoint, getFocus};
static IInvokeProviderVtbl invokeVtbl = {I_QI, I_Add, I_Release, invoke};
static IToggleProviderVtbl toggleVtbl = {T_QI, T_Add, T_Release, toggle, toggleState};
static IValueProviderVtbl valueVtbl = {V_QI, V_Add, V_Release, setValue, getValue, valueReadOnly};
static IRangeValueProviderVtbl rangeVtbl = {N_QI, N_Add, N_Release, setRange, rangeValue, rangeReadOnly, rangeMax, rangeMin, rangeLarge, rangeSmall};

#include "accessibility_text_windows.h"
#include "accessibility_selection_windows.h"

static LRESULT CALLBACK windowProc(HWND hwnd, UINT msg, WPARAM wp, LPARAM lp) {
    WinAccessibility *c = (WinAccessibility *)GetPropW(hwnd, WINDOW_PROPERTY);
    if (!c) return DefWindowProcW(hwnd, msg, wp, lp);
    if (msg == actionMessage) {
        ActionRequest *request = (ActionRequest *)lp;
        if (request->handle == c->handle) {
            int success = request->action == 5
                ? goFyneAccessibilityTextAction(request->handle, request->id, request->start, request->end, request->scroll, request->alignTop)
                : goFyneAccessibilityPerform(request->handle, request->id, request->action, request->text, request->number);
            request->result = success ? S_OK : UNAVAILABLE;
        }
        return 0;
    }
    if (msg == WM_GETOBJECT && lp == (LPARAM)UiaRootObjectId) return UiaReturnRawElementProvider(hwnd, wp, lp, &c->root->simple);
    if (msg == WM_SETTEXT) {
        // Keep the cached root name current, including title changes made by
        // native code. Default processing can re-enter and close the window.
        InterlockedIncrement(&c->refs);
        LRESULT result = CallWindowProcW(c->original, hwnd, msg, wp, lp);
        WCHAR *title = result ? windowName(hwnd) : NULL;
        if (title) {
            AcquireSRWLockExclusive(&c->lock);
            if (!c->closed) { WCHAR *old = c->windowName; c->windowName = title; title = old; }
            ReleaseSRWLockExclusive(&c->lock);
            free(title);
        }
        contextRelease(c);
        return result;
    }
    // Leave all keyboard, pointer and focus handling to GLFW/Fyne.
    return CallWindowProcW(c->original, hwnd, msg, wp, lp);
}
static WCHAR *wide(const char *text) {
    if (!text) text = "";
    int count = MultiByteToWideChar(CP_UTF8, 0, text, -1, NULL, 0);
    WCHAR *out = calloc(count, sizeof(WCHAR));
    if (out) MultiByteToWideChar(CP_UTF8, 0, text, -1, out, count);
    return out;
}
// Called only on the HWND thread; queries from UIA use the owned copy instead.
static WCHAR *windowName(HWND hwnd) {
    int length = GetWindowTextLengthW(hwnd);
    WCHAR *name = calloc((size_t)length + 1, sizeof(WCHAR));
    if (name) GetWindowTextW(hwnd, name, length + 1);
    return name;
}
static void freeRecords(Record *records, int count) {
    if (!records) return;
    Snapshot *snapshot = (Snapshot *)((char *)records - offsetof(Snapshot, records));
    if (InterlockedDecrement(&snapshot->refs)) return;
    for (int i = 0; i < count; ++i) {
        free(records[i].name); free(records[i].description); free(records[i].value);
        free(records[i].text); free(records[i].offsets); free(records[i].positions);
        free(records[i].wordBoundaries);
        if (records[i].element) release(records[i].element);
    }
    free(snapshot);
}
WinAccessibility *WinAccessibilityCreate(void *hwnd, uintptr_t handle) {
    if (!InitOnceExecuteOnce(&uiaOnce, loadUIA, NULL, NULL)) return NULL;
    WinAccessibility *c = calloc(1, sizeof(*c)); if (!c) return NULL;
    HRESULT com = CoInitializeEx(NULL, COINIT_APARTMENTTHREADED);
    if (FAILED(com) && com != RPC_E_CHANGED_MODE) { free(c); return NULL; }
    c->comInitialized = SUCCEEDED(com);
    InitializeSRWLock(&c->lock); c->refs = 1; c->hwnd = hwnd; c->handle = handle;
    c->windowName = windowName(hwnd);
    GetClassNameW(hwnd, c->windowClass, sizeof(c->windowClass)/sizeof(c->windowClass[0]));
    if (!c->windowName) { if (c->comInitialized) CoUninitialize(); contextRelease(c); return NULL; }
    c->root = newElement(c, 0);
    if (!c->root) { if (c->comInitialized) CoUninitialize(); contextRelease(c); return NULL; }
    if (!SetPropW(hwnd, WINDOW_PROPERTY, c)) { if (c->comInitialized) CoUninitialize(); release(c->root); contextRelease(c); return NULL; }
    SetLastError(0);
    c->original = (WNDPROC)SetWindowLongPtrW(hwnd, GWLP_WNDPROC, (LONG_PTR)windowProc);
    if (!c->original) { RemovePropW(hwnd, WINDOW_PROPERTY); if (c->comInitialized) CoUninitialize(); release(c->root); contextRelease(c); return NULL; }
    return c;
}
static int equalVariant(VARIANT *a, VARIANT *b) {
    if (a->vt != b->vt) return 0;
    switch (a->vt) {
    case VT_BSTR: return wcscmp(a->bstrVal ? a->bstrVal : L"", b->bstrVal ? b->bstrVal : L"") == 0;
    case VT_BOOL: return a->boolVal == b->boolVal;
    case VT_I4: return a->lVal == b->lVal;
    case VT_R8: return a->dblVal == b->dblVal;
    case VT_ARRAY | VT_R8:
        for (LONG i = 0; i < 4; ++i) {
            double x, y; SafeArrayGetElement(a->parray, &i, &x); SafeArrayGetElement(b->parray, &i, &y);
            if (x != y) return 0;
        }
        return 1;
    default: return 1;
    }
}
static int eventsCurrent(WinAccessibility *c, uint64_t generation) {
    AcquireSRWLockShared(&c->lock);
    int current = !c->closed && c->generation == generation;
    ReleaseSRWLockShared(&c->lock);
    return current;
}
#include "accessibility_diagnostic_windows.h"

int WinAccessibilityUpdate(WinAccessibility *c, const WinAccessibilityNode *nodes, int count) {
    return WinAccessibilityUpdateWithStats(c, nodes, count, NULL);
}

int WinAccessibilityUpdateWithStats(WinAccessibility *c, const WinAccessibilityNode *nodes, int count, WinAccessibilityStats *stats) {
    if (stats) memset(stats, 0, sizeof(*stats));
    double snapshotStart = diagnosticNow(stats);
    if (!c || count < 0) return 0;
    Snapshot *snapshot = calloc(1, sizeof(*snapshot) + count * sizeof(Record));
    if (!snapshot) return 0;
    snapshot->refs = 1;
    Record *next = snapshot->records;
    uint32_t focused = 0;
    AcquireSRWLockExclusive(&c->lock);
    if (c->closed) { ReleaseSRWLockExclusive(&c->lock); free(snapshot); return 0; }
    for (int i = 0; i < count; ++i) {
        Record *r = &next[i]; r->data = nodes[i];
        // No borrowed strings or Go memory survives this call.
        r->data.name = r->data.description = r->data.value = r->data.text = NULL;
        r->data.positions = NULL;
        r->data.word_boundaries = NULL;
        r->name = wide(nodes[i].name); r->description = wide(nodes[i].description);
        r->value = wide((nodes[i].flags & WinAccProtected) ? "" : nodes[i].value);
        int textOK = copyText(r, &nodes[i]);
        Record *old = find(c, nodes[i].id);
        if (old) { r->element = old->element; addRef(r->element); }
        else r->element = newElement(c, nodes[i].id);
        if (!r->name || !r->description || !r->value || !r->element || !textOK) {
            ReleaseSRWLockExclusive(&c->lock); freeRecords(next, count); return 0;
        }
        if (nodes[i].flags & WinAccFocused) focused = nodes[i].id;
    }
    Record *old = c->records; int oldCount = c->count;
    int structure = count != oldCount;
    for (int i = 0; !structure && i < count; ++i) structure = old[i].data.id != next[i].data.id || old[i].data.parent != next[i].data.parent;
    int foreground = GetForegroundWindow() == c->hwnd;
    int focusChanged = c->focus != focused || c->foreground != foreground;
    for (int i = 0; i < count; ++i) {
        Record *previous = find(c, next[i].data.id);
        if (previous) updateTextRanges(previous, &next[i]);
    }
    c->records = next; c->count = count; c->focus = focused; c->foreground = foreground;
    uint64_t generation = ++c->generation;
    holdRecords(next); // local event publication reference, besides context ownership
    Element *root = c->root; addRef(root);
    Element *focusElement = elementFor(c, focused);
    if (focusElement) addRef(focusElement);
    ReleaseSRWLockExclusive(&c->lock);
    if (stats) stats->snapshot_ms = diagnosticNow(stats) - snapshotStart;
    // Queries may re-enter during events and see the complete new tree.
    if (structure) diagnosticStructure(stats, root);
    PROPERTYID properties[] = {UIA_NamePropertyId, UIA_HelpTextPropertyId, UIA_IsEnabledPropertyId,
        UIA_IsKeyboardFocusablePropertyId, UIA_IsPasswordPropertyId, UIA_IsRequiredForFormPropertyId,
        UIA_IsDataValidForFormPropertyId, UIA_ToggleToggleStatePropertyId, UIA_ValueValuePropertyId,
        UIA_ValueIsReadOnlyPropertyId, UIA_RangeValueValuePropertyId, UIA_RangeValueIsReadOnlyPropertyId,
        UIA_RangeValueMinimumPropertyId, UIA_RangeValueMaximumPropertyId, UIA_RangeValueSmallChangePropertyId,
        UIA_ControlTypePropertyId, UIA_IsDialogPropertyId, UIA_BoundingRectanglePropertyId,
        UIA_IsInvokePatternAvailablePropertyId, UIA_IsTogglePatternAvailablePropertyId,
        UIA_IsValuePatternAvailablePropertyId, UIA_IsRangeValuePatternAvailablePropertyId,
        UIA_IsTextPatternAvailablePropertyId, UIA_IsTextPattern2AvailablePropertyId,
        UIA_IsSelectionPatternAvailablePropertyId, UIA_IsSelectionItemPatternAvailablePropertyId,
        UIA_IsExpandCollapsePatternAvailablePropertyId, UIA_SelectionCanSelectMultiplePropertyId,
        UIA_SelectionIsSelectionRequiredPropertyId, UIA_SelectionItemIsSelectedPropertyId,
        UIA_ExpandCollapseExpandCollapseStatePropertyId, UIA_PositionInSetPropertyId, UIA_SizeOfSetPropertyId,
        UIA_IsOffscreenPropertyId};
    // Announce the user's focused control before potentially slow client calls
    // for every affected background control (e.g. "Disable choices"). The whole
    // snapshot is already committed, so reentrant queries see all state changes.
    for (int pass = 0; pass < 2; ++pass) for (int i = 0; i < count; ++i) {
        if ((next[i].data.id == focused) != (pass == 0)) continue;
        for (int j = 0; j < oldCount; ++j) if (next[i].data.id == old[j].data.id) {
            for (unsigned int k = 0; k < sizeof(properties)/sizeof(properties[0]) && eventsCurrent(c, generation); ++k) {
                // Never emit the old password when a previously public field becomes protected.
                if (properties[k] == UIA_ValueValuePropertyId && ((old[j].data.flags | next[i].data.flags) & WinAccProtected)) continue;
                VARIANT a, b;
                propertyValue(c, &old[j], properties[k], &a); propertyValue(c, &next[i], properties[k], &b);
                if (!equalVariant(&a, &b)) diagnosticProperty(stats, next[i].element, properties[k], a, b);
                VariantClear(&a); VariantClear(&b);
            }
            if ((next[i].data.flags & WinAccSelectable) &&
                ((old[j].data.flags ^ next[i].data.flags) & WinAccSelected) && eventsCurrent(c, generation)) {
                EVENTID event = UIA_SelectionItem_ElementRemovedFromSelectionEventId;
                if (next[i].data.flags & WinAccSelected) {
                    event = UIA_SelectionItem_ElementSelectedEventId;
                    for (int k = 0; k < count; ++k)
                        if (next[k].data.id == next[i].data.selection_owner && (next[k].data.flags & WinAccMultiple))
                            event = UIA_SelectionItem_ElementAddedToSelectionEventId;
                }
                diagnosticAutomation(stats, next[i].element, event);
            }
            if ((next[i].data.flags & WinAccText) && eventsCurrent(c, generation)) {
                int textChanged = old[j].data.text_revision != next[i].data.text_revision || wcscmp(old[j].text, next[i].text);
                if (textChanged) diagnosticAutomation(stats, next[i].element, UIA_Text_TextChangedEventId);
                if (eventsCurrent(c, generation) && (textChanged || old[j].data.caret != next[i].data.caret ||
                    old[j].data.selection_start != next[i].data.selection_start || old[j].data.selection_end != next[i].data.selection_end))
                    diagnosticAutomation(stats, next[i].element, UIA_Text_TextSelectionChangedEventId);
            }
            break;
        }
    }
    if (focusChanged && foreground && focusElement && eventsCurrent(c, generation)) diagnosticAutomation(stats, focusElement, UIA_AutomationFocusChangedEventId);
    if (focusElement) release(focusElement);
    // Old providers now resolve their ID against the new tree. Removed providers
    // return ELEMENTNOTAVAILABLE until the last external COM reference is released.
    freeRecords(old, oldCount);
    freeRecords(next, count);
    release(root);
    return 1;
}
void WinAccessibilityCleanup(WinAccessibility *c) {
    if (!c) return;
    HWND hwnd = c->hwnd;
    SetWindowLongPtrW(hwnd, GWLP_WNDPROC, (LONG_PTR)c->original);
    RemovePropW(hwnd, WINDOW_PROPERTY);
    AcquireSRWLockExclusive(&c->lock);
    c->closed = 1; c->hwnd = NULL;
    Record *records = c->records; int count = c->count;
    c->records = NULL; c->count = 0;
    ReleaseSRWLockExclusive(&c->lock);
    UiaDisconnectProvider(&c->root->simple);
    for (int i = 0; i < count; ++i) UiaDisconnectProvider(&records[i].element->simple);
    freeRecords(records, count); release(c->root);
    if (c->comInitialized) CoUninitialize();
    contextRelease(c);
}

void WinAccessibilityFocus(WinAccessibility *c, uint32_t id) {
    AcquireSRWLockExclusive(&c->lock);
    if (c->closed) { ReleaseSRWLockExclusive(&c->lock); return; }
    if (id && !find(c, id)) id = 0;
    int foreground = GetForegroundWindow() == c->hwnd;
    int changed = c->focus != id || c->foreground != foreground;
    c->focus = id; c->foreground = foreground;
    for (int i = 0; i < c->count; ++i) {
        c->records[i].data.flags &= ~WinAccFocused;
        if (c->records[i].data.id == id) c->records[i].data.flags |= WinAccFocused;
    }
    Element *target = elementFor(c, id);
    if (target) addRef(target);
    ReleaseSRWLockExclusive(&c->lock);
    if (changed && foreground && target) UiaRaiseAutomationEvent(&target->simple, UIA_AutomationFocusChangedEventId);
    if (target) release(target);
}
