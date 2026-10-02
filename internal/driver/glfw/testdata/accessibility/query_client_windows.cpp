// Exercise the same HWND normalization and cache request used by NVDA, through
// the real UIAutomationCore and a separate provider process (no mocked events).
#include <windows.h>
#include <UIAutomation.h>
#include <stdio.h>
#include <stdlib.h>
#include <wrl/client.h>
#pragma comment(lib, "user32.lib")
using Microsoft::WRL::ComPtr;

static void check(HRESULT hr) {
    if (FAILED(hr)) { fprintf(stderr, "UIA query failed: 0x%08lx\n", hr); exit(1); }
}
static double run(const wchar_t *mode) {
    wchar_t command[256];
    swprintf_s(command, L"query-host.exe %s", mode);
    STARTUPINFOW startup = {sizeof(startup)};
    PROCESS_INFORMATION process = {};
    if (!CreateProcessW(nullptr, command, nullptr, nullptr, FALSE, 0, nullptr, nullptr, &startup, &process)) exit(2);
    HWND hwnd = nullptr;
    for (int i=0; i<500 && !hwnd; ++i) { hwnd = FindWindowW(L"FyneUIAQueryTest", nullptr); Sleep(10); }
    if (!hwnd) exit(3);
    ComPtr<IUIAutomation> uia;
    check(CoCreateInstance(CLSID_CUIAutomation, nullptr, CLSCTX_INPROC_SERVER, IID_PPV_ARGS(&uia)));
    ComPtr<IUIAutomationElement> root;
    check(uia->ElementFromHandle(hwnd, &root));
    ComPtr<IUIAutomationCondition> all, noWindow, hasWindow;
    check(uia->CreateTrueCondition(&all));
    VARIANT zero; VariantInit(&zero); zero.vt = VT_I4; zero.lVal = 0;
    check(uia->CreatePropertyCondition(UIA_NativeWindowHandlePropertyId, zero, &noWindow));
    check(uia->CreateNotCondition(noWindow.Get(), &hasWindow));
    ComPtr<IUIAutomationTreeWalker> walker;
    check(uia->CreateTreeWalker(hasWindow.Get(), &walker));
    ComPtr<IUIAutomationCacheRequest> cache;
    check(uia->CreateCacheRequest(&cache));
    PROPERTYID properties[] = {UIA_NativeWindowHandlePropertyId, UIA_ProcessIdPropertyId,
        UIA_FrameworkIdPropertyId, UIA_AutomationIdPropertyId, UIA_ClassNamePropertyId,
        UIA_ControlTypePropertyId, UIA_ProviderDescriptionPropertyId, UIA_IsTextPatternAvailablePropertyId,
        UIA_IsContentElementPropertyId, UIA_IsControlElementPropertyId, UIA_NamePropertyId,
        UIA_LocalizedControlTypePropertyId, UIA_HasKeyboardFocusPropertyId};
    for (auto id : properties) check(cache->AddProperty(id));
    ComPtr<IUIAutomationElementArray> children;
    check(root->FindAll(TreeScope_Descendants, all.Get(), &children));
    int count; check(children->get_Length(&count));
    ComPtr<IUIAutomationElement> target;
    for (int i=0; i<count; ++i) {
        ComPtr<IUIAutomationElement> child; check(children->GetElement(i, &child));
        BSTR name = nullptr; check(child->get_CurrentAutomationId(&name));
        bool match = name && !wcscmp(name, L"fyne_5"); SysFreeString(name);
        if (match) { target = child; break; }
    }
    if (!target) exit(4);
    LARGE_INTEGER frequency, start, end;
    QueryPerformanceFrequency(&frequency); QueryPerformanceCounter(&start);
    for (int i=0; i<20; ++i) {
        ComPtr<IUIAutomationElement> window;
        check(walker->NormalizeElementBuildCache(target.Get(), cache.Get(), &window));
        UIA_HWND found; check(window->get_CachedNativeWindowHandle(&found));
        if ((HWND)found != hwnd) exit(5);
    }
    QueryPerformanceCounter(&end);
    double elapsed = 1000.0 * (end.QuadPart - start.QuadPart) / frequency.QuadPart;
    printf("%ls: 20 HWND normalizations %.2f ms (%.2f ms/query)\n", mode, elapsed, elapsed/20);
    fflush(stdout);
    PostMessageW(hwnd, WM_CLOSE, 0, 0);
    if (WaitForSingleObject(process.hProcess, 10000) != WAIT_OBJECT_0) exit(6);
    DWORD code; GetExitCodeProcess(process.hProcess, &code);
    CloseHandle(process.hThread); CloseHandle(process.hProcess);
    if (code) exit(7);
    return elapsed;
}
int main() {
    check(CoInitializeEx(nullptr, COINIT_MULTITHREADED));
    double baseline = run(L"baseline");
    run(L"metadata"); // Show the remaining cost if native queries still wait for a frame.
    double cached = run(L"cached");
    printf("HWND normalization ratio cached/baseline: %.3f\n", cached/baseline);
    // Gate a large relative regression when the old host fallback reproduces
    // message-pump stalls. Avoid a tight machine-dependent latency threshold.
    if (baseline >= 400 && cached >= baseline * 0.2) {
        fprintf(stderr, "Cached HWND queries still depend on the native polling interval\n");
        return 8;
    }
    CoUninitialize();
    return 0;
}
