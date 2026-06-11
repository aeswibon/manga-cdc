package generate

import (
	"strings"
	"testing"
)

func TestHelmValuesContent(t *testing.T) {
	cfg := Config{
		Eventing:  EventingBoth,
		Notifiers: []string{"discord"},
	}
	output, err := renderHelmValues(cfg)
	if err != nil {
		t.Fatalf("renderHelmValues error: %v", err)
	}
	if !strings.Contains(output, "kafka:\n    enabled: true") {
		t.Error("expected kafka enabled for EventingBoth")
	}
	if !strings.Contains(output, "qstash:\n    enabled: true") {
		t.Error("expected qstash enabled for EventingBoth")
	}
	if !strings.Contains(output, "discord: true") {
		t.Error("expected discord: true")
	}
	if strings.Contains(output, "slack: true") {
		t.Error("unexpected slack: true")
	}
}
