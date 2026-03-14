package signals

import (
	"bufio"
	"strings"
	"time"
)

// Parse parses debug exporter stdout output into structured signal data.
// The debug exporter output format varies by OTel Collector version.
// This parser handles the v0.90+ format with verbosity: basic.
func Parse(output string, captureStart time.Time, duration time.Duration) *CapturedSignals {
	signals := &CapturedSignals{
		CaptureAt: captureStart,
		Duration:  duration,
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	var currentSection string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Detect section headers from debug exporter output
		switch {
		case strings.Contains(line, "MetricsExporter"):
			currentSection = "metrics"
		case strings.Contains(line, "LogsExporter"):
			currentSection = "logs"
		case strings.Contains(line, "TracesExporter"):
			currentSection = "traces"
		case strings.HasPrefix(line, "Metric #") || strings.HasPrefix(line, "NumberDataPoints"):
			currentSection = "metrics"
		case strings.HasPrefix(line, "LogRecord #"):
			currentSection = "logs"
		case strings.HasPrefix(line, "Span #"):
			currentSection = "traces"
		}

		// Parse based on current section
		switch currentSection {
		case "metrics":
			if dp := parseMetricLine(line); dp != nil {
				signals.Metrics = append(signals.Metrics, *dp)
			}
		case "logs":
			if lr := parseLogLine(line); lr != nil {
				signals.Logs = append(signals.Logs, *lr)
			}
		case "traces":
			if sd := parseSpanLine(line); sd != nil {
				signals.Traces = append(signals.Traces, *sd)
			}
		}
	}

	return signals
}

func parseMetricLine(line string) *MetricDataPoint {
	// Parse "-> Name: metric.name" format
	if strings.HasPrefix(line, "-> Name:") {
		name := strings.TrimSpace(strings.TrimPrefix(line, "-> Name:"))
		return &MetricDataPoint{Name: name, Labels: make(map[string]string)}
	}
	return nil
}

func parseLogLine(line string) *LogRecord {
	// Parse "Body: Str(log message)" format
	if strings.HasPrefix(line, "Body: Str(") {
		body := strings.TrimSuffix(strings.TrimPrefix(line, "Body: Str("), ")")
		return &LogRecord{
			Body:               body,
			Attributes:         make(map[string]string),
			ResourceAttributes: make(map[string]string),
		}
	}
	return nil
}

func parseSpanLine(line string) *SpanData {
	// Parse "Span #N" to create a new span
	if strings.HasPrefix(line, "Span #") {
		return &SpanData{Attributes: make(map[string]string)}
	}

	// Parse "TraceId: hex" format
	if strings.HasPrefix(line, "TraceId:") {
		return &SpanData{
			TraceID:    strings.TrimSpace(strings.TrimPrefix(line, "TraceId:")),
			Attributes: make(map[string]string),
		}
	}

	return nil
}
