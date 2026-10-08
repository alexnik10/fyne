// Private implementation included by accessibility_windows.c.
// All selection relationships resolve against the current immutable snapshot.

// The caller holds the context lock. SAFEARRAY owns its provider references.
static HRESULT selectionArray(Element *e, SAFEARRAY **out) {
    *out = NULL;
    WinAccessibility *c = e->context;
    Record *owner = find(c, e->id);
    if (!alive(e)) return UNAVAILABLE;
    if (!owner || !(owner->data.flags & WinAccSelection)) return (HRESULT)UIA_E_NOTSUPPORTED;
    LONG count = 0;
    for (int i = 0; i < c->count; ++i) {
        WinAccessibilityNode *n = &c->records[i].data;
        if (n->selection_owner == e->id && (n->flags & WinAccSelectable) && (n->flags & WinAccSelected)) ++count;
    }
    SAFEARRAY *array = SafeArrayCreateVector(VT_UNKNOWN, 0, count);
    if (!array) return E_OUTOFMEMORY;
    LONG index = 0;
    for (int i = 0; i < c->count; ++i) {
        Record *r = &c->records[i];
        if (r->data.selection_owner != e->id || !(r->data.flags & WinAccSelectable) || !(r->data.flags & WinAccSelected)) continue;
        HRESULT hr = SafeArrayPutElement(array, &index, &r->element->simple);
        if (FAILED(hr)) { SafeArrayDestroy(array); return hr; }
        ++index;
    }
    *out = array;
    return S_OK;
}
static HRESULT STDMETHODCALLTYPE getSelection(ISelectionProvider *p, SAFEARRAY **out) {
    if (!out) return E_POINTER;
    Element *e = OWNER(p, selection);
    AcquireSRWLockShared(&e->context->lock);
    HRESULT hr = selectionArray(e, out);
    ReleaseSRWLockShared(&e->context->lock);
    return hr;
}
static HRESULT STDMETHODCALLTYPE canSelectMultiple(ISelectionProvider *p, BOOL *out) {
    return readOnly(OWNER(p, selection), UIA_SelectionCanSelectMultiplePropertyId, out);
}
static HRESULT STDMETHODCALLTYPE selectionRequired(ISelectionProvider *p, BOOL *out) {
    return readOnly(OWNER(p, selection), UIA_SelectionIsSelectionRequiredPropertyId, out);
}
static HRESULT STDMETHODCALLTYPE selectItem(ISelectionItemProvider *p) { return action(OWNER(p, selectionItem), 6, NULL, 0); }
static HRESULT STDMETHODCALLTYPE addSelection(ISelectionItemProvider *p) { return action(OWNER(p, selectionItem), 7, NULL, 0); }
static HRESULT STDMETHODCALLTYPE removeSelection(ISelectionItemProvider *p) { return action(OWNER(p, selectionItem), 8, NULL, 0); }
static HRESULT STDMETHODCALLTYPE itemSelected(ISelectionItemProvider *p, BOOL *out) {
    return readOnly(OWNER(p, selectionItem), UIA_SelectionItemIsSelectedPropertyId, out);
}
static HRESULT STDMETHODCALLTYPE selectionContainer(ISelectionItemProvider *p, IRawElementProviderSimple **out) {
    if (!out) return E_POINTER;
    *out = NULL;
    Element *e = OWNER(p, selectionItem); WinAccessibility *c = e->context;
    AcquireSRWLockShared(&c->lock);
    Record *r = find(c, e->id);
    HRESULT hr = !alive(e) ? UNAVAILABLE : !r || !(r->data.flags & WinAccSelectable) ? (HRESULT)UIA_E_NOTSUPPORTED : S_OK;
    if (SUCCEEDED(hr)) {
        Record *owner = find(c, r->data.selection_owner);
        if (owner && (owner->data.flags & WinAccSelection)) { addRef(owner->element); *out = &owner->element->simple; }
    }
    ReleaseSRWLockShared(&c->lock);
    return hr;
}
static HRESULT STDMETHODCALLTYPE expandControl(IExpandCollapseProvider *p) { return action(OWNER(p, expand), 9, NULL, 0); }
static HRESULT STDMETHODCALLTYPE collapseControl(IExpandCollapseProvider *p) { return action(OWNER(p, expand), 10, NULL, 0); }
static HRESULT STDMETHODCALLTYPE expansionState(IExpandCollapseProvider *p, enum ExpandCollapseState *out) {
    if (!out) return E_POINTER;
    VARIANT v; HRESULT hr = property(&OWNER(p, expand)->simple, UIA_ExpandCollapseExpandCollapseStatePropertyId, &v);
    if (SUCCEEDED(hr) && v.vt != VT_I4) return (HRESULT)UIA_E_NOTSUPPORTED;
    if (SUCCEEDED(hr)) *out = (enum ExpandCollapseState)v.lVal;
    return hr;
}
static ISelectionProviderVtbl selectionVtbl = {SL_QI, SL_Add, SL_Release, getSelection, canSelectMultiple, selectionRequired};
static ISelectionItemProviderVtbl selectionItemVtbl = {SI_QI, SI_Add, SI_Release, selectItem, addSelection, removeSelection, itemSelected, selectionContainer};
static IExpandCollapseProviderVtbl expandVtbl = {EC_QI, EC_Add, EC_Release, expandControl, collapseControl, expansionState};
static HRESULT STDMETHODCALLTYPE scrollIntoView(IScrollItemProvider *p) { return action(OWNER(p, scrollItem), 11, NULL, 0); }
static IScrollItemProviderVtbl scrollItemVtbl = {SC_QI, SC_Add, SC_Release, scrollIntoView};
