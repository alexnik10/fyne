param([long]$WindowHandle)
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
$root = [System.Windows.Automation.AutomationElement]::FromHandle([IntPtr]$WindowHandle)
function Assert($condition, $message) { if (-not $condition) { throw $message } }
function Named($name) {
    $condition = New-Object System.Windows.Automation.PropertyCondition([System.Windows.Automation.AutomationElement]::NameProperty, $name)
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
$view = Named 'List view'
Assert ($view.Current.ControlType -eq [System.Windows.Automation.ControlType]::Button) 'View control is not a button'
$toggle = $view.GetCurrentPattern([System.Windows.Automation.TogglePattern]::Pattern)
$identity = $view.GetRuntimeId() -join ','
$toggle.Toggle()
WaitFor { $toggle.Current.ToggleState -eq [System.Windows.Automation.ToggleState]::On } 'List view state did not change'
Assert (($view.GetRuntimeId() -join ',') -eq $identity) 'View toggle identity changed'
$file = Named 'report.txt'
$file.GetCurrentPattern([System.Windows.Automation.SelectionItemPattern]::Pattern).Select()
(Named 'Open').GetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern).Invoke()
WaitFor { $null -ne (Named 'Open file') } 'File dialog did not close'
$button = Named 'Open file'
Assert ($button.Current.HelpText -eq 'Selected file: report.txt') 'Open button lost selected-file description'
Assert $button.Current.HasKeyboardFocus 'File dialog failed to restore focus'
Assert ($null -eq (Named 'Selected file')) 'Redundant selected-file field remains'
Write-Output 'Dialog view state, selected-file description and keyboard focus passed through external UIA'
