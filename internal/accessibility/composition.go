package accessibility

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/internal/cache"
)

func compositionMode(obj fyne.CanvasObject) fyne.AccessibilityMode {
	if c, ok := obj.(fyne.AccessibleComposition); ok {
		if mode := c.AccessibilityMode(); mode <= fyne.AccessibilityExclude {
			return mode
		}
	}
	return fyne.AccessibilityAuto
}

func semanticChildren(obj fyne.CanvasObject, mode fyne.AccessibilityMode) []fyne.CanvasObject {
	if mode == fyne.AccessibilitySingle || mode == fyne.AccessibilityExclude {
		return nil
	}
	if c, ok := obj.(fyne.AccessibleChildren); ok {
		return c.AccessibilityChildren()
	}
	if c, ok := obj.(*fyne.Container); ok {
		return c.Objects
	}
	if _, accessible := obj.(fyne.Accessible); accessible && mode == fyne.AccessibilityAuto {
		return nil
	}
	if w, ok := obj.(fyne.Widget); ok {
		// Use the same renderer as layout and input. Never call CreateRenderer
		// directly: that would create duplicate children and unstable identities.
		if r := cache.Renderer(w); r != nil {
			return r.Objects()
		}
	}
	return nil
}

type bounds struct {
	position fyne.Position
	size     fyne.Size
}

func clippedBounds(pos fyne.Position, size fyne.Size, clip *bounds) (fyne.Position, fyne.Size) {
	if clip == nil {
		return pos, size
	}
	left, top := max(pos.X, clip.position.X), max(pos.Y, clip.position.Y)
	right := min(pos.X+size.Width, clip.position.X+clip.size.Width)
	bottom := min(pos.Y+size.Height, clip.position.Y+clip.size.Height)
	return fyne.NewPos(left, top), fyne.NewSize(max(0, right-left), max(0, bottom-top))
}

func childClip(obj fyne.CanvasObject, pos fyne.Position, clip *bounds) *bounds {
	_, clips := obj.(fyne.Scrollable)
	if w, ok := obj.(fyne.Widget); ok {
		if r, found := cache.CachedRenderer(w); found {
			_, rendererClips := r.(interface{ IsClip() })
			clips = clips || rendererClips
		}
	}
	if !clips {
		return clip
	}
	position, size := clippedBounds(pos, obj.Size(), clip)
	return &bounds{position: position, size: size}
}

// Roots describes the same overlay scope for native adapters and public tests.
// Extra roots may contain platform-specific content such as a window menu.
func Roots(canvas fyne.Canvas, extra ...fyne.CanvasObject) []Root {
	overlays := canvas.Overlays().List()
	roots := []Root{{Object: canvas.Content(), Suppressed: len(overlays) != 0}}
	for _, obj := range extra {
		roots = append(roots, Root{Object: obj, Suppressed: len(overlays) != 0})
	}
	for i, overlay := range overlays {
		roots = append(roots, Root{Object: overlay, Suppressed: i != len(overlays)-1})
	}
	if owner, ok := canvas.Overlays().Top().(fyne.AccessibleOverlayOwner); ok {
		if scope := owner.AccessibilityOverlayOwner(); scope != nil {
			for i := range roots {
				roots[i].Suppressed, roots[i].Scope = false, scope
			}
		}
	}
	return roots
}

func (t *Tree) snapshotNode(obj fyne.CanvasObject, pos fyne.Position, parent uint32, describers []fyne.AccessibleChildDescriber, clip *bounds) Node {
	id := t.ids[obj]
	if id == 0 {
		t.next++
		id = t.next
		t.ids[obj] = id
	}
	return t.snapshotObject(obj, id, pos, parent, describers, clip)
}

func (t *Tree) snapshotObject(obj fyne.CanvasObject, id uint32, pos fyne.Position, parent uint32, describers []fyne.AccessibleChildDescriber, clip *bounds) Node {
	n := Node{ID: id, Parent: parent, Role: fyne.AccessibleRoleContainer, Position: pos, Size: obj.Size()}
	if a, ok := obj.(fyne.Accessible); ok {
		n.Name, n.Role = a.AccessibilityLabel(), a.AccessibilityRole()
	}
	n.BoundsPosition, n.BoundsSize = clippedBounds(pos, obj.Size(), clip)
	// Layout-only containers need not be spoken as "Container".
	if _, plain := obj.(*fyne.Container); plain {
		n.Name = ""
	}
	for _, d := range describers {
		applyInfo(&n, d.AccessibilityChildInfo(obj))
	}
	if d, ok := obj.(fyne.AccessibleDescribed); ok {
		applyInfo(&n, d.AccessibilityInfo())
	}
	populateCapabilities(&n, obj)
	if n.Document != nil && clip != nil {
		viewport, size := clippedBounds(pos.Add(n.Document.ViewportPosition), n.Document.ViewportSize, clip)
		n.Document.ViewportPosition = fyne.NewPos(viewport.X-pos.X, viewport.Y-pos.Y)
		n.Document.ViewportSize = size
	}
	t.nodes[id] = n
	t.objects[id] = obj
	return n
}

func (t *Tree) checkUnrepresented(obj fyne.CanvasObject) {
	if d, ok := obj.(fyne.AccessibleDescribed); ok && d.AccessibilityInfo() != (fyne.AccessibilityInfo{}) {
		t.Issues = append(t.Issues, Issue{obj, "unrepresented-metadata", "Metadata needs Accessible semantics or Group mode."})
	}
	if _, focusable := obj.(fyne.Focusable); focusable {
		t.Issues = append(t.Issues, Issue{obj, "missing-semantics", "A keyboard control has no Accessible semantics."})
	}
}
