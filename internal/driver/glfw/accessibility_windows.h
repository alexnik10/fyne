//go:build accessibility && windows

#ifndef FYNE_ACCESSIBILITY_WINDOWS_H
#define FYNE_ACCESSIBILITY_WINDOWS_H
#include <stdint.h>

typedef struct WinAccessibility WinAccessibility;
// Values are private to the Windows adapter; no Windows types enter public API.
enum {
    WinAccDisabled = 1, WinAccFocusable = 2, WinAccFocused = 4,
    WinAccRequired = 8, WinAccInvalid = 16, WinAccInvoke = 32,
    WinAccToggle = 64, WinAccValue = 128, WinAccRange = 256,
    WinAccChecked = 512, WinAccReadOnly = 1024, WinAccProtected = 2048,
    WinAccText = 4096, WinAccSelection = 8192, WinAccMultiple = 16384,
    WinAccSelectionRequired = 32768, WinAccSelectable = 65536,
    WinAccSelected = 131072, WinAccExpandable = 262144, WinAccExpanded = 524288
};
typedef struct {
    double x, y, height;
    int line;
} WinAccessibilityTextPosition;
typedef struct {
    uint32_t id, parent, selection_owner;
    int set_position, set_size;
    int role, flags;
    const char *name, *description, *value;
    double x, y, width, height;
    double number, minimum, maximum, step;
    const char *text;
    int caret, selection_start, selection_end;
    uint64_t text_revision;
    const int *word_boundaries;
    int word_boundary_count;
    const WinAccessibilityTextPosition *positions;
    int position_count;
    double viewport_x, viewport_y, viewport_width, viewport_height;
} WinAccessibilityNode;

WinAccessibility *WinAccessibilityCreate(void *hwnd, uintptr_t handle);
// Copies a complete snapshot; on allocation failure the previous tree survives.
int WinAccessibilityUpdate(WinAccessibility *, const WinAccessibilityNode *, int count);
// Optional diagnostic measurements contain no names, values or input text.
typedef struct {
    double snapshot_ms, events_ms, slowest_ms;
    int event_count, slowest_kind, slowest_id;
    uint32_t slowest_node;
} WinAccessibilityStats;
int WinAccessibilityUpdateWithStats(WinAccessibility *, const WinAccessibilityNode *, int count, WinAccessibilityStats *);
void WinAccessibilityFocus(WinAccessibility *, uint32_t id);
void WinAccessibilityCleanup(WinAccessibility *);
// Main-thread wait services synchronous HWND queries; posted input stays with GLFW.
int WinAccessibilityWaitForMessage(uint32_t timeout);
void WinAccessibilityWake(void);
#endif
