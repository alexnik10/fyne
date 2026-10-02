param([long]$WindowHandle)
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
Add-Type @'
using System;
using System.Runtime.InteropServices;
public static class NativeListWindow {
    [DllImport("user32.dll")]
    public static extern IntPtr GetForegroundWindow();
}
'@
$window = [System.Windows.Automation.AutomationElement]::FromHandle([IntPtr]$WindowHandle)
function Find-Name([string]$name) {
    $condition = New-Object System.Windows.Automation.PropertyCondition([System.Windows.Automation.AutomationElement]::NameProperty, $name)
    return $window.FindFirst([System.Windows.Automation.TreeScope]::Descendants, $condition)
}
function Wait-Until([scriptblock]$check) {
    $deadline = [DateTime]::UtcNow.AddSeconds(5)
    do {
        if (& $check) { return }
        Start-Sleep -Milliseconds 25
    } while ([DateTime]::UtcNow -lt $deadline)
    throw "UIA list condition timed out: $check"
}
function Invoke-Name([string]$name) {
    (Find-Name $name).GetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern).Invoke()
}
function Runtime-ID($element) { return [string]::Join(',', $element.GetRuntimeId()) }
$list = Find-Name 'Native list'
if ($list.Current.ControlType -ne [System.Windows.Automation.ControlType]::List) { throw 'Wrong list role' }
$selection = $list.GetCurrentPattern([System.Windows.Automation.SelectionPattern]::Pattern)
if ($selection.Current.CanSelectMultiple -or $selection.Current.IsSelectionRequired) { throw 'Wrong selection policy' }
$item = Find-Name 'item-119'
$id = Runtime-ID $item
if ($item.Current.ControlType -ne [System.Windows.Automation.ControlType]::ListItem) { throw 'Wrong item role' }
if (!$item.Current.IsOffscreen) { throw 'Unrendered last item must be offscreen' }
$parent = [System.Windows.Automation.TreeWalker]::RawViewWalker.GetParent($item)
if ((Runtime-ID $parent) -ne (Runtime-ID $list)) { throw 'Wrong semantic parent' }
$item.GetCurrentPattern([System.Windows.Automation.ScrollItemPattern]::Pattern).ScrollIntoView()
Wait-Until { !$item.Current.IsOffscreen }
$select = $item.GetCurrentPattern([System.Windows.Automation.SelectionItemPattern]::Pattern)
if ($select.Current.IsSelected) { throw 'Scrolling selected an item' }
$select.Select()
if (!$select.Current.IsSelected) { throw 'Selection did not update' }
if ((Runtime-ID $select.Current.SelectionContainer) -ne (Runtime-ID $list)) { throw 'Wrong selection owner' }
$item.SetFocus()
Invoke-Name 'Reverse list'
Wait-Until { [System.Windows.Automation.TreeWalker]::RawViewWalker.GetFirstChild($list).Current.Name -eq 'item-119' }
$item = Find-Name 'item-119'
if ((Runtime-ID $item) -ne $id) { throw 'Reorder changed identity' }
# UIA can focus the invoking button before its callback. Return to the list
# owner, not to the row: this must restore the previous model-key highlight.
$list.SetFocus()
if ([NativeListWindow]::GetForegroundWindow().ToInt64() -eq $WindowHandle) {
    Wait-Until { $item.Current.HasKeyboardFocus -or [NativeListWindow]::GetForegroundWindow().ToInt64() -ne $WindowHandle }
}
if (!$select.Current.IsSelected) { throw 'Reorder lost selection' }
if ((Runtime-ID $selection.Current.GetSelection()[0]) -ne $id) { throw 'Selection container lost the selected model item' }
Invoke-Name 'Replace target'
Wait-Until { (Runtime-ID (Find-Name 'item-119')) -ne $id }
$rejected = $false
try { $select.Select() } catch { $rejected = $true }
if (!$rejected) { throw 'Replaced stale command was accepted' }
$item = Find-Name 'item-119'
if ($item.GetCurrentPattern([System.Windows.Automation.SelectionItemPattern]::Pattern).Current.IsSelected) { throw 'Replacement inherited selection' }
$item.SetFocus()
Write-Output 'List roles, hierarchy, scroll, selection, focus, reorder and between-snapshot replacement passed.'
