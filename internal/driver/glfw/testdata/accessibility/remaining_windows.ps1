param([long]$WindowHandle)
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
$root = [System.Windows.Automation.AutomationElement]::FromHandle([IntPtr]$WindowHandle)
function Assert($condition, $message) { if (-not $condition) { throw $message } }
function Named($name, $type = $null) {
    $condition = New-Object System.Windows.Automation.PropertyCondition([System.Windows.Automation.AutomationElement]::NameProperty, $name)
    if ($null -ne $type) {
        $role = New-Object System.Windows.Automation.PropertyCondition([System.Windows.Automation.AutomationElement]::ControlTypeProperty, $type)
        $condition = [System.Windows.Automation.AndCondition]::new($condition, $role)
    }
    return $root.FindFirst([System.Windows.Automation.TreeScope]::Descendants, $condition)
}
function WaitFor($test, $message) {
    $deadline = [DateTime]::UtcNow.AddSeconds(8)
    do {
        if (& $test) { return }
        Start-Sleep -Milliseconds 40
    } while ([DateTime]::UtcNow -lt $deadline)
    throw $message
}
$progress = Named 'Native progress'
Assert ($progress.Current.ControlType -eq [System.Windows.Automation.ControlType]::ProgressBar) 'Wrong progress role'
$range = $progress.GetCurrentPattern([System.Windows.Automation.RangeValuePattern]::Pattern)
Assert $range.Current.IsReadOnly 'Progress must be read-only'
Assert ($range.Current.Value -eq 0.25) 'Wrong progress value'
$rejected = $false
try { $range.SetValue(0.75) } catch { $rejected = $true }
Assert $rejected 'Progress accepted a write'
$calendar = Named 'Native calendar'
Assert ($calendar.Current.ControlType -eq [System.Windows.Automation.ControlType]::Calendar) 'Wrong calendar role'
$calendarGrid = $calendar.GetCurrentPattern([System.Windows.Automation.GridPattern]::Pattern)
Assert ($calendarGrid.Current.ColumnCount -eq 7) 'Wrong calendar columns'
for ($r = 0; $r -lt $calendarGrid.Current.RowCount; $r++) {
 for ($c = 0; $c -lt 7; $c++) {
  $cell = $calendarGrid.GetItem($r,$c)
  Assert ($null -ne $cell) 'Missing calendar cell, including trailing blanks'
  $item = $cell.GetCurrentPattern([System.Windows.Automation.GridItemPattern]::Pattern)
  Assert ($item.Current.Row -eq $r -and $item.Current.Column -eq $c) 'Wrong date coordinates'
 }
}
$day = Named 'Monday, 5 October 2026'
$day.GetCurrentPattern([System.Windows.Automation.SelectionItemPattern]::Pattern).Select()
$day.SetFocus()
WaitFor { $day.Current.HasKeyboardFocus } 'Date focus did not reach the canvas'
$selection = $calendar.GetCurrentPattern([System.Windows.Automation.SelectionPattern]::Pattern).Current.GetSelection()
Assert ($selection.Length -eq 1 -and $selection[0].Current.Name -eq 'Monday, 5 October 2026') 'Wrong selected date'
$section = Named 'Native section'
$expand = $section.GetCurrentPattern([System.Windows.Automation.ExpandCollapsePattern]::Pattern)
$expand.Expand()
WaitFor { $null -ne (Named 'Section content') } 'Accordion content missing'
$expand.Collapse()
WaitFor { $null -eq (Named 'Section content') } 'Collapsed content is exposed'
$combo = Named 'Native editable choice'
Assert ($combo.Current.ControlType -eq [System.Windows.Automation.ControlType]::ComboBox) 'Wrong editable combo role'
$value = $combo.GetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern)
$value.SetValue('Free text')
Assert ($value.Current.Value -eq 'Free text') 'Editable combo did not accept text'
(Named 'Second').GetCurrentPattern([System.Windows.Automation.SelectionItemPattern]::Pattern).Select()
Assert ($value.Current.Value -eq 'Second') 'Editable combo selection did not change text'
$text = (Named 'Native text grid').GetCurrentPattern([System.Windows.Automation.TextPattern]::Pattern)
Assert ($text.DocumentRange.GetText(-1) -eq "Read only`nРусский 😀") 'TextGrid lost model text'
Assert ($text.SupportedTextSelection -eq [System.Windows.Automation.SupportedTextSelection]::None) 'TextGrid invents selection'
$grid = Named 'Native wrapping grid'
$items = $grid.GetCurrentPattern([System.Windows.Automation.ItemContainerPattern]::Pattern)
$last = $items.FindItemByProperty($null,[System.Windows.Automation.AutomationElement]::NameProperty,'Record 999')
Assert ($null -ne $last) 'Offscreen GridWrap record missing'
$last.GetCurrentPattern([System.Windows.Automation.ScrollItemPattern]::Pattern).ScrollIntoView()
WaitFor { -not $last.Current.IsOffscreen } 'GridWrap failed to reveal record'
$children = $last.GetCurrentPattern([System.Windows.Automation.ItemContainerPattern]::Pattern)
$flag = $children.FindItemByProperty($null,[System.Windows.Automation.AutomationElement]::NameProperty,'Flag 999')
Assert ($null -ne $flag) 'Nested model control missing'
Assert (-not $flag.Current.IsKeyboardFocusable) 'Temporary model control advertises phantom focus'
$toggle = $flag.GetCurrentPattern([System.Windows.Automation.TogglePattern]::Pattern)
$toggle.Toggle()
Assert ($toggle.Current.ToggleState -eq [System.Windows.Automation.ToggleState]::On) 'Nested command did not update model'
# C# receives UIA callbacks on client threads without a PowerShell runspace.
Add-Type -ReferencedAssemblies UIAutomationClient,UIAutomationTypes -TypeDefinition @'
using System;
using System.Collections.Concurrent;
using System.Windows.Automation;
public static class LiveRegionProbe {
    public static readonly ConcurrentQueue<string> Messages = new ConcurrentQueue<string>();
    public static readonly AutomationEventHandler Handler = OnChanged;
    private static void OnChanged(object sender, AutomationEventArgs args) {
        try { Messages.Enqueue(((AutomationElement)sender).Current.Name); }
        catch (Exception e) { Messages.Enqueue("ERROR: " + e.Message); }
    }
}
'@
$status = Named 'Native ready'
$liveProperty = [System.Windows.Automation.AutomationProperty]::LookupById(30135)
$liveEvent = [System.Windows.Automation.AutomationEvent]::LookupById(20024)
Assert ($null -ne $liveProperty -and $null -ne $liveEvent) 'UIA live-region identifiers unavailable'
Assert ([int]$status.GetCurrentPropertyValue($liveProperty) -eq 1) 'Status is not polite'
$save = Named 'Native save'
$save.SetFocus()
[System.Windows.Automation.Automation]::AddAutomationEventHandler($liveEvent, $status, [System.Windows.Automation.TreeScope]::Element, [LiveRegionProbe]::Handler)
try {
    $invoke = $save.GetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern)
    $invoke.Invoke()
    WaitFor { [LiveRegionProbe]::Messages.Count -ge 1 } 'Save did not deliver a live-region event'
    $invoke.Invoke()
    WaitFor { [LiveRegionProbe]::Messages.Count -ge 2 } 'Repeated Save did not announce its identical result'
    $messages = [LiveRegionProbe]::Messages.ToArray()
    Assert ($messages[0] -eq 'Native saved' -and $messages[1] -eq 'Native saved') 'Live event exposed stale text'
    Assert $save.Current.HasKeyboardFocus 'Status announcement moved keyboard focus'
} finally {
    [System.Windows.Automation.Automation]::RemoveAutomationEventHandler($liveEvent, $status, [LiveRegionProbe]::Handler)
}
Write-Output 'Remaining controls and live status passed through external UIA'
