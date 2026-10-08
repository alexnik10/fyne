// Exercise live-region properties and events through the native UIA client.
// The legacy .NET UIAutomationClient cannot decode the LiveSetting property.
#include <windows.h>
#include <UIAutomation.h>
#include <atomic>
#include <stdio.h>
#include <stdlib.h>
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
    std::atomic<int> count{0};
    std::atomic<bool> bad{false};
    HRESULT STDMETHODCALLTYPE QueryInterface(REFIID iid, void **out) override {
        if (!out) return E_POINTER;
        *out = nullptr;
        if (iid == IID_IUnknown || iid == __uuidof(IUIAutomationEventHandler)) {
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
        if (id != UIA_LiveRegionChangedEventId || FAILED(hr) || !name || wcscmp(name, L"Native saved")) bad = true;
        SysFreeString(name);
        ++count;
        return S_OK;
    }
};

int main(int argc, char **argv) {
    expect(argc == 2, "Expected the provider window handle");
    check(CoInitializeEx(nullptr, COINIT_MULTITHREADED));
    {
        ComPtr<IUIAutomation> uia;
        check(CoCreateInstance(CLSID_CUIAutomation, nullptr, CLSCTX_INPROC_SERVER, IID_PPV_ARGS(&uia)));
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
    }
    CoUninitialize();
    return 0;
}
