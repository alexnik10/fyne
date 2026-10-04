param([long]$WindowHandle)
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
$window = [System.Windows.Automation.AutomationElement]::FromHandle([IntPtr]$WindowHandle)
function Find-Name([string]$name) {
    $condition = New-Object System.Windows.Automation.PropertyCondition([System.Windows.Automation.AutomationElement]::NameProperty, $name)
    return $window.FindFirst([System.Windows.Automation.TreeScope]::Descendants, $condition)
}
function Wait-Until([scriptblock]$check) {
    $deadline = [DateTime]::UtcNow.AddSeconds(5)
    do {
        try { if (& $check) { return } }
        catch {
            if (!($_.Exception -is [System.Windows.Automation.ElementNotAvailableException]) -and
                !($_.Exception.InnerException -is [System.Windows.Automation.ElementNotAvailableException])) { throw }
        }
        Start-Sleep -Milliseconds 25
    } while ([DateTime]::UtcNow -lt $deadline)
    throw "UIA navigation condition timed out: $check"
}
function Invoke-Name([string]$name) { (Find-Name $name).GetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern).Invoke() }
function Runtime-ID($element) { return [string]::Join(',', $element.GetRuntimeId()) }
$tabs=Find-Name 'Sections'
if ($tabs.Current.ControlType -ne [System.Windows.Automation.ControlType]::Tab) { throw 'Wrong tab role' }
if (Find-Name 'Native table') { throw 'Inactive page exposed' }
$reports=Find-Name 'Reports'
if ($reports.Current.ControlType -ne [System.Windows.Automation.ControlType]::TabItem) { throw 'Wrong tab item role' }
$reports.GetCurrentPattern([System.Windows.Automation.SelectionItemPattern]::Pattern).Select()
$table=Find-Name 'Native table'
if (!$table) { throw 'Selected page missing' }
$grid=$table.GetCurrentPattern([System.Windows.Automation.GridPattern]::Pattern)
if ($grid.Current.RowCount -ne 120 -or $grid.Current.ColumnCount -ne 2) { throw 'Wrong grid dimensions' }
$cell=$grid.GetItem(119,1)
$id=Runtime-ID $cell
if ($cell.Current.Name -ne 'record-119 State' -or !$cell.Current.IsOffscreen) { throw 'Wrong virtual cell' }
$item=$cell.GetCurrentPattern([System.Windows.Automation.GridItemPattern]::Pattern)
if ($item.Current.Row -ne 119 -or $item.Current.Column -ne 1 -or $item.Current.RowSpan -ne 1) { throw 'Wrong coordinates' }
if ((Runtime-ID $item.Current.ContainingGrid) -ne (Runtime-ID $table)) { throw 'Wrong grid owner' }
$headers=$cell.GetCurrentPattern([System.Windows.Automation.TableItemPattern]::Pattern).Current.GetColumnHeaderItems()
if ($headers.Length -ne 1 -or $headers[0].Current.Name -ne 'State') { throw 'Wrong column header relation' }
$allHeaders=$table.GetCurrentPattern([System.Windows.Automation.TablePattern]::Pattern).Current.GetColumnHeaders()
if ($allHeaders.Length -ne 2) { throw 'Wrong table header count' }
$cell.GetCurrentPattern([System.Windows.Automation.ScrollItemPattern]::Pattern).ScrollIntoView()
Wait-Until { !$cell.Current.IsOffscreen }
$cell.GetCurrentPattern([System.Windows.Automation.SelectionItemPattern]::Pattern).Select()
Invoke-Name 'Reverse table'
Wait-Until { $grid.GetItem(0,1).Current.Name -eq 'record-119 State' }
$cell=$grid.GetItem(0,1)
if ((Runtime-ID $cell) -ne $id) { throw 'Reorder changed cell identity' }
if (!$cell.GetCurrentPattern([System.Windows.Automation.SelectionItemPattern]::Pattern).Current.IsSelected) { throw 'Reorder lost selection' }
Invoke-Name 'Replace table target'
Wait-Until {
    $replacement=$grid.GetItem(119,1)
    $replacement.Current.Name -eq 'record-119 State' -and (Runtime-ID $replacement) -ne $id
}
try { $cell.GetCurrentPattern([System.Windows.Automation.SelectionItemPattern]::Pattern).Select(); throw 'Stale cell accepted' }
catch {
    if (!($_.Exception -is [System.Windows.Automation.ElementNotAvailableException]) -and
        !($_.Exception.InnerException -is [System.Windows.Automation.ElementNotAvailableException])) { throw }
}
Invoke-Name 'Open menu'
Wait-Until { $null -ne (Find-Name 'More') }
if (Find-Name 'Native table') { throw 'Menu did not isolate background scope' }
$more=Find-Name 'More'
if ($more.Current.ControlType -ne [System.Windows.Automation.ControlType]::MenuItem) { throw 'Wrong menu role' }
$more.GetCurrentPattern([System.Windows.Automation.ExpandCollapsePattern]::Pattern).Expand()
Wait-Until { $null -ne (Find-Name 'Nested command') }
$nested=Find-Name 'Nested command'
if ($nested.Current.IsOffscreen) { throw 'Submenu clipped by parent' }
Invoke-Name 'Nested command'
Wait-Until { $null -ne (Find-Name 'Native table') }
Write-Output 'UIA tabs, menu scope, grid coordinates, headers and keyed cell lifetime passed'
