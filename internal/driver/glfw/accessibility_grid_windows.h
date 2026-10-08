// Private UIA grid/table providers. Discovery marshals to the event thread;
// ordinary properties use immutable snapshots and never call Go under a lock.
static HRESULT gridEntry(Element *e, int row, int column, IRawElementProviderSimple **out) {
    if (!out) return E_POINTER;
    *out = NULL;
    WinAccessibility *c = e->context;
    AcquireSRWLockShared(&c->lock);
    Record *r = find(c,e->id);
    HRESULT hr = !alive(e) ? UNAVAILABLE : !r || !(r->data.flags & WinAccGrid) ? (HRESULT)UIA_E_NOTSUPPORTED : S_OK;
    if (SUCCEEDED(hr) && (row < -1 || column < -1 || (row == -1 && column == -1) || row >= r->data.rows || column >= r->data.columns)) hr = E_INVALIDARG;
    if (SUCCEEDED(hr) && ((row == -1 && !(r->data.flags & WinAccColumnHeaders)) || (column == -1 && !(r->data.flags & WinAccRowHeaders)))) hr = E_INVALIDARG;
    HWND hwnd = c->hwnd; uintptr_t handle = c->handle;
    ReleaseSRWLockShared(&c->lock);
    if (FAILED(hr)) return hr;
    char coordinates[64]; snprintf(coordinates,sizeof(coordinates),"%d,%d",row,column);
    ActionRequest request = {.handle=handle,.id=e->id,.action=13,.property=4,.text=coordinates,.result=UNAVAILABLE};
    SendMessageW(hwnd,actionMessage,0,(LPARAM)&request);
    if (FAILED(request.result)) return request.result;
    AcquireSRWLockShared(&c->lock);
    Element *target = !c->closed ? elementFor(c,request.resultID) : NULL;
    if (target) { addRef(target); *out=&target->simple; }
    ReleaseSRWLockShared(&c->lock);
    return target ? S_OK : UNAVAILABLE;
}
static HRESULT STDMETHODCALLTYPE gridGetItem(IGridProvider *p, int row, int column, IRawElementProviderSimple **out) {
    if (!out) return E_POINTER;
    *out=NULL;
    if (row < 0 || column < 0) return E_INVALIDARG;
    return gridEntry(OWNER(p,grid),row,column,out);
}
static HRESULT gridInteger(Element *e, PROPERTYID id, int *out) {
    if (!out) return E_POINTER;
    *out=0;
    VARIANT v; HRESULT hr=property(&e->simple,id,&v);
    if (SUCCEEDED(hr) && v.vt!=VT_I4) hr=(HRESULT)UIA_E_NOTSUPPORTED;
    if (SUCCEEDED(hr)) *out=v.lVal;
    VariantClear(&v); return hr;
}
#define GRID_INT(fn, Type, member, pid) \
static HRESULT STDMETHODCALLTYPE fn(Type *p,int *out) { return gridInteger(OWNER(p,member),pid,out); }
GRID_INT(gridRows,IGridProvider,grid,UIA_GridRowCountPropertyId)
GRID_INT(gridColumns,IGridProvider,grid,UIA_GridColumnCountPropertyId)
GRID_INT(cellRow,IGridItemProvider,gridItem,UIA_GridItemRowPropertyId)
GRID_INT(cellColumn,IGridItemProvider,gridItem,UIA_GridItemColumnPropertyId)
GRID_INT(cellRowSpan,IGridItemProvider,gridItem,UIA_GridItemRowSpanPropertyId)
GRID_INT(cellColumnSpan,IGridItemProvider,gridItem,UIA_GridItemColumnSpanPropertyId)
#undef GRID_INT
static HRESULT STDMETHODCALLTYPE cellGrid(IGridItemProvider *p, IRawElementProviderSimple **out) {
    if (!out) return E_POINTER;
    *out=NULL;
    Element *e=OWNER(p,gridItem); WinAccessibility *c=e->context;
    AcquireSRWLockShared(&c->lock); Record *r=find(c,e->id);
    HRESULT hr=!alive(e) ? UNAVAILABLE : !r || !(r->data.flags & WinAccGridItem) ? (HRESULT)UIA_E_NOTSUPPORTED : S_OK;
    Element *owner=SUCCEEDED(hr) ? elementFor(c,r->data.grid_owner) : NULL;
    if (owner) { addRef(owner); *out=&owner->simple; } else if (SUCCEEDED(hr)) hr=UNAVAILABLE;
    ReleaseSRWLockShared(&c->lock); return hr;
}
static HRESULT tableHeaders(Element *e, int rowHeaders, int cell, SAFEARRAY **out) {
    if (!out) return E_POINTER;
    *out=NULL;
    WinAccessibility *c=e->context;
    AcquireSRWLockShared(&c->lock); Record *r=find(c,e->id);
    HRESULT hr=!alive(e) ? UNAVAILABLE : !r || !(r->data.flags & WinAccTable) ? (HRESULT)UIA_E_NOTSUPPORTED : S_OK;
    Record *grid=SUCCEEDED(hr) ? (cell ? find(c,r->data.grid_owner) : r) : NULL;
    int count=0, index=0;
    Element *owner=grid ? grid->element : NULL;
    if (grid && (grid->data.flags & (rowHeaders ? WinAccRowHeaders : WinAccColumnHeaders))) {
        count=cell ? 1 : rowHeaders ? grid->data.rows : grid->data.columns;
        if (cell) index=rowHeaders ? r->data.row : r->data.column;
    }
    if (owner) addRef(owner);
    ReleaseSRWLockShared(&c->lock);
    if (FAILED(hr)) { if (owner) release(owner); return hr; }
    SAFEARRAY *array=SafeArrayCreateVector(VT_UNKNOWN,0,count);
    if (!array) { if (owner) release(owner); return E_OUTOFMEMORY; }
    for (LONG j=0; j<count; ++j) {
        IRawElementProviderSimple *header=NULL;
        int coordinate=cell ? index : (int)j;
        hr=gridEntry(owner,rowHeaders ? coordinate : -1,rowHeaders ? -1 : coordinate,&header);
        if (FAILED(hr)) break;
        hr=SafeArrayPutElement(array,&j,header); IRawElementProviderSimple_Release(header);
        if (FAILED(hr)) break;
    }
    if (owner) release(owner);
    if (FAILED(hr)) { SafeArrayDestroy(array); return hr; }
    *out=array; return S_OK;
}
static HRESULT STDMETHODCALLTYPE tableRowHeaders(ITableProvider *p,SAFEARRAY **out) { return tableHeaders(OWNER(p,table),1,0,out); }
static HRESULT STDMETHODCALLTYPE tableColumnHeaders(ITableProvider *p,SAFEARRAY **out) { return tableHeaders(OWNER(p,table),0,0,out); }
static HRESULT STDMETHODCALLTYPE cellRowHeaders(ITableItemProvider *p,SAFEARRAY **out) { return tableHeaders(OWNER(p,tableItem),1,1,out); }
static HRESULT STDMETHODCALLTYPE cellColumnHeaders(ITableItemProvider *p,SAFEARRAY **out) { return tableHeaders(OWNER(p,tableItem),0,1,out); }
static HRESULT STDMETHODCALLTYPE tableMajor(ITableProvider *p,enum RowOrColumnMajor *out) {
    if (!out) return E_POINTER;
    int value=0; HRESULT hr=gridInteger(OWNER(p,table),UIA_TableRowOrColumnMajorPropertyId,&value);
    if (SUCCEEDED(hr)) *out=(enum RowOrColumnMajor)value;
    return hr;
}
static IGridProviderVtbl gridVtbl={GR_QI,GR_Add,GR_Release,gridGetItem,gridRows,gridColumns};
static IGridItemProviderVtbl gridItemVtbl={GI_QI,GI_Add,GI_Release,cellRow,cellColumn,cellRowSpan,cellColumnSpan,cellGrid};
static ITableProviderVtbl tableVtbl={TB_QI,TB_Add,TB_Release,tableRowHeaders,tableColumnHeaders,tableMajor};
static ITableItemProviderVtbl tableItemVtbl={TI_QI,TI_Add,TI_Release,cellRowHeaders,cellColumnHeaders};
