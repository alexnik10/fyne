static void testWindowMetadata(void) {
    HWND hwnd = newWindow(); assert(hwnd);
    WinAccessibility *c = WinAccessibilityCreate(hwnd, 101); assert(c);
    WinAccessibilityNode node = {.id=1, .role=4, .name="Choice", .flags=WinAccToggle};
    assert(WinAccessibilityUpdate(c, &node, 1));
    Element *root = retain(c, 0), *child = retain(c, 1);
    VARIANT v;
    assert(property(&root->simple, UIA_NativeWindowHandlePropertyId, &v) == S_OK);
    assert(v.vt == VT_I4 && v.lVal == (LONG)(LONG_PTR)hwnd); VariantClear(&v);
    assert(property(&child->simple, UIA_NativeWindowHandlePropertyId, &v) == S_OK);
    assert(v.vt == VT_I4 && v.lVal == 0); VariantClear(&v); // child does not own the HWND
    assert(property(&child->simple, UIA_ProcessIdPropertyId, &v) == S_OK);
    assert(v.vt == VT_I4 && (DWORD)v.lVal == GetCurrentProcessId()); VariantClear(&v);
    assert(property(&child->simple, UIA_FrameworkIdPropertyId, &v) == S_OK);
    assert(v.vt == VT_BSTR && !wcscmp(v.bstrVal, L"Fyne")); VariantClear(&v);
    assert(property(&root->simple, UIA_ClassNamePropertyId, &v) == S_OK);
    assert(v.vt == VT_BSTR && !wcsicmp(v.bstrVal, L"STATIC")); VariantClear(&v);
    assert(property(&root->simple, UIA_ControlTypePropertyId, &v) == S_OK);
    assert(v.vt == VT_I4 && v.lVal == UIA_WindowControlTypeId); VariantClear(&v);
    const WCHAR *titles[] = {L"Provider test", L"Updated \u041e\u043a\u043d\u043e", L""};
    for (unsigned int i=0; i<sizeof(titles)/sizeof(titles[0]); ++i) {
        assert(SetWindowTextW(hwnd, titles[i]));
        assert(property(&root->simple, UIA_NamePropertyId, &v) == S_OK);
        assert(v.vt == VT_BSTR && !wcscmp(v.bstrVal, titles[i])); VariantClear(&v);
        assert(property(&child->simple, UIA_NamePropertyId, &v) == S_OK);
        assert(v.vt == VT_BSTR && !wcscmp(v.bstrVal, L"Choice")); VariantClear(&v);
    }
    WinAccessibilityCleanup(c);
    assert(property(&root->simple, UIA_NamePropertyId, &v) == UNAVAILABLE);
    assert(property(&child->simple, UIA_NativeWindowHandlePropertyId, &v) == UNAVAILABLE);
    release(root); release(child); DestroyWindow(hwnd);
}

// Replay NVDA's caret-range expansion on a final blank line, including updates
// from an existing last line and geometry-free custom AccessibleText controls.
static void testFinalEmptyLine(void) {
    HWND hwnd = newWindow(); assert(hwnd);
    WinAccessibility *c = WinAccessibilityCreate(hwnd, 77); assert(c);
    WinAccessibilityNode node = {.id=1, .role=5, .flags=WinAccText|WinAccFocusable};
    assert(WinAccessibilityUpdate(c, &node, 1));
    Element *entry = retain(c, 1);
    const char *texts[] = {"first\nlast", "first\nlast\n", "first\nlast\n\n", "", "\n"};
    for (int geometry=0; geometry<2; ++geometry) for (unsigned int i=0; i<sizeof(texts)/sizeof(texts[0]); ++i) {
        node.text = texts[i];
        int length = (int)strlen(node.text), line = 0, column = 0;
        WinAccessibilityTextPosition positions[32];
        for (int j=0; j<=length; ++j) {
            positions[j] = (WinAccessibilityTextPosition){column*10, line*20, 20, line};
            if (node.text[j] == '\n') { ++line; column = 0; } else ++column;
        }
        node.positions = geometry ? positions : NULL;
        node.position_count = geometry ? length+1 : 0;
        node.caret = node.selection_start = node.selection_end = length;
        ++node.text_revision;
        assert(WinAccessibilityUpdate(c, &node, 1));
        enum TextUnit units[] = {TextUnit_Line, TextUnit_Paragraph};
        for (unsigned int j=0; j<sizeof(units)/sizeof(units[0]); ++j) {
            SAFEARRAY *selection = NULL; ITextRangeProvider *range = NULL; LONG index = 0;
            assert(ITextProvider2_GetSelection(&entry->text, &selection) == S_OK);
            assert(SafeArrayGetElement(selection, &index, &range) == S_OK); SafeArrayDestroy(selection);
            assert(ITextRangeProvider_ExpandToEnclosingUnit(range, units[j]) == S_OK);
            expectText(range, i == 0 ? L"last" : L"");
            if (i == 1) {
                int moved = 0;
                assert(ITextRangeProvider_Move(range, units[j], -1, &moved) == S_OK && moved == -1);
                assert(ITextRangeProvider_ExpandToEnclosingUnit(range, units[j]) == S_OK);
                expectText(range, L"last\n");
                assert(ITextRangeProvider_Move(range, units[j], 1, &moved) == S_OK && moved == 1);
                expectText(range, L"");
                assert(TEXT_RANGE(range)->start == length && TEXT_RANGE(range)->end == length);
                assert(ITextRangeProvider_Move(range, units[j], 1, &moved) == S_OK && moved == 0);
                // Document expansion still includes the entire text.
                assert(ITextRangeProvider_ExpandToEnclosingUnit(range, TextUnit_Document) == S_OK);
                expectText(range, L"first\nlast\n");
            }
            ITextRangeProvider_Release(range);
        }
    }
    WinAccessibilityCleanup(c); release(entry); DestroyWindow(hwnd);
}

static int feedbackCount, backgroundDisabled;
static void observeFeedback(IRawElementProviderSimple *provider, PROPERTYID id) {
    Element *e = OWNER(provider, simple);
    if (feedbackCount++ == 0) {
        assert(e->id == 3 && id == UIA_ToggleToggleStatePropertyId);
        // Even the first callback sees the complete committed snapshot.
        Element *background = retain(e->context, 1);
        VARIANT value;
        assert(property(&background->simple, UIA_IsEnabledPropertyId, &value) == S_OK);
        assert(value.vt == VT_BOOL && value.boolVal == (backgroundDisabled ? VARIANT_FALSE : VARIANT_TRUE));
        VariantClear(&value); release(background);
    } else {
        assert((e->id == 1 || e->id == 2) && id == UIA_IsEnabledPropertyId);
    }
}

static void testFocusedFeedbackOrder(void) {
    HWND hwnd = newWindow(); assert(hwnd);
    WinAccessibility *c = WinAccessibilityCreate(hwnd, 88); assert(c);
    WinAccessibilityNode nodes[] = {
        {.id=1, .role=8, .name="Language"},
        {.id=2, .role=9, .name="Daily"},
        {.id=3, .role=4, .name="Disable choices", .flags=WinAccToggle|WinAccFocusable|WinAccFocused}
    };
    assert(WinAccessibilityUpdate(c, nodes, 3));
    observeProperty = observeFeedback;
    for (int disabled=1; disabled>=0; --disabled) {
        backgroundDisabled = disabled; feedbackCount = 0;
        nodes[0].flags = nodes[1].flags = disabled ? WinAccDisabled : 0;
        nodes[2].flags = WinAccToggle|WinAccFocusable|WinAccFocused|(disabled ? WinAccChecked : 0);
        assert(WinAccessibilityUpdate(c, nodes, 3));
        assert(feedbackCount == 3); // Each changed property is emitted once.
        assert(WinAccessibilityUpdate(c, nodes, 3));
        assert(feedbackCount == 3); // An unchanged snapshot emits no duplicates.
    }
    observeProperty = NULL;
    WinAccessibilityCleanup(c); DestroyWindow(hwnd);
}

static void slowDiagnosticClient(void) { Sleep(30); }

static void testDiagnosticTimings(void) {
    HWND hwnd = newWindow(); assert(hwnd);
    WinAccessibility *c = WinAccessibilityCreate(hwnd, 99); assert(c);
    WinAccessibilityNode node = {.id=1, .role=4, .flags=WinAccToggle};
    assert(WinAccessibilityUpdate(c, &node, 1));
    node.flags |= WinAccChecked;
    WinAccessibilityStats stats;
    duringProperty = slowDiagnosticClient;
    assert(WinAccessibilityUpdateWithStats(c, &node, 1, &stats));
    assert(stats.event_count == 1 && stats.events_ms >= 15 && stats.slowest_ms >= 15);
    assert(stats.slowest_node == 1 && stats.slowest_kind == 1 && stats.slowest_id == UIA_ToggleToggleStatePropertyId);
    assert(stats.snapshot_ms >= 0 && stats.events_ms >= stats.slowest_ms);
    // Each update has its own measurements; the unchanged tree raises no event.
    assert(WinAccessibilityUpdateWithStats(c, &node, 1, &stats));
    assert(stats.event_count == 0 && stats.events_ms == 0 && stats.slowest_id == 0);
    Element *check = retain(c, 1);
    VARIANT value;
    assert(property(&check->simple, UIA_ToggleToggleStatePropertyId, &value) == S_OK && value.lVal == ToggleState_On);
    VariantClear(&value);
    WinAccessibilityCleanup(c); release(check); DestroyWindow(hwnd);
}
