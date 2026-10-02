// Included after Element is defined. A NULL stats pointer leaves timing off.
static double diagnosticNow(WinAccessibilityStats *stats) {
    if (!stats) return 0;
    LARGE_INTEGER counter, frequency;
    if (!QueryPerformanceCounter(&counter) || !QueryPerformanceFrequency(&frequency)) return 0;
    return 1000.0 * (double)counter.QuadPart / (double)frequency.QuadPart;
}

static void diagnosticEvent(WinAccessibilityStats *stats, double start, Element *element, int kind, int id) {
    if (!stats) return;
    double elapsed = diagnosticNow(stats) - start;
    ++stats->event_count;
    stats->events_ms += elapsed;
    if (stats->event_count == 1 || elapsed > stats->slowest_ms) {
        stats->slowest_ms = elapsed; stats->slowest_node = element->id;
        stats->slowest_kind = kind; stats->slowest_id = id;
    }
}

static void diagnosticProperty(WinAccessibilityStats *stats, Element *element, PROPERTYID id, VARIANT before, VARIANT after) {
    double start = diagnosticNow(stats);
    UiaRaiseAutomationPropertyChangedEvent(&element->simple, id, before, after);
    diagnosticEvent(stats, start, element, 1, id);
}

static void diagnosticAutomation(WinAccessibilityStats *stats, Element *element, EVENTID id) {
    double start = diagnosticNow(stats);
    UiaRaiseAutomationEvent(&element->simple, id);
    diagnosticEvent(stats, start, element, 2, id);
}

static void diagnosticStructure(WinAccessibilityStats *stats, Element *element) {
    double start = diagnosticNow(stats);
    UiaRaiseStructureChangedEvent(&element->simple, StructureChangeType_ChildrenInvalidated, NULL, 0);
    diagnosticEvent(stats, start, element, 3, StructureChangeType_ChildrenInvalidated);
}
