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
        if (& $check) { return }
        Start-Sleep -Milliseconds 25
    } while ([DateTime]::UtcNow -lt $deadline)
    throw "UIA tree condition timed out: $check"
}
function Invoke-Name([string]$name) {
    (Find-Name $name).GetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern).Invoke()
}
function Runtime-ID($element) { return [string]::Join(',', $element.GetRuntimeId()) }
$tree = Find-Name 'Native tree'
if ($tree.Current.ControlType -ne [System.Windows.Automation.ControlType]::Tree) { throw 'Wrong tree role' }
$branch = Find-Name 'folder'
$expand = $branch.GetCurrentPattern([System.Windows.Automation.ExpandCollapsePattern]::Pattern)
if (Find-Name 'item-119') { throw 'Collapsed child was exposed' }
$expand.Expand()
Wait-Until { $null -ne (Find-Name 'item-119') }
$leaf = Find-Name 'item-119'
$id = Runtime-ID $leaf
if ($leaf.Current.ControlType -ne [System.Windows.Automation.ControlType]::TreeItem) { throw 'Wrong item role' }
if (!$leaf.Current.IsOffscreen) { throw 'Unrendered last item must be offscreen' }
$parent = [System.Windows.Automation.TreeWalker]::RawViewWalker.GetParent($leaf)
if ((Runtime-ID $parent) -ne (Runtime-ID $branch)) { throw 'Wrong semantic parent' }
$leafExpand = $leaf.GetCurrentPattern([System.Windows.Automation.ExpandCollapsePattern]::Pattern)
if ($leafExpand.Current.ExpandCollapseState -ne [System.Windows.Automation.ExpandCollapseState]::LeafNode) { throw 'Wrong leaf state' }
$scroll = $leaf.GetCurrentPattern([System.Windows.Automation.ScrollItemPattern]::Pattern)
$scroll.ScrollIntoView()
Wait-Until { !$leaf.Current.IsOffscreen }
$select = $leaf.GetCurrentPattern([System.Windows.Automation.SelectionItemPattern]::Pattern)
if ($select.Current.IsSelected) { throw 'Scrolling selected an item' }
$select.Select()
if (!$select.Current.IsSelected) { throw 'Selection did not update' }
if ((Runtime-ID $select.Current.SelectionContainer) -ne (Runtime-ID $tree)) { throw 'Wrong selection owner' }
$leaf.SetFocus()
$expand.Collapse()
Wait-Until { $null -eq (Find-Name 'item-119') }
$rejected = $false
try { $select.Select() } catch { $rejected = $true }
if (!$rejected) { throw 'Collapsed stale command was accepted' }
$expand.Expand()
Wait-Until { $null -ne (Find-Name 'item-119') }
$leaf = Find-Name 'item-119'
if ((Runtime-ID $leaf) -ne $id) { throw 'Collapse changed identity' }
Invoke-Name 'Reverse tree'
Wait-Until { [System.Windows.Automation.TreeWalker]::RawViewWalker.GetFirstChild($branch).Current.Name -eq 'item-119' }
$leaf = Find-Name 'item-119'
if ((Runtime-ID $leaf) -ne $id) { throw 'Reorder changed identity' }
$select = $leaf.GetCurrentPattern([System.Windows.Automation.SelectionItemPattern]::Pattern)
Invoke-Name 'Remove target'
Wait-Until { $null -eq (Find-Name 'item-119') }
$rejected = $false
try { $select.Select() } catch { $rejected = $true }
if (!$rejected) { throw 'Removed stale command was accepted' }
Invoke-Name 'Restore target'
Wait-Until { $null -ne (Find-Name 'item-119') }
$leaf = Find-Name 'item-119'
if ((Runtime-ID $leaf) -eq $id) { throw 'Restored item reused a removed identity' }
$leaf.SetFocus()
Write-Output 'Tree roles, hierarchy, leaf state, scroll, selection, focus, stable keys and stale commands passed.'
