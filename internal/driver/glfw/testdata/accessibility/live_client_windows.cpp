// Exercise live-region properties and events through the native UIA client.
// The legacy .NET UIAutomationClient cannot decode the LiveSetting property.
#include <windows.h>
#include <UIAutomation.h>
#include <atomic>
#include <stdio.h>
#include <stdlib.h>
#include <string>
#include <wrl/client.h>
using Microsoft::WRL::ComPtr;
static constexpr LONG politeLiveSetting = 1; // UIA LiveSetting's documented polite value.

static void check(HRESULT hr) {
    if (FAILED(hr)) { fprintf(stderr, "UIA live-region call failed: 0x%08lx\n", hr); exit(1); }
}
static void expect(bool condition, const char *message) {
    if (!condition) { fprintf(stderr, "%s\n", message); exit(2); }
}
static ComPtr<IUIAutomationElement> named(IUIAutomation *uia, IUIAutomationElement *root, const wchar_t *name) {
    VARIANT value; VariantInit(&value); value.vt = VT_BSTR; value.bstrVal = SysAllocString(name);
    ComPtr<IUIAutomationCondition> condition;
    check(uia->CreatePropertyCondition(UIA_NamePropertyId, value, &condition));
    VariantClear(&value);
    ComPtr<IUIAutomationElement> element;
    check(root->FindFirst(TreeScope_Descendants, condition.Get(), &element));
    expect(element.Get() != nullptr, "Named live-status control was not found");
    return element;
}

class LiveEvents final : public IUIAutomationEventHandler {
    std::atomic<ULONG> refs{1};
public:
    const std::wstring expected;
    explicit LiveEvents(const wchar_t *name = L"Native saved") : expected(name) {}
    std::atomic<int> count{0};
    std::atomic<bool> bad{false};
    HRESULT STDMETHODCALLTYPE QueryInterface(REFIID iid, void **out) override {
        if (!out) return E_POINTER;
        *out = nullptr;
        if (iid == __uuidof(IUnknown) || iid == __uuidof(IUIAutomationEventHandler)) {
            *out = static_cast<IUIAutomationEventHandler *>(this); AddRef(); return S_OK;
        }
        return E_NOINTERFACE;
    }
    ULONG STDMETHODCALLTYPE AddRef() override { return ++refs; }
    ULONG STDMETHODCALLTYPE Release() override {
        ULONG remaining = --refs; if (!remaining) delete this; return remaining;
    }
    HRESULT STDMETHODCALLTYPE HandleAutomationEvent(IUIAutomationElement *sender, EVENTID id) override {
        BSTR name = nullptr;
        HRESULT hr = sender->get_CurrentName(&name);
        if (id != UIA_LiveRegionChangedEventId || FAILED(hr) || !name || wcscmp(name, expected.c_str())) bad = true;
        SysFreeString(name);
        ++count;
        return S_OK;
    }
};

int main(int argc, char **argv) {
    expect(argc == 2, "Expected the provider window handle");
    check(CoInitializeEx(nullptr, COINIT_MULTITHREADED));
    {
        ComPtr<IUIAutomation2> uia;
        check(CoCreateInstance(CLSID_CUIAutomation8, nullptr, CLSCTX_INPROC_SERVER, IID_PPV_ARGS(&uia)));
        // Exercise provider focus preservation without UIA focusing each action's
        // target automatically (the legacy managed client keeps that default).
        check(uia->put_AutoSetFocus(FALSE));
        ComPtr<IUIAutomationElement> root;
        check(uia->ElementFromHandle(reinterpret_cast<HWND>(strtoull(argv[1], nullptr, 10)), &root));
        auto status = named(uia.Get(), root.Get(), L"Native ready");
        auto save = named(uia.Get(), root.Get(), L"Native save");
        VARIANT setting;
        check(status->GetCurrentPropertyValue(UIA_LiveSettingPropertyId, &setting));
        expect(setting.vt == VT_I4 && setting.lVal == politeLiveSetting, "Status is not a polite live region");
        VariantClear(&setting);
        check(save->SetFocus());
        ComPtr<IUIAutomationInvokePattern> invoke;
        check(save->GetCurrentPatternAs(UIA_InvokePatternId, IID_PPV_ARGS(&invoke)));
        ComPtr<LiveEvents> events; events.Attach(new LiveEvents());
        check(uia->AddAutomationEventHandler(UIA_LiveRegionChangedEventId, status.Get(), TreeScope_Element, nullptr, events.Get()));
        for (int action = 1; action <= 2; ++action) {
            check(invoke->Invoke());
            ULONGLONG deadline = GetTickCount64() + 8000;
            while (events->count < action && GetTickCount64() < deadline) Sleep(10);
            expect(events->count == action, "Save did not deliver exactly one live-region event per action");
            expect(!events->bad, "Live event exposed stale or unexpected text");
            BOOL focused = FALSE; check(save->get_CurrentHasKeyboardFocus(&focused));
            expect(focused != FALSE, "Status announcement moved keyboard focus");
        }
        check(uia->RemoveAutomationEventHandler(UIA_LiveRegionChangedEventId, status.Get(), events.Get()));
        puts("Two identical live status results reached the native UIA client without moving focus");

        auto gridStatus = named(uia.Get(), root.Get(), L"Record 499 enabled: false");
        check(gridStatus->GetCurrentPropertyValue(UIA_LiveSettingPropertyId, &setting));
        expect(setting.vt == VT_I4 && setting.lVal == politeLiveSetting, "Grid status is not a polite live region");
        VariantClear(&setting);
        auto checkbox = named(uia.Get(), root.Get(), L"Enable Record 499");
        check(checkbox->SetFocus());
        auto grid = named(uia.Get(), root.Get(), L"Keyed wrapping grid");
        auto swap = named(uia.Get(), root.Get(), L"Swap first and last");
        ComPtr<IUIAutomationItemContainerPattern> items;
        check(grid->GetCurrentPatternAs(UIA_ItemContainerPatternId, IID_PPV_ARGS(&items)));
        ComPtr<IUIAutomationInvokePattern> reorder;
        check(swap->GetCurrentPatternAs(UIA_InvokePatternId, IID_PPV_ARGS(&reorder)));
        const wchar_t *firstNames[] = {L"Record 000", L"Record 499"};
        for (const wchar_t *firstName : firstNames) {
            check(reorder->Invoke());
            bool reordered = false;
            ULONGLONG deadline = GetTickCount64() + 8000;
            do {
                VARIANT any; VariantInit(&any);
                ComPtr<IUIAutomationElement> first;
                check(items->FindItemByProperty(nullptr, 0, any, &first));
                expect(first.Get() != nullptr, "Reorder removed the first record");
                BSTR name = nullptr; check(first->get_CurrentName(&name));
                reordered = name && !wcscmp(name, firstName);
                SysFreeString(name);
                if (!reordered) Sleep(10);
            } while (!reordered && GetTickCount64() < deadline);
            expect(reordered, "Asynchronous reorder did not complete");
            BOOL focused = FALSE; check(checkbox->get_CurrentHasKeyboardFocus(&focused));
            expect(focused != FALSE, "Reorder moved focus with AutoSetFocus disabled");
            BOOL offscreen = TRUE; check(checkbox->get_CurrentIsOffscreen(&offscreen));
            expect(offscreen == FALSE, "Reorder hid the active checkbox");
        }
        ComPtr<IUIAutomationTogglePattern> toggle;
        check(checkbox->GetCurrentPatternAs(UIA_TogglePatternId, IID_PPV_ARGS(&toggle)));
        const wchar_t *results[] = {L"Record 499 enabled: true", L"Record 499 enabled: false"};
        for (const wchar_t *result : results) {
            ComPtr<LiveEvents> changed; changed.Attach(new LiveEvents(result));
            check(uia->AddAutomationEventHandler(UIA_LiveRegionChangedEventId, gridStatus.Get(), TreeScope_Element, nullptr, changed.Get()));
            check(toggle->Toggle());
            ULONGLONG deadline = GetTickCount64() + 8000;
            while (changed->count < 1 && GetTickCount64() < deadline) Sleep(10);
            expect(changed->count == 1 && !changed->bad, "Checkbox status did not deliver its current live announcement");
            BOOL focused = FALSE; check(checkbox->get_CurrentHasKeyboardFocus(&focused));
            expect(focused != FALSE, "Checkbox toggle or announcement moved keyboard focus");
            check(uia->RemoveAutomationEventHandler(UIA_LiveRegionChangedEventId, gridStatus.Get(), changed.Get()));
        }
        puts("Keyed checkbox toggles delivered live status while retaining semantic focus");
    }
    CoUninitialize();
    return 0;
}
