//go:build accessibility && windows

// Private implementation included by accessibility_windows.c. All text ranges
// refer to live element IDs, never widget memory or an obsolete text buffer.
static WCHAR *wide(const char *text);
// Older MinGW headers omit the documented UIAutomationClient.h text IDs.
#ifndef UIA_IsReadOnlyAttributeId
#define UIA_IsReadOnlyAttributeId 40015
#endif
struct TextRange {
    ITextRangeProvider iface;
    LONG refs;
    Element *element;
    int start, end; // rune offsets; UTF-16 conversion happens only at the boundary
    TextRange *previous, *next;
};
static ITextRangeProviderVtbl textRangeVtbl;
#define TEXT_RANGE(p) ((TextRange *)(p))
static int clampOffset(int value, int length) { return value < 0 ? 0 : value > length ? length : value; }
static int surrogatePair(const WCHAR *p) { return p[0] >= 0xd800 && p[0] <= 0xdbff && p[1] >= 0xdc00 && p[1] <= 0xdfff; }
static int copyText(Record *r, const WinAccessibilityNode *n) {
    r->text = wide(n->text);
    if (!r->text) return 0;
    int units = (int)wcslen(r->text);
    r->offsets = calloc((size_t)units + 1, sizeof(int));
    if (!r->offsets) return 0;
    for (int i = 0; i < units; i += surrogatePair(r->text + i) ? 2 : 1) r->offsets[r->length++] = i;
    r->offsets[r->length] = units;
    if (n->flags & WinAccProtected) {
        // Defense in depth: even an incorrectly implemented Go control cannot
        // leak a password via GetText, FindText, attributes or retained ranges.
        for (int i = 0; i < r->length; ++i) { r->text[i] = 0x2022; r->offsets[i] = i; }
        r->text[r->length] = 0; r->offsets[r->length] = r->length;
        r->wordBoundaries = calloc((size_t)r->length + 1, 1);
        if (!r->wordBoundaries) return 0;
        r->wordBoundaries[0] = r->wordBoundaries[r->length] = 1;
    } else if (n->word_boundaries && n->word_boundary_count > 0) {
        int valid = n->word_boundaries[0] == 0 && n->word_boundaries[n->word_boundary_count - 1] == r->length;
        for (int i = 1; valid && i < n->word_boundary_count; ++i)
            valid = n->word_boundaries[i] > n->word_boundaries[i-1] && n->word_boundaries[i] <= r->length;
        if (valid) {
            r->wordBoundaries = calloc((size_t)r->length + 1, 1);
            if (!r->wordBoundaries) return 0;
            for (int i = 0; i < n->word_boundary_count; ++i) r->wordBoundaries[n->word_boundaries[i]] = 1;
        }
    }
    r->data.caret = clampOffset(n->caret, r->length);
    r->data.selection_start = clampOffset(n->selection_start, r->length);
    r->data.selection_end = clampOffset(n->selection_end, r->length);
    if (r->data.selection_end < r->data.selection_start) r->data.selection_end = r->data.selection_start;
    if (n->positions && n->position_count == r->length + 1) {
        r->positions = malloc((size_t)n->position_count * sizeof(*r->positions));
        if (!r->positions) return 0;
        memcpy(r->positions, n->positions, (size_t)n->position_count * sizeof(*r->positions));
    }
    return 1;
}
static unsigned int runeAt(Record *r, int offset) {
    const WCHAR *p = r->text + r->offsets[offset];
    return surrogatePair(p) ? 0x10000 + ((p[0] - 0xd800) << 10) + p[1] - 0xdc00 : p[0];
}
static void updateTextRanges(Record *old, Record *next) {
    int prefix = 0, suffix = 0;
    while (prefix < old->length && prefix < next->length && runeAt(old, prefix) == runeAt(next, prefix)) ++prefix;
    while (suffix < old->length - prefix && suffix < next->length - prefix &&
           runeAt(old, old->length - suffix - 1) == runeAt(next, next->length - suffix - 1)) ++suffix;
    if (prefix == old->length && prefix == next->length) return;
    for (TextRange *range = old->element->textRanges; range; range = range->next) {
        int *ends[] = {&range->start, &range->end};
        for (int i = 0; i < 2; ++i) {
            int pos = *ends[i];
            if (pos >= old->length - suffix) pos += next->length - old->length;
            else if (pos > prefix) pos = prefix;
            *ends[i] = clampOffset(pos, next->length);
        }
        if (range->end < range->start) range->end = range->start;
    }
}
// Caller holds the exclusive context lock, including for range list mutations.
static Record *textRecord(Element *e) {
    Record *r = find(e->context, e->id);
    return !e->context->closed && r && (r->data.flags & WinAccText) ? r : NULL;
}
static TextRange *newTextRange(Element *e, int start, int end) {
    TextRange *range = calloc(1, sizeof(*range));
    if (!range) return NULL;
    range->iface.lpVtbl = &textRangeVtbl; range->refs = 1; range->element = e;
    range->start = start; range->end = end; addRef(e);
    range->next = e->textRanges;
    if (range->next) range->next->previous = range;
    e->textRanges = range;
    return range;
}
static HRESULT STDMETHODCALLTYPE TX_QI(ITextRangeProvider *p, REFIID iid, void **out) {
    if (!out) return E_POINTER;
    *out = NULL;
    if (!IsEqualIID(iid, &IID_IUnknown) && !IsEqualIID(iid, &IID_ITextRangeProvider)) return E_NOINTERFACE;
    *out = p; InterlockedIncrement(&TEXT_RANGE(p)->refs); return S_OK;
}
static ULONG STDMETHODCALLTYPE TX_Add(ITextRangeProvider *p) { return InterlockedIncrement(&TEXT_RANGE(p)->refs); }
static ULONG STDMETHODCALLTYPE TX_Release(ITextRangeProvider *p) {
    TextRange *r = TEXT_RANGE(p); ULONG refs = InterlockedDecrement(&r->refs);
    if (!refs) {
        Element *e = r->element; WinAccessibility *c = e->context;
        AcquireSRWLockExclusive(&c->lock);
        if (r->previous) r->previous->next = r->next; else e->textRanges = r->next;
        if (r->next) r->next->previous = r->previous;
        ReleaseSRWLockExclusive(&c->lock); free(r); release(e);
    }
    return refs;
}
static HRESULT textRangeResult(Element *e, int start, int end, ITextRangeProvider **out) {
    TextRange *r = newTextRange(e, start, end);
    *out = r ? &r->iface : NULL; return r ? S_OK : E_OUTOFMEMORY;
}
static HRESULT STDMETHODCALLTYPE textDocument(ITextProvider2 *p, ITextRangeProvider **out) {
    if (!out) return E_POINTER;
    *out = NULL; Element *e = OWNER(p, text); WinAccessibility *c = e->context;
    AcquireSRWLockExclusive(&c->lock); Record *r = textRecord(e);
    HRESULT hr = r ? textRangeResult(e, 0, r->length, out) : UNAVAILABLE;
    ReleaseSRWLockExclusive(&c->lock); return hr;
}
static HRESULT STDMETHODCALLTYPE textCaret(ITextProvider2 *p, BOOL *active, ITextRangeProvider **out) {
    if (!active || !out) return E_POINTER;
    *out = NULL; *active = FALSE;
    Element *e = OWNER(p, text); WinAccessibility *c = e->context;
    AcquireSRWLockExclusive(&c->lock); Record *r = textRecord(e);
    HRESULT hr = UNAVAILABLE;
    if (r) {
        *active = c->foreground && c->focus == e->id;
        hr = textRangeResult(e, r->data.caret, r->data.caret, out);
    }
    ReleaseSRWLockExclusive(&c->lock); return hr;
}
static HRESULT STDMETHODCALLTYPE textSelectionSupport(ITextProvider2 *p, enum SupportedTextSelection *out) {
    if (!out) return E_POINTER;
    Element *e = OWNER(p, text); AcquireSRWLockShared(&e->context->lock);
    HRESULT hr = textRecord(e) ? S_OK : UNAVAILABLE;
    *out = SupportedTextSelection_Single; ReleaseSRWLockShared(&e->context->lock); return hr;
}
static HRESULT STDMETHODCALLTYPE textNoChild(ITextProvider2 *p, IRawElementProviderSimple *child, ITextRangeProvider **out) {
    if (!out) return E_POINTER;
    *out = NULL; Element *e = OWNER(p, text); AcquireSRWLockShared(&e->context->lock);
    HRESULT hr = textRecord(e) ? E_INVALIDARG : UNAVAILABLE;
    ReleaseSRWLockShared(&e->context->lock); return hr;
}
static HRESULT STDMETHODCALLTYPE textSelection(ITextProvider2 *p, SAFEARRAY **out) {
    if (!out) return E_POINTER;
    *out = NULL; Element *e = OWNER(p, text); WinAccessibility *c = e->context;
    ITextRangeProvider *range = NULL;
    AcquireSRWLockExclusive(&c->lock); Record *r = textRecord(e);
    HRESULT hr = r ? textRangeResult(e, r->data.selection_start, r->data.selection_end, &range) : UNAVAILABLE;
    ReleaseSRWLockExclusive(&c->lock);
    if (FAILED(hr)) return hr;
    *out = SafeArrayCreateVector(VT_UNKNOWN, 0, 1);
    LONG i = 0;
    hr = *out ? SafeArrayPutElement(*out, &i, range) : E_OUTOFMEMORY;
    ITextRangeProvider_Release(range);
    if (FAILED(hr) && *out) { SafeArrayDestroy(*out); *out = NULL; }
    return hr;
}
// Text geometry is in client pixels and already accounts for Entry scrolling.
static int textRect(Record *r, int start, int end, struct UiaRect *rect) {
    if (!r->positions) return 0;
    WinAccessibilityTextPosition *a = &r->positions[start], *b = &r->positions[end];
    double left = a->x, right = end == start ? a->x + 1 : b->x;
    if (b->line != a->line) right = r->positions[end - 1].x + 1;
    if (right < left) { double swap = left; left = right; right = swap; }
    double top = a->y, bottom = top + a->height;
    WinAccessibilityNode *n = &r->data;
    if (left < n->viewport_x) left = n->viewport_x;
    if (top < n->viewport_y) top = n->viewport_y;
    if (right > n->viewport_x + n->viewport_width) right = n->viewport_x + n->viewport_width;
    if (bottom > n->viewport_y + n->viewport_height) bottom = n->viewport_y + n->viewport_height;
    if (right <= left || bottom <= top) return 0;
    rect->left = left; rect->top = top; rect->width = right - left; rect->height = bottom - top; return 1;
}
static HRESULT STDMETHODCALLTYPE textVisible(ITextProvider2 *p, SAFEARRAY **out) {
    if (!out) return E_POINTER;
    *out = NULL; Element *e = OWNER(p, text); WinAccessibility *c = e->context;
    AcquireSRWLockExclusive(&c->lock); Record *r = textRecord(e);
    if (!r) { ReleaseSRWLockExclusive(&c->lock); return UNAVAILABLE; }
    // One range for each contiguous visible run. The array owns COM references.
    ITextRangeProvider **ranges = calloc((size_t)r->length + 1, sizeof(*ranges));
    if (!ranges) { ReleaseSRWLockExclusive(&c->lock); return E_OUTOFMEMORY; }
    int count = 0, start = -1; HRESULT hr = S_OK;
    for (int i = 0; i <= r->length; ++i) {
        struct UiaRect rect;
        int visible = textRect(r, i, i < r->length ? i + 1 : i, &rect);
        if (visible && start < 0) start = i;
        if ((!visible || i == r->length) && start >= 0) {
            hr = textRangeResult(e, start, i, &ranges[count]);
            if (FAILED(hr)) break;
            ++count; start = -1;
        }
    }
    ReleaseSRWLockExclusive(&c->lock);
    if (SUCCEEDED(hr)) {
        *out = SafeArrayCreateVector(VT_UNKNOWN, 0, count);
        if (!*out) hr = E_OUTOFMEMORY;
        for (LONG i = 0; SUCCEEDED(hr) && i < count; ++i) hr = SafeArrayPutElement(*out, &i, ranges[i]);
    }
    for (int i = 0; i < count; ++i) ITextRangeProvider_Release(ranges[i]);
    free(ranges);
    if (FAILED(hr) && *out) { SafeArrayDestroy(*out); *out = NULL; }
    return hr;
}
static HRESULT STDMETHODCALLTYPE textPoint(ITextProvider2 *p, struct UiaPoint point, ITextRangeProvider **out) {
    if (!out) return E_POINTER;
    *out = NULL; Element *e = OWNER(p, text); WinAccessibility *c = e->context;
    AcquireSRWLockExclusive(&c->lock); Record *r = textRecord(e);
    if (!r) { ReleaseSRWLockExclusive(&c->lock); return UNAVAILABLE; }
    HRESULT hr = (HRESULT)UIA_E_NOTSUPPORTED;
    if (r->positions) {
        POINT origin = {0, 0}; ClientToScreen(c->hwnd, &origin);
        point.x -= origin.x; point.y -= origin.y;
        int best = 0; double distance = HUGE_VAL;
        for (int i = 0; i <= r->length; ++i) {
            WinAccessibilityTextPosition *pos = &r->positions[i];
            double dy = point.y < pos->y ? pos->y - point.y : point.y > pos->y + pos->height ? point.y - pos->y - pos->height : 0;
            double dx = fabs(point.x - pos->x);
            double d = dy * 1000000 + dx;
            if (d < distance) { distance = d; best = i; }
        }
        hr = textRangeResult(e, best, best, out);
    }
    ReleaseSRWLockExclusive(&c->lock); return hr;
}
static HRESULT STDMETHODCALLTYPE rangeClone(ITextRangeProvider *p, ITextRangeProvider **out) {
    if (!out) return E_POINTER;
    *out = NULL; TextRange *r = TEXT_RANGE(p); WinAccessibility *c = r->element->context;
    AcquireSRWLockExclusive(&c->lock);
    HRESULT hr = textRecord(r->element) ? textRangeResult(r->element, r->start, r->end, out) : UNAVAILABLE;
    ReleaseSRWLockExclusive(&c->lock); return hr;
}
static int peerRange(TextRange *r, ITextRangeProvider *other) {
    return other && other->lpVtbl == &textRangeVtbl && TEXT_RANGE(other)->element == r->element;
}
static int validEndpoint(enum TextPatternRangeEndpoint endpoint) {
    return endpoint == TextPatternRangeEndpoint_Start || endpoint == TextPatternRangeEndpoint_End;
}
static HRESULT STDMETHODCALLTYPE rangeCompare(ITextRangeProvider *p, ITextRangeProvider *other, BOOL *out) {
    if (!out) return E_POINTER;
    *out = FALSE; TextRange *r = TEXT_RANGE(p);
    if (!peerRange(r, other)) return E_INVALIDARG;
    WinAccessibility *c = r->element->context; AcquireSRWLockExclusive(&c->lock);
    HRESULT hr = textRecord(r->element) ? S_OK : UNAVAILABLE;
    if (SUCCEEDED(hr)) *out = r->start == TEXT_RANGE(other)->start && r->end == TEXT_RANGE(other)->end;
    ReleaseSRWLockExclusive(&c->lock); return hr;
}
static int endpointValue(TextRange *r, enum TextPatternRangeEndpoint endpoint) { return endpoint == TextPatternRangeEndpoint_Start ? r->start : r->end; }
static void setEndpoint(TextRange *r, enum TextPatternRangeEndpoint endpoint, int value) {
    if (endpoint == TextPatternRangeEndpoint_Start) { r->start = value; if (r->end < value) r->end = value; }
    else { r->end = value; if (r->start > value) r->start = value; }
}
static HRESULT STDMETHODCALLTYPE rangeCompareEnds(ITextRangeProvider *p, enum TextPatternRangeEndpoint endpoint,
    ITextRangeProvider *other, enum TextPatternRangeEndpoint otherEndpoint, int *out) {
    if (!out) return E_POINTER;
    *out = 0; TextRange *r = TEXT_RANGE(p);
    if (!peerRange(r, other) || !validEndpoint(endpoint) || !validEndpoint(otherEndpoint)) return E_INVALIDARG;
    WinAccessibility *c = r->element->context; AcquireSRWLockExclusive(&c->lock);
    HRESULT hr = textRecord(r->element) ? S_OK : UNAVAILABLE;
    if (SUCCEEDED(hr)) *out = endpointValue(r, endpoint) - endpointValue(TEXT_RANGE(other), otherEndpoint);
    ReleaseSRWLockExclusive(&c->lock); return hr;
}
static HRESULT STDMETHODCALLTYPE rangeMoveEndRange(ITextRangeProvider *p, enum TextPatternRangeEndpoint endpoint,
    ITextRangeProvider *other, enum TextPatternRangeEndpoint otherEndpoint) {
    TextRange *r = TEXT_RANGE(p);
    if (!peerRange(r, other) || !validEndpoint(endpoint) || !validEndpoint(otherEndpoint)) return E_INVALIDARG;
    WinAccessibility *c = r->element->context; AcquireSRWLockExclusive(&c->lock);
    HRESULT hr = textRecord(r->element) ? S_OK : UNAVAILABLE;
    if (SUCCEEDED(hr)) setEndpoint(r, endpoint, endpointValue(TEXT_RANGE(other), otherEndpoint));
    ReleaseSRWLockExclusive(&c->lock); return hr;
}
static int runeClass(Record *r, int pos) {
    unsigned int cp = runeAt(r, pos); WORD kind = 0;
    if (cp > 0xffff) return 2;
    WCHAR ch = (WCHAR)cp; GetStringTypeW(CT_CTYPE1, &ch, 1, &kind);
    return kind & C1_SPACE ? 0 : kind & (C1_ALPHA | C1_DIGIT) ? 1 : 2;
}
static int unitBoundary(Record *r, enum TextUnit unit, int pos) {
    if (pos == 0 || pos == r->length) return 1;
    switch (unit) {
    case TextUnit_Character: return 1;
    case TextUnit_Word: {
        if (r->wordBoundaries) return r->wordBoundaries[pos] != 0;
        int before = runeClass(r, pos - 1), after = runeClass(r, pos);
        return after != 0 && before != after;
    }
    case TextUnit_Line:
        if (r->positions) return r->positions[pos - 1].line != r->positions[pos].line;
        // Without layout, a line is a newline-delimited paragraph.
        /* fall through */
    case TextUnit_Paragraph: return runeAt(r, pos - 1) == '\n';
    default: return 0; // uniform formatting; page falls back to the document
    }
}
static int validUnit(enum TextUnit unit) { return unit >= TextUnit_Character && unit <= TextUnit_Document; }
static int nextBoundary(Record *r, enum TextUnit unit, int pos, int direction) {
    int next = pos;
    do { next += direction; } while (next > 0 && next < r->length && !unitBoundary(r, unit, next));
    return clampOffset(next, r->length);
}
static void expandUnit(TextRange *range, Record *r, enum TextUnit unit) {
    int start = range->start;
    if ((unit == TextUnit_Character || unit == TextUnit_Word) && start == r->length) { range->end = start; return; }
    // Expanding the end-of-document caret selects the final unit, if any.
    if (start == r->length && start > 0) --start;
    while (start > 0 && !unitBoundary(r, unit, start)) --start;
    range->start = start; range->end = nextBoundary(r, unit, start, 1);
}
static HRESULT STDMETHODCALLTYPE rangeExpand(ITextRangeProvider *p, enum TextUnit unit) {
    if (!validUnit(unit)) return E_INVALIDARG;
    TextRange *range = TEXT_RANGE(p); WinAccessibility *c = range->element->context;
    AcquireSRWLockExclusive(&c->lock); Record *r = textRecord(range->element);
    if (r) expandUnit(range, r, unit);
    ReleaseSRWLockExclusive(&c->lock); return r ? S_OK : UNAVAILABLE;
}
static HRESULT STDMETHODCALLTYPE rangeMoveEnd(ITextRangeProvider *p, enum TextPatternRangeEndpoint endpoint, enum TextUnit unit, int count, int *out) {
    if (!out) return E_POINTER;
    *out = 0;
    if (!validUnit(unit) || !validEndpoint(endpoint)) return E_INVALIDARG;
    TextRange *range = TEXT_RANGE(p); WinAccessibility *c = range->element->context;
    AcquireSRWLockExclusive(&c->lock); Record *r = textRecord(range->element);
    if (r) {
        int position = endpointValue(range, endpoint), direction = count < 0 ? -1 : 1;
        while (*out != count) {
            int next = nextBoundary(r, unit, position, direction);
            if (next == position) break;
            position = next; *out += direction;
        }
        setEndpoint(range, endpoint, position);
    }
    ReleaseSRWLockExclusive(&c->lock); return r ? S_OK : UNAVAILABLE;
}
static HRESULT STDMETHODCALLTYPE rangeMove(ITextRangeProvider *p, enum TextUnit unit, int count, int *out) {
    if (!out) return E_POINTER;
    *out = 0;
    if (!validUnit(unit)) return E_INVALIDARG;
    TextRange *range = TEXT_RANGE(p); WinAccessibility *c = range->element->context;
    AcquireSRWLockExclusive(&c->lock); Record *r = textRecord(range->element);
    if (r && count != 0) {
        int expanded = range->start != range->end, direction = count < 0 ? -1 : 1;
        if (expanded) expandUnit(range, r, unit);
        int pos = range->start;
        while (*out != count) {
            int next = nextBoundary(r, unit, pos, direction);
            if (next == pos || (expanded && next == r->length)) break;
            pos = next; *out += direction;
        }
        range->start = range->end = pos;
        if (expanded) expandUnit(range, r, unit);
    }
    ReleaseSRWLockExclusive(&c->lock); return r ? S_OK : UNAVAILABLE;
}
static HRESULT STDMETHODCALLTYPE rangeText(ITextRangeProvider *p, int maximum, BSTR *out) {
    if (!out) return E_POINTER;
    *out = NULL; if (maximum < -1) return E_INVALIDARG;
    TextRange *range = TEXT_RANGE(p); WinAccessibility *c = range->element->context;
    AcquireSRWLockExclusive(&c->lock); Record *r = textRecord(range->element); HRESULT hr = UNAVAILABLE;
    if (r) {
        int start = r->offsets[range->start], end = r->offsets[range->end];
        if (maximum >= 0 && end - start > maximum) {
            end = start + maximum;
            if (end > start && r->text[end - 1] >= 0xd800 && r->text[end - 1] <= 0xdbff) --end;
        }
        *out = SysAllocStringLen(r->text + start, end - start); hr = *out ? S_OK : E_OUTOFMEMORY;
    }
    ReleaseSRWLockExclusive(&c->lock); return hr;
}
static HRESULT STDMETHODCALLTYPE rangeFindText(ITextRangeProvider *p, BSTR needle, BOOL backwards, BOOL ignoreCase, ITextRangeProvider **out) {
    if (!out) return E_POINTER;
    *out = NULL; if (!needle || !SysStringLen(needle)) return E_INVALIDARG;
    TextRange *range = TEXT_RANGE(p); WinAccessibility *c = range->element->context;
    AcquireSRWLockExclusive(&c->lock); Record *r = textRecord(range->element);
    if (!r) { ReleaseSRWLockExclusive(&c->lock); return UNAVAILABLE; }
    int length = (int)SysStringLen(needle); HRESULT hr = S_OK;
    for (int i = backwards ? range->end : range->start; i >= range->start && i <= range->end; i += backwards ? -1 : 1) {
        int endUnit = r->offsets[i] + length;
        if (endUnit > r->offsets[range->end]) continue;
        int matched = CompareStringOrdinal(r->text + r->offsets[i], length, needle, length, ignoreCase) == CSTR_EQUAL;
        if (matched) {
            int end = i;
            while (end < range->end && r->offsets[end] < endUnit) ++end;
            if (r->offsets[end] != endUnit) continue;
            hr = textRangeResult(range->element, i, end, out); break;
        }
    }
    ReleaseSRWLockExclusive(&c->lock); return hr;
}
static HRESULT textAttribute(int flags, TEXTATTRIBUTEID attribute, VARIANT *out) {
    VariantInit(out);
    if (attribute == UIA_IsReadOnlyAttributeId) variantBool(out, flags & WinAccReadOnly);
    else {
        IUnknown *value = NULL; HRESULT hr = uiaNotSupported(&value);
        if (FAILED(hr)) return hr;
        out->vt = VT_UNKNOWN; out->punkVal = value;
    }
    return S_OK;
}
static HRESULT STDMETHODCALLTYPE rangeAttribute(ITextRangeProvider *p, TEXTATTRIBUTEID attribute, VARIANT *out) {
    if (!out) return E_POINTER;
    VariantInit(out); TextRange *range = TEXT_RANGE(p); WinAccessibility *c = range->element->context;
    AcquireSRWLockExclusive(&c->lock); Record *r = textRecord(range->element);
    int flags = r ? r->data.flags : 0, valid = r != NULL;
    ReleaseSRWLockExclusive(&c->lock);
    return valid ? textAttribute(flags, attribute, out) : UNAVAILABLE;
}
static HRESULT STDMETHODCALLTYPE rangeFindAttribute(ITextRangeProvider *p, TEXTATTRIBUTEID attribute, VARIANT value, BOOL backwards, ITextRangeProvider **out) {
    if (!out) return E_POINTER;
    *out = NULL; TextRange *range = TEXT_RANGE(p); WinAccessibility *c = range->element->context;
    AcquireSRWLockExclusive(&c->lock); Record *r = textRecord(range->element); HRESULT hr = r ? S_OK : UNAVAILABLE;
    if (r && attribute == UIA_IsReadOnlyAttributeId && value.vt == VT_BOOL &&
        (value.boolVal != VARIANT_FALSE) == ((r->data.flags & WinAccReadOnly) != 0))
        hr = textRangeResult(range->element, range->start, range->end, out);
    ReleaseSRWLockExclusive(&c->lock); return hr;
}
static HRESULT STDMETHODCALLTYPE rangeEnclosing(ITextRangeProvider *p, IRawElementProviderSimple **out) {
    if (!out) return E_POINTER;
    *out = NULL; TextRange *range = TEXT_RANGE(p); Element *e = range->element;
    AcquireSRWLockShared(&e->context->lock); HRESULT hr = textRecord(e) ? S_OK : UNAVAILABLE;
    if (SUCCEEDED(hr)) { addRef(e); *out = &e->simple; }
    ReleaseSRWLockShared(&e->context->lock); return hr;
}
static HRESULT STDMETHODCALLTYPE rangeChildren(ITextRangeProvider *p, SAFEARRAY **out) {
    if (!out) return E_POINTER;
    *out = NULL; TextRange *range = TEXT_RANGE(p); Element *e = range->element;
    AcquireSRWLockShared(&e->context->lock); HRESULT hr = textRecord(e) ? S_OK : UNAVAILABLE;
    if (SUCCEEDED(hr)) { *out = SafeArrayCreateVector(VT_UNKNOWN, 0, 0); if (!*out) hr = E_OUTOFMEMORY; }
    ReleaseSRWLockShared(&e->context->lock); return hr;
}
static HRESULT STDMETHODCALLTYPE rangeRectangles(ITextRangeProvider *p, SAFEARRAY **out) {
    if (!out) return E_POINTER;
    *out = NULL; TextRange *range = TEXT_RANGE(p); WinAccessibility *c = range->element->context;
    AcquireSRWLockExclusive(&c->lock); Record *r = textRecord(range->element);
    if (!r) { ReleaseSRWLockExclusive(&c->lock); return UNAVAILABLE; }
    int capacity = range->end - range->start + 1, count = 0;
    double *values = malloc((size_t)capacity * 4 * sizeof(double));
    if (!values) { ReleaseSRWLockExclusive(&c->lock); return E_OUTOFMEMORY; }
    POINT origin = {0, 0}; ClientToScreen(c->hwnd, &origin);
    for (int start = range->start; start <= range->end;) {
        int end = start;
        if (r->positions) while (end < range->end && r->positions[end].line == r->positions[start].line) ++end;
        struct UiaRect rect;
        if (textRect(r, start, end, &rect)) {
            values[count++] = origin.x + rect.left; values[count++] = origin.y + rect.top;
            values[count++] = rect.width; values[count++] = rect.height;
        }
        if (end >= range->end || end == start) break;
        start = end;
    }
    ReleaseSRWLockExclusive(&c->lock);
    *out = SafeArrayCreateVector(VT_R8, 0, count); HRESULT hr = *out ? S_OK : E_OUTOFMEMORY;
    for (LONG i = 0; SUCCEEDED(hr) && i < count; ++i) hr = SafeArrayPutElement(*out, &i, &values[i]);
    free(values);
    if (FAILED(hr) && *out) { SafeArrayDestroy(*out); *out = NULL; }
    return hr;
}
static HRESULT rangeCommand(TextRange *range, int scroll, int alignTop) {
    Element *e = range->element; WinAccessibility *c = e->context;
    AcquireSRWLockExclusive(&c->lock); Record *r = textRecord(e);
    HRESULT hr = !r ? UNAVAILABLE : !scroll && (r->data.flags & WinAccDisabled) ? (HRESULT)UIA_E_ELEMENTNOTENABLED : S_OK;
    ActionRequest request = {.handle=c->handle, .id=e->id, .action=5, .result=UNAVAILABLE,
        .start=range->start, .end=range->end, .scroll=scroll, .alignTop=alignTop};
    HWND hwnd = c->hwnd; ReleaseSRWLockExclusive(&c->lock);
    if (FAILED(hr)) return hr;
    SendMessageW(hwnd, actionMessage, 0, (LPARAM)&request); return request.result;
}
static HRESULT STDMETHODCALLTYPE rangeSelect(ITextRangeProvider *p) { return rangeCommand(TEXT_RANGE(p), 0, 0); }
static HRESULT STDMETHODCALLTYPE rangeScroll(ITextRangeProvider *p, BOOL alignTop) { return rangeCommand(TEXT_RANGE(p), 1, alignTop); }
static HRESULT STDMETHODCALLTYPE rangeSelectionUnsupported(ITextRangeProvider *p) {
    TextRange *range = TEXT_RANGE(p); WinAccessibility *c = range->element->context;
    AcquireSRWLockShared(&c->lock); HRESULT hr = textRecord(range->element) ? (HRESULT)UIA_E_INVALIDOPERATION : UNAVAILABLE;
    ReleaseSRWLockShared(&c->lock); return hr; // Single selection: use Select instead.
}
static ITextProvider2Vtbl textVtbl = {X_QI, X_Add, X_Release, textSelection, textVisible, textNoChild, textPoint,
    textDocument, textSelectionSupport, textNoChild, textCaret};
static ITextRangeProviderVtbl textRangeVtbl = {TX_QI, TX_Add, TX_Release, rangeClone, rangeCompare,
    rangeCompareEnds, rangeExpand, rangeFindAttribute, rangeFindText, rangeAttribute, rangeRectangles,
    rangeEnclosing, rangeText, rangeMove, rangeMoveEnd, rangeMoveEndRange, rangeSelect,
    rangeSelectionUnsupported, rangeSelectionUnsupported, rangeScroll, rangeChildren};
