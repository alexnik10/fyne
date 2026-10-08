//go:build accessibility && windows

// Private text formatting helpers. Runs own copied scalar data only.
#ifndef UIA_FontSizeAttributeId
#define UIA_FontSizeAttributeId 40006
#define UIA_FontWeightAttributeId 40007
#define UIA_ForegroundColorAttributeId 40008
#define UIA_HorizontalTextAlignmentAttributeId 40009
#define UIA_IsItalicAttributeId 40014
#define UIA_StrikethroughStyleAttributeId 40026
#define UIA_UnderlineStyleAttributeId 40030
#define UIA_StyleIdAttributeId 40034
#endif

static int copyTextFormats(Record *r, const WinAccessibilityNode *n) {
    if ((n->flags & WinAccProtected) || !n->runs || n->run_count <= 0 || n->run_count > r->length + 1) return 1;
    int end = 0;
    for (int i = 0; i < n->run_count; ++i) {
        const WinAccessibilityTextRun *run = &n->runs[i];
        if (run->start != end || run->end < end || run->end > r->length ||
            (run->end == end && r->length) || !isfinite(run->size) || run->size <= 0) return 1;
        end = run->end;
    }
    if (end != r->length) return 1;
    r->runs = malloc((size_t)n->run_count * sizeof(*r->runs));
    if (!r->runs) return 0;
    memcpy(r->runs, n->runs, (size_t)n->run_count * sizeof(*r->runs));
    r->runCount = n->run_count;
    return 1;
}
static int sameTextFormat(const WinAccessibilityTextRun *a, const WinAccessibilityTextRun *b) {
    return a->size == b->size && a->weight == b->weight && a->italic == b->italic &&
        a->underline == b->underline && a->strike == b->strike && a->monospace == b->monospace &&
        a->alignment == b->alignment && a->heading == b->heading && a->foreground == b->foreground;
}
static int textFormatsChanged(Record *a, Record *b) {
    if (a->runCount != b->runCount) return 1;
    for (int i = 0; i < a->runCount; ++i)
        if (a->runs[i].start != b->runs[i].start || a->runs[i].end != b->runs[i].end || !sameTextFormat(&a->runs[i], &b->runs[i])) return 1;
    return 0;
}
static int textFormatIndex(Record *r, int offset) {
    if (!r->runCount) return -1;
    offset = clampOffset(offset, r->length ? r->length - 1 : 0);
    int lo = 0, hi = r->runCount;
    while (lo < hi) {
        int mid = lo + (hi - lo) / 2;
        if (r->runs[mid].start <= offset) lo = mid + 1; else hi = mid;
    }
    return lo - 1;
}
static int textFormatBoundary(Record *r, int offset) {
    int index = textFormatIndex(r, offset);
    return index > 0 && r->runs[index].start == offset && !sameTextFormat(&r->runs[index-1], &r->runs[index]);
}
// Unknown attributes remain unsupported, not guessed from the theme resource name.
static int textAttributeAt(Record *r, int offset, TEXTATTRIBUTEID attribute, VARIANT *out) {
    VariantInit(out);
    if (attribute == UIA_IsReadOnlyAttributeId) { variantBool(out, r->data.flags & WinAccReadOnly); return 1; }
    int index = textFormatIndex(r, offset);
    if (index < 0) return 0;
    WinAccessibilityTextRun *run = &r->runs[index];
    switch (attribute) {
    case UIA_FontSizeAttributeId: variantNumber(out, run->size); break;
    case UIA_FontWeightAttributeId: integer(out, run->weight); break;
    case UIA_IsItalicAttributeId: variantBool(out, run->italic); break;
    case UIA_UnderlineStyleAttributeId: integer(out, run->underline ? 1 : 0); break;
    case UIA_StrikethroughStyleAttributeId: integer(out, run->strike ? 1 : 0); break;
    case UIA_ForegroundColorAttributeId: integer(out, run->foreground); break;
    case UIA_HorizontalTextAlignmentAttributeId: integer(out, run->alignment); break;
    case UIA_StyleIdAttributeId: integer(out, run->heading >= 1 && run->heading <= 6 ? 70000 + run->heading : 70012); break;
    default: return 0;
    }
    return 1;
}
static int sameTextAttribute(VARIANT *a, VARIANT *b) {
    if (a->vt != b->vt) return 0;
    switch (a->vt) {
    case VT_BOOL: return (a->boolVal != VARIANT_FALSE) == (b->boolVal != VARIANT_FALSE);
    case VT_I4: return a->lVal == b->lVal;
    case VT_R8: return a->dblVal == b->dblVal;
    default: return 0;
    }
}
static HRESULT reservedTextAttribute(VARIANT *out, int mixed) {
    IUnknown *value = NULL;
    HRESULT hr = mixed ? uiaMixedAttribute(&value) : uiaNotSupported(&value);
    if (SUCCEEDED(hr)) { out->vt = VT_UNKNOWN; out->punkVal = value; }
    return hr;
}
