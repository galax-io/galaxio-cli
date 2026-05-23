// Package metrics is rendered only when MetricsEnabled=true.
// This file is an example of a conditional plugin overlay.
package metrics

// Collector is a placeholder metrics collector for {{ .Name }}.
type Collector struct {
	ServiceName string
}

// New returns a Collector for the {{ .Name }} service.
func New() *Collector {
	return &Collector{ServiceName: "{{ .Name }}"}
}
