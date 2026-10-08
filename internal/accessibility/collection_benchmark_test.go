package accessibility_test

import (
	"fmt"
	"strconv"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/internal/accessibility"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func BenchmarkCollectionSnapshots(b *testing.B) {
	app := test.NewApp()
	defer app.Quit()
	for _, size := range []int{1000, 10000, 100000} {
		for _, kind := range []string{"List", "Tree"} {
			b.Run(fmt.Sprintf("%s/%d", kind, size), func(b *testing.B) {
				described := 0
				var object fyne.CanvasObject
				if kind == "List" {
					list := widget.NewList(func() int { return size }, func() fyne.CanvasObject { return widget.NewLabel("Template") }, func(int, fyne.CanvasObject) {})
					list.ItemKey = strconv.Itoa
					list.DescribeItem = func(id int) fyne.AccessibilityInfo {
						described++
						return fyne.AccessibilityInfo{Name: strconv.Itoa(id)}
					}
					object = list
				} else {
					keys := make([]string, size)
					for i := range keys {
						keys[i] = strconv.Itoa(i)
					}
					tree := widget.NewTreeWithStrings(map[string][]string{"": keys})
					tree.DescribeNode = func(id string) fyne.AccessibilityInfo { described++; return fyne.AccessibilityInfo{Name: id} }
					object = tree
				}
				w := test.NewWindow(object)
				defer w.Close()
				w.Resize(fyne.NewSize(300, 300))
				var tree accessibility.Tree
				roots := []accessibility.Root{{Object: object}}
				tree.Build(roots, nil)
				described = 0
				b.ReportAllocs()
				b.ResetTimer()
				nodes := 0
				for i := 0; i < b.N; i++ {
					nodes = len(tree.Build(roots, nil))
				}
				b.StopTimer()
				b.ReportMetric(float64(described)/float64(b.N), "descriptions/op")
				b.ReportMetric(float64(nodes), "nodes/op")
			})
		}
	}
}
