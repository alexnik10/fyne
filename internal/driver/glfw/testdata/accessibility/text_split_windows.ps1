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
$grid = Named 'Text grid document'
$second = Named 'Pane two document'
$firstText = $grid.GetCurrentPattern([System.Windows.Automation.TextPattern]::Pattern)
Assert ($firstText.GetSelection()[0].GetText(-1) -eq 'Русский текст 😀') 'Native keyboard selection was not exposed to UIA'
$expected = "Read-only TextGrid`nColumns`tValue`nРусский текст 😀`nUse the arrow keys to read; Shift selects and Control+C copies."
Assert ($firstText.DocumentRange.GetText(-1) -eq $expected) 'TextGrid lost tabs, lines or Unicode'
$identity = $grid.GetRuntimeId() -join ','
$range = (Named 'Resize panes').GetCurrentPattern([System.Windows.Automation.RangeValuePattern]::Pattern)
foreach ($offset in @($range.Current.Minimum, $range.Current.Maximum)) {
    $range.SetValue($offset)
    foreach ($document in @($grid, $second)) {
        Assert ($document.Current.ControlType -eq [System.Windows.Automation.ControlType]::Document) 'Wrong document role'
        Assert $document.Current.IsKeyboardFocusable 'Document is not a keyboard target'
        Assert (-not $document.Current.IsOffscreen) 'Resizing hid a document'
        $document.SetFocus()
        Assert $document.Current.HasKeyboardFocus 'Document focus did not reach the canvas'
        $text = $document.GetCurrentPattern([System.Windows.Automation.TextPattern]::Pattern)
        Assert ($text.DocumentRange.GetAttributeValue([System.Windows.Automation.TextPattern]::IsReadOnlyAttribute)) 'Document is not read-only'
        Assert ($text.SupportedTextSelection -eq [System.Windows.Automation.SupportedTextSelection]::Single) 'Reading caret/selection is unavailable'
        $text.DocumentRange.Select()
        Assert ($text.GetSelection()[0].GetText(-1) -eq $text.DocumentRange.GetText(-1)) 'Read-only selection did not round-trip'
        $caret = $text.DocumentRange.Clone()
        $caret.MoveEndpointByRange([System.Windows.Automation.Text.TextPatternRangeEndpoint]::End, $caret, [System.Windows.Automation.Text.TextPatternRangeEndpoint]::Start)
        $caret.Select()
        Assert ($text.GetSelection().Length -eq 1 -and $text.GetSelection()[0].GetText(-1) -eq '') 'Collapsed reading caret is unavailable to NVDA'
    }
    Assert (($grid.GetRuntimeId() -join ',') -eq $identity) 'Resizing replaced the text document'
    Assert ($firstText.DocumentRange.GetText(-1) -eq $expected) 'Resizing changed TextGrid content'
}
$emoji = $firstText.DocumentRange.FindText('😀', $false, $false)
Assert ($null -ne $emoji) 'Emoji was not found'
$emoji.Select()
Assert ($firstText.GetSelection()[0].GetText(-1) -eq '😀') 'UTF-16 selection split the emoji'
Write-Output 'Both read-only panes expose keyboard focus, text and selection after resizing'
