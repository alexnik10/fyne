// A separate process is required: direct vtable tests cannot detect UIA's
// round trips to the HWND thread. Match Fyne's 60 Hz native message polling.
#include "../../accessibility_windows.c"
#include <stdio.h>

void goFyneAccessibilityAction(uintptr_t h, uint32_t id, int a, char *v, double n) {}
int goFyneAccessibilityPerform(uintptr_t h, uint32_t id, int a, char *v, double n) { return 1; }
int goFyneAccessibilityTextAction(uintptr_t h, uint32_t id, int s, int e, int scroll, int top) { return 1; }

static int cachedMetadata;
static LONG messages[0x10000];
static HRESULT STDMETHODCALLTYPE probeProperty(IRawElementProviderSimple *p, PROPERTYID id, VARIANT *out) {
    Element *e = OWNER(p, simple);
    if (cachedMetadata) {
        VariantInit(out);
        switch (id) {
        case UIA_NativeWindowHandlePropertyId: integer(out, e->id ? 0 : (LONG)(LONG_PTR)e->context->hwnd); return S_OK;
        case UIA_ProcessIdPropertyId: integer(out, GetCurrentProcessId()); return S_OK;
        case UIA_FrameworkIdPropertyId: variantString(out, L"Fyne"); return S_OK;
        case UIA_ClassNamePropertyId: variantString(out, e->id ? L"" : L"FyneUIAQueryTest"); return S_OK;
        case UIA_NamePropertyId: if (!e->id) { variantString(out, L"Fyne UIA query test"); return S_OK; } break;
        case UIA_ControlTypePropertyId: if (!e->id) { integer(out, UIA_WindowControlTypeId); return S_OK; } break;
        }
    }
    return property(p, id, out);
}
static LRESULT CALLBACK fixtureProc(HWND hwnd, UINT message, WPARAM wp, LPARAM lp) {
    if (message < 0x10000) InterlockedIncrement(&messages[message]);
    if (message == WM_CLOSE) { PostQuitMessage(0); return 0; }
    return DefWindowProcW(hwnd, message, wp, lp);
}
int main(int argc, char **argv) {
    cachedMetadata = argc > 1 && !strcmp(argv[1], "cached");
    WNDCLASSW cls = {.lpfnWndProc=fixtureProc, .hInstance=GetModuleHandleW(NULL), .lpszClassName=L"FyneUIAQueryTest"};
    if (!RegisterClassW(&cls)) return 1;
    HWND hwnd = CreateWindowW(cls.lpszClassName, L"Fyne UIA query test", WS_OVERLAPPEDWINDOW,
        0, 0, 400, 300, NULL, NULL, cls.hInstance, NULL);
    if (!hwnd) return 2;
    WinAccessibility *c = WinAccessibilityCreate(hwnd, 1);
    if (!c) return 3;
    simpleVtbl.GetPropertyValue = probeProperty;
    WinAccessibilityNode nodes[] = {
        {.id=1, .role=0, .name="Choices"},
        {.id=2, .parent=1, .role=8, .name="Language", .flags=WinAccFocusable},
        {.id=3, .parent=2, .role=10, .name="English"},
        {.id=4, .parent=1, .role=0, .name="Notifications"},
        {.id=5, .parent=4, .role=9, .name="Daily", .flags=WinAccFocusable},
        {.id=6, .parent=1, .role=4, .name="Disable choices", .flags=WinAccFocusable|WinAccToggle}
    };
    if (!WinAccessibilityUpdate(c, nodes, 6)) return 4;
    ShowWindow(hwnd, SW_SHOWNOACTIVATE);
    MSG msg;
    int quit = 0;
    ULONGLONG started = GetTickCount64();
    while (!quit && GetTickCount64() - started < 30000) {
        while (PeekMessageW(&msg, NULL, 0, 0, PM_REMOVE)) {
            if (msg.message == WM_QUIT) { quit = 1; break; }
            TranslateMessage(&msg); DispatchMessageW(&msg);
        }
        Sleep(16);
    }
    WinAccessibilityCleanup(c); DestroyWindow(hwnd);
    for (int i=0; i<0x10000; ++i) if (messages[i]) printf("message 0x%04x: %ld\n", i, messages[i]);
    return quit ? 0 : 5;
}
