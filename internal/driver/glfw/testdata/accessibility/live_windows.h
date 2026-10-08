// Included by the native provider test after its provider helpers.
static const WCHAR *expectedLiveName;
static void observeLiveSnapshot(IRawElementProviderSimple *p) {
    VARIANT v;
    assert(IRawElementProviderSimple_GetPropertyValue(p, UIA_NamePropertyId, &v) == S_OK);
    assert(v.vt == VT_BSTR && !wcscmp(v.bstrVal, expectedLiveName)); VariantClear(&v);
    assert(IRawElementProviderSimple_GetPropertyValue(p, UIA_LiveSettingPropertyId, &v) == S_OK);
    assert(v.vt == VT_I4 && v.lVal == 1); VariantClear(&v);
}

static void testLiveRegions(void) {
    HWND hwnd = newWindow(); assert(hwnd);
    WinAccessibility *c = WinAccessibilityCreate(hwnd, 78); assert(c);
    WinAccessibilityNode nodes[] = {
        {.id=1, .role=1, .name="Save", .flags=WinAccFocusable|WinAccFocused|WinAccInvoke},
        {.id=2, .role=2, .name="Ready", .live_setting=1},
        {.id=3, .role=2, .name="Ordinary label"}
    };
    int events = liveEvents;
    assert(WinAccessibilityUpdate(c, nodes, 3));
    int focus = focusEvents;
    assert(liveEvents == events); // Initial exposure is silent.
    Element *status = retain(c, 2); VARIANT v;
    assert(property(&status->simple, UIA_LiveSettingPropertyId, &v) == S_OK);
    assert(v.vt == VT_I4 && v.lVal == 1); VariantClear(&v);
    observeLive = observeLiveSnapshot; expectedLiveName = L"Saved \u0414\u043e\u043a\u0443\u043c\u0435\u043d\u0442";
    nodes[1].name = "Saved \xd0\x94\xd0\xbe\xd0\xba\xd1\x83\xd0\xbc\xd0\xb5\xd0\xbd\xd1\x82";
    nodes[1].live_revision++;
    assert(WinAccessibilityUpdate(c, nodes, 3)); assert(liveEvents == ++events);
    assert(c->focus == 1 && !(find(c, 2)->data.flags & WinAccFocusable));
    assert(WinAccessibilityUpdate(c, nodes, 3)); assert(liveEvents == events);
    nodes[1].live_revision++;
    assert(WinAccessibilityUpdate(c, nodes, 3)); assert(liveEvents == ++events);
    nodes[2].name = "Changed ordinary label";
    assert(WinAccessibilityUpdate(c, nodes, 3)); assert(liveEvents == events);
    observeLive = NULL;
    nodes[1].name = ""; nodes[1].live_revision++;
    assert(WinAccessibilityUpdate(c, nodes, 3)); assert(liveEvents == events);
    nodes[1].live_setting = 0; nodes[1].name = "Disabled live setting";
    assert(WinAccessibilityUpdate(c, nodes, 3)); assert(liveEvents == events);
    nodes[1].live_setting = 2;
    assert(WinAccessibilityUpdate(c, nodes, 3)); assert(liveEvents == events);
    assert(property(&status->simple, UIA_LiveSettingPropertyId, &v) == S_OK && v.lVal == 2); VariantClear(&v);
    nodes[1].live_revision++;
    assert(WinAccessibilityUpdate(c, nodes, 3)); assert(liveEvents == ++events);
    nodes[1].flags = WinAccProtected; nodes[1].live_revision++;
    assert(WinAccessibilityUpdate(c, nodes, 3)); assert(liveEvents == events);
    assert(property(&status->simple, UIA_LiveSettingPropertyId, &v) == S_OK && v.lVal == 0); VariantClear(&v);
    nodes[1].flags = 0;
    assert(WinAccessibilityUpdate(c, nodes, 1)); // Hidden/removed status.
    nodes[1].live_revision++;
    assert(WinAccessibilityUpdate(c, nodes, 3)); assert(liveEvents == events); // Reappearance is silent.
    assert(focusEvents == focus); // No synthetic focus event for announcements.
    release(status); WinAccessibilityCleanup(c); DestroyWindow(hwnd);
}
