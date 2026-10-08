package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/internal/cache"
)

// AccessibilityLabel leaves the accordion name to application metadata.
// Since: 2.9
func (*Accordion) AccessibilityLabel() string { return "" }

// AccessibilityRole identifies a group of expandable sections.
// Since: 2.9
func (*Accordion) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleContainer }

// AccessibilityChildren places each section's content immediately after its header.
// Since: 2.9
func (a *Accordion) AccessibilityChildren() []fyne.CanvasObject {
	r, ok := cache.Renderer(a).(*accordionRenderer)
	if !ok {
		return nil
	}
	children := make([]fyne.CanvasObject, 0, len(a.Items)*2)
	for i, item := range a.Items {
		if i < len(r.headers) {
			children = append(children, r.headers[i])
		}
		if item.Open && item.Detail != nil {
			children = append(children, item.Detail)
		}
	}
	return children
}

type accordionHeader struct {
	Button
	owner *Accordion
	item  *AccordionItem
}

func (h *accordionHeader) AccessibilityExpanded() bool { return h.item.Open }
func (h *accordionHeader) AccessibilitySetExpanded(expanded bool) {
	for i, item := range h.owner.Items {
		if item != h.item {
			continue
		}
		if expanded {
			h.owner.Open(i)
		} else {
			h.owner.Close(i)
		}
		return
	}
}

func (h *accordionHeader) TypedKey(event *fyne.KeyEvent) {
	switch event.Name {
	case fyne.KeyRight:
		h.AccessibilitySetExpanded(true)
	case fyne.KeyLeft:
		h.AccessibilitySetExpanded(false)
	default:
		h.Button.TypedKey(event)
	}
}

func (r *accordionRenderer) syncHeaders() {
	old := make(map[*AccordionItem]*accordionHeader)
	for _, header := range r.headers {
		old[header.item] = header
	}
	items := make([]*accordionHeader, len(r.container.Items))
	for i, item := range r.container.Items {
		header := old[item]
		if header == nil {
			header = &accordionHeader{owner: r.container, item: item}
			header.ExtendBaseWidget(header)
		}
		delete(old, item)
		items[i] = header
	}
	for _, header := range old {
		header.Hide()
	}
	r.headers = items
}
