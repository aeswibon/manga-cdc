package generate

import (
	"strings"
	"testing"
)

func TestTerraformTfvars(t *testing.T) {
	cfg := Config{
		Eventing:  EventingKafka,
		Notifiers: []string{"discord", "slack"},
	}
	output, err := renderTerraformTfvars(cfg)
	if err != nil {
		t.Fatalf("renderTerraformTfvars error: %v", err)
	}
	if !strings.Contains(output, `eventing_type = "kafka"`) {
		t.Errorf("expected eventing_type = kafka, got:\n%s", output)
	}
	if !strings.Contains(output, `"discord"`) {
		t.Error("expected discord in notifiers list")
	}
	if !strings.Contains(output, `"slack"`) {
		t.Error("expected slack in notifiers list")
	}
}
