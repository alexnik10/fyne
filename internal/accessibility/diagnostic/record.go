// Package diagnostic provides opt-in, in-memory timing for accessibility demos.
// Callers supply only fixed operation/control names and numeric measurements,
// never field contents, accessible names, key text, or password data.
package diagnostic

import (
	"encoding/json"
	"io"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

const maximumSamples = 20000

type Control interface {
	DiagnosticControlID() string
}

type Sample struct {
	Kind       string `json:"kind"`
	AtUS       int64  `json:"at_us"`
	Window     uint64 `json:"window,omitempty"`
	Control    string `json:"control,omitempty"`
	DurationUS int64  `json:"duration_us,omitempty"`
	Count      int    `json:"count,omitempty"`
	Node       uint32 `json:"node,omitempty"`
	EventID    int    `json:"event_id,omitempty"`
	EventKind  int    `json:"event_kind,omitempty"`
}

type Summary struct {
	Samples int64 `json:"samples"`
	TotalUS int64 `json:"total_us"`
	MaxUS   int64 `json:"max_us"`
}

type Report struct {
	Schema    int                `json:"schema"`
	GoVersion string             `json:"go_version"`
	OS        string             `json:"os"`
	Arch      string             `json:"arch"`
	Revision  string             `json:"revision,omitempty"`
	Started   time.Time          `json:"started_utc"`
	Dropped   int64              `json:"dropped_samples"`
	Summary   map[string]Summary `json:"summary"`
	Samples   []Sample           `json:"samples"`
}

type recorder struct {
	sync.Mutex
	started time.Time
	samples []Sample
	summary map[string]Summary
	windows map[any]uint64
	dropped int64
}

var recording = recorder{started: time.Now(), summary: make(map[string]Summary), windows: make(map[any]uint64)}

func WindowID(window any) uint64 {
	if !Enabled {
		return 0
	}
	recording.Lock()
	defer recording.Unlock()
	if id, ok := recording.windows[window]; ok {
		return id
	}
	id := uint64(len(recording.windows) + 1)
	recording.windows[window] = id
	return id
}

func Start() time.Time {
	if Enabled {
		return time.Now()
	}
	return time.Time{}
}

func Duration(kind string, window uint64, control string, start time.Time) {
	if Enabled {
		Add(Sample{Kind: kind, Window: window, Control: control, DurationUS: time.Since(start).Microseconds()})
	}
}

func Add(sample Sample) {
	if !Enabled {
		return
	}
	recording.Lock()
	defer recording.Unlock()
	sample.AtUS = time.Since(recording.started).Microseconds()
	recording.add(sample)
}

func (r *recorder) add(sample Sample) {
	key := sample.Kind
	if sample.Control != "" {
		key += "/" + sample.Control
	}
	metric := r.summary[key]
	metric.Samples++
	metric.TotalUS += sample.DurationUS
	metric.MaxUS = max(metric.MaxUS, sample.DurationUS)
	r.summary[key] = metric
	if len(r.samples) < maximumSamples {
		r.samples = append(r.samples, sample)
	} else {
		r.dropped++
	}
}

func Write(out io.Writer) error {
	recording.Lock()
	report := Report{Schema: 1, Started: recording.started.UTC(), Dropped: recording.dropped,
		Summary: make(map[string]Summary, len(recording.summary)), Samples: append([]Sample(nil), recording.samples...)}
	for key, value := range recording.summary {
		report.Summary[key] = value
	}
	recording.Unlock()
	report.GoVersion, report.OS, report.Arch = runtime.Version(), runtime.GOOS, runtime.GOARCH
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				report.Revision = setting.Value
			}
		}
	}
	// Disk I/O and encoding happen only on explicit save/exit, outside the lock.
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
