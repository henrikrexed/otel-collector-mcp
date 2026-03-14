package signals

import "time"

// CapturedSignals holds all parsed signal data from a capture session.
type CapturedSignals struct {
	Metrics   []MetricDataPoint `json:"metrics"`
	Logs      []LogRecord       `json:"logs"`
	Traces    []SpanData        `json:"traces"`
	CaptureAt time.Time         `json:"capture_at"`
	Duration  time.Duration     `json:"duration"`
}

// MetricDataPoint represents a single metric data point.
type MetricDataPoint struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
	Value  float64           `json:"value"`
	Type   string            `json:"type"` // gauge, counter, histogram, summary
}

// LogRecord represents a single log entry.
type LogRecord struct {
	Body               string            `json:"body"`
	Attributes         map[string]string `json:"attributes"`
	ResourceAttributes map[string]string `json:"resource_attributes"`
	Severity           string            `json:"severity"`
	Timestamp          time.Time         `json:"timestamp"`
}

// SpanData represents a single trace span.
type SpanData struct {
	TraceID      string            `json:"trace_id"`
	SpanID       string            `json:"span_id"`
	ParentSpanID string            `json:"parent_span_id"`
	Name         string            `json:"name"`
	Attributes   map[string]string `json:"attributes"`
	Events       []SpanEvent       `json:"events"`
	Duration     time.Duration     `json:"duration"`
	StartTime    time.Time         `json:"start_time"`
}

// SpanEvent represents an event attached to a span.
type SpanEvent struct {
	Name       string            `json:"name"`
	Attributes map[string]string `json:"attributes"`
	Timestamp  time.Time         `json:"timestamp"`
}

// Summary provides aggregate statistics about captured signals.
func (cs *CapturedSignals) Summary() map[string]interface{} {
	uniqueMetrics := make(map[string]struct{})
	for _, m := range cs.Metrics {
		uniqueMetrics[m.Name] = struct{}{}
	}

	uniqueTraces := make(map[string]struct{})
	for _, s := range cs.Traces {
		uniqueTraces[s.TraceID] = struct{}{}
	}

	return map[string]interface{}{
		"metrics.data_points":        len(cs.Metrics),
		"metrics.unique_metric_names": len(uniqueMetrics),
		"logs.records":                len(cs.Logs),
		"traces.spans":                len(cs.Traces),
		"traces.unique_trace_ids":     len(uniqueTraces),
	}
}
