// Private implementation included by accessibility_windows.c.
// Queries stay on immutable native snapshots. Only explicit discovery/realization
// crosses to the event thread, never while a provider lock is held.

static HRESULT STDMETHODCALLTYPE realizeItem(IVirtualizedItemProvider *p) {
    Element *e = OWNER(p, virtualizedItem); WinAccessibility *c = e->context;
    AcquireSRWLockShared(&c->lock);
    HWND hwnd = c->hwnd; uintptr_t handle = c->handle;
    int valid = !c->closed && e->virtualized;
    ReleaseSRWLockShared(&c->lock);
    if (!valid) return UNAVAILABLE;
    ActionRequest request = {.handle=handle, .id=e->id, .action=12, .result=UNAVAILABLE};
    SendMessageW(hwnd, actionMessage, 0, (LPARAM)&request);
    return request.result;
}

static HRESULT STDMETHODCALLTYPE findItem(IItemContainerProvider *p, IRawElementProviderSimple *startAfter,
                                          PROPERTYID property, VARIANT value, IRawElementProviderSimple **out) {
    if (!out) return E_POINTER;
    *out = NULL;
    Element *e = OWNER(p, itemContainer); WinAccessibility *c = e->context;
    int search = 0;
    LPCWSTR text = L"";
    switch (property) {
    case 0: break;
    case UIA_NamePropertyId:
    case UIA_AutomationIdPropertyId:
        if (value.vt != VT_BSTR) return E_INVALIDARG;
        search = property == UIA_NamePropertyId ? 1 : 3;
        text = value.bstrVal ? value.bstrVal : L"";
        break;
    case UIA_SelectionItemIsSelectedPropertyId:
        if (value.vt != VT_BOOL) return E_INVALIDARG;
        search = 2; text = value.boolVal == VARIANT_FALSE ? L"false" : L"true";
        break;
    default: return (HRESULT)UIA_E_NOTSUPPORTED;
    }
    uint32_t start = 0;
    if (startAfter) {
        // UIA passes back the provider returned by this container. Never treat
        // an unrelated provider or a different window's ID as our own object.
        if (startAfter->lpVtbl != &simpleVtbl) return E_INVALIDARG;
        Element *previous = OWNER(startAfter, simple);
        if (previous->context != c) return E_INVALIDARG;
        start = previous->id;
    }
    AcquireSRWLockShared(&c->lock);
    Record *r = find(c, e->id);
    int valid = alive(e) && r && (r->data.flags & WinAccItemContainer);
    HWND hwnd = c->hwnd; uintptr_t handle = c->handle;
    ReleaseSRWLockShared(&c->lock);
    if (!valid) return UNAVAILABLE;
    int size = WideCharToMultiByte(CP_UTF8, 0, text, -1, NULL, 0, NULL, NULL);
    char *utf8 = malloc(size);
    if (!utf8) return E_OUTOFMEMORY;
    WideCharToMultiByte(CP_UTF8, 0, text, -1, utf8, size, NULL, NULL);
    ActionRequest request = {.handle=handle, .id=e->id, .action=13, .text=utf8,
        .startAfter=start, .property=search, .result=UNAVAILABLE};
    SendMessageW(hwnd, actionMessage, 0, (LPARAM)&request);
    free(utf8);
    if (FAILED(request.result)) return request.result;
    if (!request.resultID) return S_OK;
    AcquireSRWLockShared(&c->lock);
    Element *target = !c->closed ? elementFor(c, request.resultID) : NULL;
    if (target) { addRef(target); *out = &target->simple; }
    ReleaseSRWLockShared(&c->lock);
    return target ? S_OK : UNAVAILABLE;
}

static IItemContainerProviderVtbl itemContainerVtbl = {IC_QI, IC_Add, IC_Release, findItem};
static IVirtualizedItemProviderVtbl virtualizedItemVtbl = {VI_QI, VI_Add, VI_Release, realizeItem};
