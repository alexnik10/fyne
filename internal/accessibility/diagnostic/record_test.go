package diagnostic

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBoundedRecordingKeepsCounters(t *testing.T) {
	r := recorder{summary: make(map[string]Summary)}
	for i := 0; i < maximumSamples+3; i++ {
		r.add(Sample{Kind: "toggle_on", Control: "disable", DurationUS: 5})
	}
	assert.Len(t, r.samples, maximumSamples)
	assert.EqualValues(t, 3, r.dropped)
	assert.EqualValues(t, maximumSamples+3, r.summary["toggle_on/disable"].Samples)
	assert.EqualValues(t, 5*(maximumSamples+3), r.summary["toggle_on/disable"].TotalUS)
	assert.EqualValues(t, 5, r.summary["toggle_on/disable"].MaxUS)
}

type reentrantWriter struct {
	strings.Builder
}

func (w *reentrantWriter) Write(data []byte) (int, error) {
	// Saving must not hold the recorder lock while calling arbitrary I/O.
	Add(Sample{Kind: "during_save"})
	return w.Builder.Write(data)
}

func TestWriteSnapshotDoesNotHoldLock(t *testing.T) {
	recording.Lock()
	before := recording.summary["during_save"]
	recording.Unlock()
	var out reentrantWriter
	done := make(chan error, 1)
	go func() { done <- Write(&out) }()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("report writer blocked recording")
	}
	var report Report
	require.NoError(t, json.Unmarshal([]byte(out.String()), &report))
	assert.Equal(t, 1, report.Schema)
	assert.Equal(t, before, report.Summary["during_save"])
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestWriteReportsFailure(t *testing.T) {
	assert.ErrorIs(t, Write(failingWriter{}), io.ErrClosedPipe)
}
