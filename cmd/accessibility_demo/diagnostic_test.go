//go:build accessibilitydiagnostics

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/internal/accessibility/diagnostic"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiagnosticCheckPreservesToggleBehavior(t *testing.T) {
	test.NewTempApp(t)
	changes := 0
	c := newDemoCheck("test_check", "Check", func(bool) { changes++ })
	w := test.NewWindow(c)
	defer w.Close()
	w.Canvas().Focus(c)
	c.TypedRune(' ')
	assert.True(t, c.Checked)
	c.TypedRune(' ')
	assert.False(t, c.Checked)
	c.Tapped(&fyne.PointEvent{})
	assert.True(t, c.Checked)
	c.TypedRune('X') // Unrelated input must not enter the trace.
	assert.Equal(t, 3, changes)
	var out bytes.Buffer
	require.NoError(t, diagnostic.Write(&out))
	var report diagnostic.Report
	require.NoError(t, json.Unmarshal(out.Bytes(), &report))
	assert.EqualValues(t, 2, report.Summary["toggle_on/test_check"].Samples)
	assert.EqualValues(t, 1, report.Summary["toggle_off/test_check"].Samples)
	assert.EqualValues(t, 2, report.Summary["check_space_dispatch/test_check"].Samples)
	assert.EqualValues(t, 1, report.Summary["check_tap/test_check"].Samples)
	assert.EqualValues(t, 3, report.Summary["widget_callback/test_check"].Samples)
	assert.NotContains(t, out.String(), "Check")
}

func TestDiagnosticSaveReportsAndOverwritesOwnFile(t *testing.T) {
	previous := reportPath
	reportPath = filepath.Join(t.TempDir(), "report.json")
	t.Cleanup(func() { reportPath = previous })
	path, err := saveDiagnosticReport()
	require.NoError(t, err)
	assert.Equal(t, reportPath, path)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.True(t, json.Valid(data))
	require.NoError(t, os.WriteFile(path, []byte("stale trailing data"), 0o600))
	_, err = saveDiagnosticReport()
	require.NoError(t, err)
	data, err = os.ReadFile(path)
	require.NoError(t, err)
	assert.True(t, json.Valid(data))
	reportPath = filepath.Join(path, "not-a-directory.json")
	_, err = saveDiagnosticReport()
	assert.Error(t, err)
}
