package widget

import "fyne.io/fyne/v2"

var _ fyne.AccessibleLiveRegion = (*Label)(nil)

// AccessibilityLiveRegion describes automatic announcements of the label's name.
// Since: 2.9
func (l *Label) AccessibilityLiveRegion() (fyne.AccessibilityLiveSetting, uint64) {
	return l.liveSetting, l.liveRevision
}

// SetAccessibilityLiveSetting opts this label into status announcements.
// The default is off. When enabled, SetText requests an announcement even if the
// text is unchanged; Refresh alone does not repeat an unchanged announcement.
// Set this before showing the label. Its accessible name should contain the
// status text; an explicit AccessibilityInfo.Name overrides the displayed text.
// This does not make the label focusable. Native adapter support is required.
// Since: 2.9
func (l *Label) SetAccessibilityLiveSetting(setting fyne.AccessibilityLiveSetting) {
	if setting > fyne.AccessibilityLiveAssertive {
		setting = fyne.AccessibilityLiveOff
	}
	l.liveSetting = setting
	l.Refresh()
}
