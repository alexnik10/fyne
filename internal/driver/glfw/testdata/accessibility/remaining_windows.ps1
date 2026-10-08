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
$grid = Named 'Keyed wrapping grid'
$items = $grid.GetCurrentPattern([System.Windows.Automation.ItemContainerPattern]::Pattern)
$last = $items.FindItemByProperty($null,[System.Windows.Automation.AutomationElement]::NameProperty,'Record 499')
Assert ($null -ne $last) 'Offscreen GridWrap record missing'
$last.GetCurrentPattern([System.Windows.Automation.ScrollItemPattern]::Pattern).ScrollIntoView()
WaitFor { -not $last.Current.IsOffscreen } 'GridWrap failed to reveal record'
$children = $last.GetCurrentPattern([System.Windows.Automation.ItemContainerPattern]::Pattern)
$flag = $children.FindItemByProperty($null,[System.Windows.Automation.AutomationElement]::NameProperty,'Enable Record 499')
Assert ($null -ne $flag) 'Nested model control missing'
Assert $flag.Current.IsKeyboardFocusable 'Model checkbox must route keyboard focus to its grid'
$flag.SetFocus()
WaitFor { $flag.Current.HasKeyboardFocus } 'Nested checkbox did not receive semantic focus'
$identity = $flag.GetRuntimeId() -join ','
$toggle = $flag.GetCurrentPattern([System.Windows.Automation.TogglePattern]::Pattern)
$toggle.Toggle()
Assert ($toggle.Current.ToggleState -eq [System.Windows.Automation.ToggleState]::On) 'Nested command did not update model'
Assert $flag.Current.HasKeyboardFocus 'Toggling the nested checkbox moved focus'
(Named 'Swap first and last').GetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern).Invoke()
WaitFor { $items.FindItemByProperty($null, $null, $null).Current.Name -eq 'Record 499' } 'Asynchronous reorder did not move Record 499 to the first position'
Assert (($flag.GetRuntimeId() -join ',') -eq $identity) 'Reorder replaced the checkbox identity'
Write-Output ("After reorder: focus={0}, offscreen={1}, bounds={2}, focused name={3}" -f $flag.Current.HasKeyboardFocus, $flag.Current.IsOffscreen, $flag.Current.BoundingRectangle, [System.Windows.Automation.AutomationElement]::FocusedElement.Current.Name)
Assert $flag.Current.HasKeyboardFocus 'Reorder lost the active checkbox focus'
Assert (-not $flag.Current.IsOffscreen) 'Reorder hid the active checkbox'
$toggle.Toggle()
Assert ($toggle.Current.ToggleState -eq [System.Windows.Automation.ToggleState]::Off) 'Old provider did not target the reordered record'
Assert $flag.Current.HasKeyboardFocus 'Second toggle moved focus'
Write-Output 'Remaining controls passed through external UIA'
