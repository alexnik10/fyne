// Private UIA Scroll provider. Queries use immutable snapshots; commands are
// synchronously marshalled to the HWND thread and revalidated by the Go tree.
static int scrollAmount(enum ScrollAmount amount) {
    switch (amount) {
    case ScrollAmount_NoAmount: return 0;
    case ScrollAmount_SmallDecrement: return 1;
    case ScrollAmount_SmallIncrement: return 2;
    case ScrollAmount_LargeDecrement: return 3;
    case ScrollAmount_LargeIncrement: return 4;
    default: return -1;
    }
}
static const double scrollNoPercent = -1.0;
static int validScrollPercent(double value) {
    return isfinite(value) && (value == scrollNoPercent || (value >= 0 && value <= 100));
}
static HRESULT scrollCommand(Element *e, int relative, double horizontal, double vertical) {
    WinAccessibility *c = e->context;
    AcquireSRWLockShared(&c->lock);
    Record *r = find(c, e->id);
    HRESULT hr = !alive(e) ? UNAVAILABLE : !r || !(r->data.flags & WinAccScroll) ? (HRESULT)UIA_E_NOTSUPPORTED : S_OK;
    if (SUCCEEDED(hr) && (r->data.flags & WinAccDisabled)) hr = (HRESULT)UIA_E_ELEMENTNOTENABLED;
    double unchanged = relative ? 0 : scrollNoPercent;
    if (SUCCEEDED(hr) && ((horizontal != unchanged && r->data.horizontal_percent < 0) ||
                          (vertical != unchanged && r->data.vertical_percent < 0))) hr = (HRESULT)UIA_E_INVALIDOPERATION;
    HWND hwnd = c->hwnd; uintptr_t handle = c->handle;
    ReleaseSRWLockShared(&c->lock);
    if (FAILED(hr)) return hr;
    ActionRequest request = {.handle=handle, .id=e->id, .action=14, .scroll=relative,
        .horizontal=horizontal, .vertical=vertical, .result=UNAVAILABLE};
    SendMessageW(hwnd, actionMessage, 0, (LPARAM)&request);
    return request.result;
}
static HRESULT STDMETHODCALLTYPE scrollBy(IScrollProvider *p, enum ScrollAmount horizontal, enum ScrollAmount vertical) {
    int x = scrollAmount(horizontal), y = scrollAmount(vertical);
    if (x < 0 || y < 0) return E_INVALIDARG;
    return scrollCommand(OWNER(p, scroll), 1, x, y);
}
static HRESULT STDMETHODCALLTYPE scrollTo(IScrollProvider *p, double horizontal, double vertical) {
    if (!validScrollPercent(horizontal) || !validScrollPercent(vertical)) return E_INVALIDARG;
    return scrollCommand(OWNER(p, scroll), 0, horizontal, vertical);
}
static HRESULT scrollNumber(Element *e, PROPERTYID id, double *out) {
    if (!out) return E_POINTER;
    *out=0;
    VARIANT v; HRESULT hr=property(&e->simple,id,&v);
    if (SUCCEEDED(hr) && v.vt != VT_R8) hr=(HRESULT)UIA_E_NOTSUPPORTED;
    if (SUCCEEDED(hr)) *out=v.dblVal;
    VariantClear(&v); return hr;
}
#define SCROLL_NUMBER(fn, pid) \
static HRESULT STDMETHODCALLTYPE fn(IScrollProvider *p, double *out) { return scrollNumber(OWNER(p,scroll),pid,out); }
SCROLL_NUMBER(scrollHorizontalPercent, UIA_ScrollHorizontalScrollPercentPropertyId)
SCROLL_NUMBER(scrollVerticalPercent, UIA_ScrollVerticalScrollPercentPropertyId)
SCROLL_NUMBER(scrollHorizontalView, UIA_ScrollHorizontalViewSizePropertyId)
SCROLL_NUMBER(scrollVerticalView, UIA_ScrollVerticalViewSizePropertyId)
#undef SCROLL_NUMBER
static HRESULT STDMETHODCALLTYPE scrollHorizontal(IScrollProvider *p, BOOL *out) {
    return readOnly(OWNER(p,scroll), UIA_ScrollHorizontallyScrollablePropertyId, out);
}
static HRESULT STDMETHODCALLTYPE scrollVertical(IScrollProvider *p, BOOL *out) {
    return readOnly(OWNER(p,scroll), UIA_ScrollVerticallyScrollablePropertyId, out);
}
static IScrollProviderVtbl scrollVtbl = {SP_QI, SP_Add, SP_Release, scrollBy, scrollTo,
    scrollHorizontalPercent, scrollVerticalPercent, scrollHorizontalView, scrollVerticalView, scrollHorizontal, scrollVertical};
