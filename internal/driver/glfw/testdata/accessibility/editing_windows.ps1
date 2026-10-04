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
$table = Named 'Editable native table'
$grid = $table.GetCurrentPattern([System.Windows.Automation.GridPattern]::Pattern)
$cell = $grid.GetItem(119, 1)
$value = $cell.GetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern)
Assert (-not $value.Current.IsReadOnly) 'Editable cell is read-only'
$value.SetValue('Direct update')
Assert ($value.Current.Value -eq 'Direct update') 'Offscreen Value did not update model'
$rejected = $false
try { $value.SetValue('') } catch { $rejected = $true }
Assert $rejected 'Invalid cell value was accepted'
Assert ($value.Current.Value -eq 'Direct update') 'Rejected value mutated the model'
$readOnly = $grid.GetItem(0, 0).GetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern)
Assert $readOnly.Current.IsReadOnly 'Read-only column is writable'
$cell.GetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern).Invoke()
WaitFor { $null -ne (Named 'Edit cell') } 'Editor did not open'
$entry = Named 'Cell 119 1' ([System.Windows.Automation.ControlType]::Edit)
Assert ($entry.Current.ControlType -eq [System.Windows.Automation.ControlType]::Edit) 'Wrong editor role'
Assert $entry.Current.HasKeyboardFocus 'Cell editor has no keyboard focus'
$editorValue = $entry.GetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern)
$editorText = $entry.GetCurrentPattern([System.Windows.Automation.TextPattern]::Pattern)
Assert ($editorText.DocumentRange.GetText(-1) -eq 'Direct update') 'Wrong editor text'
$editorValue.SetValue('Saved draft')
(Named 'Save').GetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern).Invoke()
WaitFor { $null -eq (Named 'Edit cell') } 'Editor did not close'
Assert ($value.Current.Value -eq 'Saved draft') 'Save did not update the original cell'
$rejected = $false
try { $editorValue.SetValue('stale editor') } catch { $rejected = $true }
Assert $rejected 'Detached editor accepted a command'
$rich = Named 'Native rich editor'
$text = $rich.GetCurrentPattern([System.Windows.Automation.TextPattern]::Pattern)
$doc = $text.DocumentRange
Assert ($doc.GetAttributeValue([System.Windows.Automation.TextPattern]::FontWeightAttribute) -eq [System.Windows.Automation.TextPattern]::MixedAttributeValue) 'Mixed formatting was lost'
$bold = $doc.FindAttribute([System.Windows.Automation.TextPattern]::FontWeightAttribute, [int]700, $false)
Assert ($bold.GetText(-1) -eq 'bold') 'FindAttribute returned wrong bold range'
$bold.ExpandToEnclosingUnit([System.Windows.Automation.TextUnit]::Format)
Assert ($bold.GetText(-1) -eq 'bold') 'Format unit crossed the run boundary'
$bold.Select()
Assert ($text.GetSelection()[0].GetText(-1) -eq 'bold') 'Rich selection did not round-trip'
$rich.GetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern).SetValue('Replacement')
Assert ($text.DocumentRange.GetText(-1) -eq 'Replacement') 'Rich replacement did not update segments'
$preview = Named 'Native document'
Assert ($preview.Current.ControlType -eq [System.Windows.Automation.ControlType]::Document) 'Wrong document role'
$previewText = $preview.GetCurrentPattern([System.Windows.Automation.TextPattern]::Pattern)
Assert ($previewText.DocumentRange.GetText(-1) -eq "Heading`nPlain bold") 'Document paragraph boundary lost'
$line = $previewText.DocumentRange.Clone()
$line.ExpandToEnclosingUnit([System.Windows.Automation.TextUnit]::Line)
Assert ($line.GetText(-1) -eq "Heading`n") 'Paragraph separator belongs to the wrong rendered line'
Assert ($previewText.SupportedTextSelection -eq [System.Windows.Automation.SupportedTextSelection]::None) 'Nonselectable document advertises selection'
Assert ($previewText.DocumentRange.GetAttributeValue([System.Windows.Automation.TextPattern]::IsReadOnlyAttribute)) 'Document is not read-only'
Write-Output 'Editable Table and formatted text passed through external UIA'
