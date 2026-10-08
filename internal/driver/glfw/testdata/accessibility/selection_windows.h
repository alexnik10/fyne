static void expectSelection(ISelectionProvider *provider, Element *expected) {
    SAFEARRAY *array = NULL;
    assert(ISelectionProvider_GetSelection(provider, &array) == S_OK && array);
    LONG upper = -1;
    assert(SafeArrayGetUBound(array, 1, &upper) == S_OK);
    assert(upper == (expected ? 0 : -1));
    if (expected) {
        LONG index = 0; IUnknown *item = NULL;
        assert(SafeArrayGetElement(array, &index, &item) == S_OK);
        assert(item == (IUnknown *)&expected->simple);
        IUnknown_Release(item);
    }
    SafeArrayDestroy(array);
}

static void testSelectionProviders(void) {
    HWND hwnd = newWindow(); assert(hwnd);
    WinAccessibility *c = WinAccessibilityCreate(hwnd, 66); assert(c);
    WinAccessibilityNode nodes[] = {
        {.id=1, .role=8, .name="Language", .flags=WinAccSelection|WinAccExpandable|WinAccValue|WinAccReadOnly|WinAccFocusable, .value="English"},
        {.id=2, .parent=1, .selection_owner=1, .role=10, .name="English", .flags=WinAccSelectable|WinAccSelected, .set_position=1, .set_size=2},
        {.id=3, .parent=1, .selection_owner=1, .role=10, .name="Russian", .flags=WinAccSelectable, .set_position=2, .set_size=2},
        {.id=4, .role=0, .name="Notifications", .flags=WinAccSelection|WinAccSelectionRequired},
        {.id=5, .parent=4, .selection_owner=4, .role=9, .name="Daily", .flags=WinAccSelectable|WinAccSelected|WinAccFocusable},
        {.id=6, .parent=4, .selection_owner=4, .role=9, .name="Weekly", .flags=WinAccSelectable|WinAccFocusable}
    };
    assert(WinAccessibilityUpdate(c, nodes, 6));
    Element *combo = retain(c, 1), *english = retain(c, 2), *russian = retain(c, 3);
    Element *group = retain(c, 4), *daily = retain(c, 5), *weekly = retain(c, 6);
    IUnknown *patternProvider = NULL;
    assert(pattern(&combo->simple, UIA_SelectionPatternId, &patternProvider) == S_OK && patternProvider == (IUnknown *)&combo->selection);
    IUnknown_Release(patternProvider);
    assert(query(daily, &IID_ISelectionItemProvider, (void **)&patternProvider) == S_OK);
    IUnknown_Release(patternProvider);
    assert(pattern(&group->simple, UIA_ExpandCollapsePatternId, &patternProvider) == S_OK && !patternProvider);
    BOOL value = TRUE;
    assert(ISelectionProvider_get_CanSelectMultiple(&combo->selection, &value) == S_OK && !value);
    assert(ISelectionProvider_get_IsSelectionRequired(&group->selection, &value) == S_OK && value);
    expectSelection(&combo->selection, english);
    expectSelection(&group->selection, daily);
    IRawElementProviderSimple *owner = NULL;
    assert(ISelectionItemProvider_get_SelectionContainer(&weekly->selectionItem, &owner) == S_OK && owner == &group->simple);
    IRawElementProviderSimple_Release(owner);
    VARIANT v;
    assert(property(&russian->simple, UIA_PositionInSetPropertyId, &v) == S_OK && v.vt == VT_I4 && v.lVal == 2);
    VariantClear(&v);
    assert(property(&russian->simple, UIA_SizeOfSetPropertyId, &v) == S_OK && v.vt == VT_I4 && v.lVal == 2);
    VariantClear(&v);
    assert(property(&daily->simple, UIA_ControlTypePropertyId, &v) == S_OK && v.lVal == UIA_RadioButtonControlTypeId);
    VariantClear(&v);
    assert(ISelectionItemProvider_RemoveFromSelection(&daily->selectionItem) == (HRESULT)UIA_E_INVALIDOPERATION);
    assert(ISelectionItemProvider_AddToSelection(&weekly->selectionItem) == (HRESULT)UIA_E_INVALIDOPERATION);
    assert(ISelectionItemProvider_Select(&weekly->selectionItem) == S_OK && lastAction == 6 && actionID == 6);
    assert(ISelectionItemProvider_AddToSelection(&daily->selectionItem) == S_OK && lastAction == 7);
    assert(ISelectionItemProvider_RemoveFromSelection(&english->selectionItem) == S_OK && lastAction == 8);
    assert(IExpandCollapseProvider_Expand(&combo->expand) == S_OK && lastAction == 9);
    assert(IExpandCollapseProvider_Collapse(&combo->expand) == S_OK && lastAction == 10);
    assert(IValueProvider_SetValue(&combo->value, L"arbitrary") == (HRESULT)UIA_E_INVALIDOPERATION);

    int selectedBefore = itemSelectionEvents, removedBefore = itemRemovalEvents, expandedBefore = expansionEvents;
    nodes[0].flags |= WinAccExpanded;
    nodes[4].flags &= ~WinAccSelected;
    nodes[5].flags |= WinAccSelected;
    assert(WinAccessibilityUpdate(c, nodes, 6));
    assert(itemSelectionEvents == selectedBefore + 1 && itemRemovalEvents == removedBefore + 1 && expansionEvents == expandedBefore + 1);
    expectSelection(&group->selection, weekly);
    enum ExpandCollapseState expanded;
    assert(IExpandCollapseProvider_get_ExpandCollapseState(&combo->expand, &expanded) == S_OK && expanded == ExpandCollapseState_Expanded);
    nodes[3].flags |= WinAccDisabled;
    assert(WinAccessibilityUpdate(c, nodes, 6));
    assert(ISelectionItemProvider_Select(&daily->selectionItem) == (HRESULT)UIA_E_ELEMENTNOTENABLED);
    nodes[0].flags |= WinAccDisabled;
    assert(WinAccessibilityUpdate(c, nodes, 6));
    assert(IExpandCollapseProvider_Expand(&combo->expand) == (HRESULT)UIA_E_ELEMENTNOTENABLED);

    // Retained option/owner interfaces must survive removal and closure safely.
    nodes[1].flags &= ~WinAccSelected;
    assert(WinAccessibilityUpdate(c, nodes, 3));
    expectSelection(&combo->selection, NULL);
    assert(ISelectionItemProvider_Select(&weekly->selectionItem) == UNAVAILABLE);
    WinAccessibilityCleanup(c);
    assert(IExpandCollapseProvider_Collapse(&combo->expand) == UNAVAILABLE);
    SAFEARRAY *array = NULL;
    assert(ISelectionProvider_GetSelection(&combo->selection, &array) == UNAVAILABLE && !array);
    release(combo); release(english); release(russian); release(group); release(daily); release(weekly);
    DestroyWindow(hwnd);
}

static void testTreeProviders(void) {
    HWND hwnd = newWindow(); assert(hwnd);
    WinAccessibility *c = WinAccessibilityCreate(hwnd, 77); assert(c);
    WinAccessibilityNode nodes[] = {
        {.id=1, .role=11, .name="Files", .flags=WinAccSelection|WinAccFocusable},
        {.id=2, .parent=1, .selection_owner=1, .role=12, .name="Folder", .flags=WinAccSelectable|WinAccExpandable|WinAccExpanded|WinAccScrollItem, .level=1, .set_position=1, .set_size=1},
        {.id=3, .parent=2, .selection_owner=1, .role=12, .name="File", .flags=WinAccSelectable|WinAccLeaf|WinAccScrollItem, .level=2, .set_position=1, .set_size=1}
    };
    assert(WinAccessibilityUpdate(c, nodes, 3));
    Element *tree = retain(c, 1), *branch = retain(c, 2), *leaf = retain(c, 3);
    VARIANT v;
    assert(property(&tree->simple, UIA_ControlTypePropertyId, &v) == S_OK && v.lVal == UIA_TreeControlTypeId);
    VariantClear(&v);
    assert(property(&leaf->simple, UIA_ControlTypePropertyId, &v) == S_OK && v.lVal == UIA_TreeItemControlTypeId);
    VariantClear(&v);
    assert(property(&leaf->simple, UIA_LevelPropertyId, &v) == S_OK && v.vt == VT_I4 && v.lVal == 2);
    VariantClear(&v);
    assert(property(&leaf->simple, UIA_IsOffscreenPropertyId, &v) == S_OK && v.boolVal == VARIANT_TRUE);
    VariantClear(&v);
    IUnknown *provider = NULL;
    assert(pattern(&leaf->simple, UIA_ScrollItemPatternId, &provider) == S_OK && provider == (IUnknown *)&leaf->scrollItem);
    IUnknown_Release(provider);
    assert(query(leaf, &IID_IScrollItemProvider, (void **)&provider) == S_OK);
    IUnknown_Release(provider);
    assert(pattern(&leaf->simple, UIA_ExpandCollapsePatternId, &provider) == S_OK && provider == (IUnknown *)&leaf->expand);
    IUnknown_Release(provider);
    enum ExpandCollapseState state;
    assert(IExpandCollapseProvider_get_ExpandCollapseState(&leaf->expand, &state) == S_OK && state == ExpandCollapseState_LeafNode);
    assert(IExpandCollapseProvider_Expand(&leaf->expand) == (HRESULT)UIA_E_INVALIDOPERATION);
    assert(IExpandCollapseProvider_Collapse(&leaf->expand) == (HRESULT)UIA_E_INVALIDOPERATION);
    assert(IScrollItemProvider_ScrollIntoView(&leaf->scrollItem) == S_OK && lastAction == 11 && actionID == 3);
    IRawElementProviderFragment *parent = NULL;
    assert(IRawElementProviderFragment_Navigate(&leaf->fragment, NavigateDirection_Parent, &parent) == S_OK && parent == &branch->fragment);
    IRawElementProviderFragment_Release(parent);
    // A leaf becoming a branch must announce its state and keep its provider.
    int before = expansionEvents;
    nodes[2].flags = WinAccSelectable|WinAccExpandable|WinAccScrollItem;
    assert(WinAccessibilityUpdate(c, nodes, 3));
    assert(expansionEvents == before + 1 && elementFor(c, 3) == leaf);
    assert(IExpandCollapseProvider_get_ExpandCollapseState(&leaf->expand, &state) == S_OK && state == ExpandCollapseState_Collapsed);
    assert(IExpandCollapseProvider_Expand(&leaf->expand) == S_OK);
    assert(WinAccessibilityUpdate(c, nodes, 2));
    assert(IScrollItemProvider_ScrollIntoView(&leaf->scrollItem) == UNAVAILABLE);
    WinAccessibilityCleanup(c);
    release(tree); release(branch); release(leaf);
    DestroyWindow(hwnd);
}
