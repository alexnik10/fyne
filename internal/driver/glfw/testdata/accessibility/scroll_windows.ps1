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
    throw "UIA scroll condition timed out: $check"
}
function Invoke-Name([string]$name) { (Find-Name $name).GetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern).Invoke() }
function Runtime-ID($element) { return [string]::Join(',', $element.GetRuntimeId()) }
function Focus-ID { return Runtime-ID ([System.Windows.Automation.AutomationElement]::FocusedElement) }
function Scroll-Pattern($element) { return $element.GetCurrentPattern([System.Windows.Automation.ScrollPattern]::Pattern) }
function Assert-Near([double]$actual, [double]$expected) {
    if ([Math]::Abs($actual-$expected) -gt 0.01) { throw "Expected $expected, got $actual" }
}
$none = [System.Windows.Automation.ScrollAmount]::NoAmount
$large = [System.Windows.Automation.ScrollAmount]::LargeIncrement
$smallBack = [System.Windows.Automation.ScrollAmount]::SmallDecrement
$viewport = Find-Name 'Native viewport'
$scroll = Scroll-Pattern $viewport
if (!$scroll.Current.HorizontallyScrollable -or !$scroll.Current.VerticallyScrollable) { throw 'Missing scroll axes' }
if ($scroll.Current.HorizontalViewSize -le 0 -or $scroll.Current.HorizontalViewSize -ge 100 -or
    $scroll.Current.VerticalViewSize -le 0 -or $scroll.Current.VerticalViewSize -ge 100) { throw 'Wrong view size' }
$target = Find-Name 'Scroll target'
$target.SetFocus()
$focus = Focus-ID
$scroll.SetScrollPercent(0, 0)
$scroll.SetScrollPercent(50, 75)
Assert-Near $scroll.Current.HorizontalScrollPercent 50
Assert-Near $scroll.Current.VerticalScrollPercent 75
if ((Focus-ID) -ne $focus) { throw 'Scroll moved keyboard focus' }
$scroll.Scroll($smallBack, $none)
if ($scroll.Current.HorizontalScrollPercent -ge 50) { throw 'Small decrement did not move' }
Assert-Near $scroll.Current.VerticalScrollPercent 75
$scroll.SetScrollPercent(100, 100)
$scroll.Scroll($large, $large)
Assert-Near $scroll.Current.VerticalScrollPercent 100
Assert-Near $scroll.Current.HorizontalScrollPercent 100
$scroll.SetScrollPercent(0, 0)
$target.GetCurrentPattern([System.Windows.Automation.ScrollItemPattern]::Pattern).ScrollIntoView()
if ($target.Current.IsOffscreen) { throw 'Generic ScrollItem did not reveal child' }
if ((Focus-ID) -ne $focus) { throw 'ScrollItem moved focus' }
Invoke-Name 'Open scroll modal'
Wait-Until { $null -ne (Find-Name 'Close scroll modal') }
try { $scroll.SetScrollPercent(0, 0); throw 'Background scroll accepted' }
catch {
    if (!($_.Exception -is [System.Windows.Automation.ElementNotAvailableException]) -and
        !($_.Exception.InnerException -is [System.Windows.Automation.ElementNotAvailableException])) { throw }
}
Invoke-Name 'Close scroll modal'
Wait-Until { $null -ne (Find-Name 'Native viewport') }
Invoke-Name 'Fit viewport content'
Wait-Until { !$scroll.Current.VerticallyScrollable }
if ($scroll.Current.HorizontallyScrollable) { throw 'Fitting content still scrolls' }
Assert-Near $scroll.Current.HorizontalScrollPercent -1
Assert-Near $scroll.Current.VerticalScrollPercent -1
Assert-Near $scroll.Current.HorizontalViewSize 100
Assert-Near $scroll.Current.VerticalViewSize 100
$scroll.SetScrollPercent(-1, -1)
$scroll.Scroll($none, $none)
foreach ($page in @('List', 'Tree', 'Table')) {
    (Find-Name $page).GetCurrentPattern([System.Windows.Automation.SelectionItemPattern]::Pattern).Select()
    $owner = Find-Name ('Scroll ' + $page.ToLowerInvariant())
    $pattern = Scroll-Pattern $owner
    $item = if ($page -eq 'Table') {
        $owner.GetCurrentPattern([System.Windows.Automation.GridPattern]::Pattern).GetItem(2, 2)
    } else { Find-Name ($page + ' row 2') }
    if (!$item) { throw "Missing $page item" }
    $item.SetFocus()
    $selection = $item.GetCurrentPattern([System.Windows.Automation.SelectionItemPattern]::Pattern)
    $selection.Select()
    $focus = Focus-ID
    $pattern.SetScrollPercent(-1, 100)
    Assert-Near $pattern.Current.VerticalScrollPercent 100
    if ($page -eq 'Table') {
        $pattern.SetScrollPercent(100, -1)
        Assert-Near $pattern.Current.HorizontalScrollPercent 100
    }
    if (!$selection.Current.IsSelected -or (Focus-ID) -ne $focus) { throw "$page scroll changed selection or focus" }
    if (!$item.Current.IsOffscreen) { throw "$page viewport did not move" }
    $pattern.Scroll($none, $smallBack)
    if ($pattern.Current.VerticalScrollPercent -ge 100) { throw "$page relative scroll did not move" }
}
Write-Output 'UIA scroll axes, percentages, view size, steps, scope, fitting content, nested commands and collection focus passed'
