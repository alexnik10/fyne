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
    WinAccText = 4096
};
typedef struct {
    double x, y, height;
    int line;
} WinAccessibilityTextPosition;
typedef struct {
    uint32_t id, parent;
    int role, flags;
    const char *name, *description, *value;
    double x, y, width, height;
    double number, minimum, maximum, step;
    const char *text;
    int caret, selection_start, selection_end;
    uint64_t text_revision;
    const WinAccessibilityTextPosition *positions;
    int position_count;
    double viewport_x, viewport_y, viewport_width, viewport_height;
} WinAccessibilityNode;

WinAccessibility *WinAccessibilityCreate(void *hwnd, uintptr_t handle);
// Copies a complete snapshot; on allocation failure the previous tree survives.
int WinAccessibilityUpdate(WinAccessibility *, const WinAccessibilityNode *, int count);
void WinAccessibilityFocus(WinAccessibility *, uint32_t id);
void WinAccessibilityCleanup(WinAccessibility *);
#endif
