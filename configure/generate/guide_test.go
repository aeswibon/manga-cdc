package generate

import (
	"strings"
	"testing"
)

func TestGuideContainsSections(t *testing.T) {
	cfg := Config{
		Eventing:    EventingQStash,
		Notifiers:   []string{"discord"},
		Deployments: []string{"local-compose", "helm"},
	}
	output, err := renderGuide(cfg)
	if err != nil {
		t.Fatalf("renderGuide error: %v", err)
	}
	if !strings.Contains(output, "Prerequisites") {
		t.Error("expected Prerequisites section")
	}
	if !strings.Contains(output, "Environment") {
		t.Error("expected Environment section")
	}
	if !strings.Contains(output, "Docker Compose") {
		t.Error("expected Docker Compose section for local-compose")
	}
	if !strings.Contains(output, "Helm") {
		t.Error("expected Helm section")
	}
	if !strings.Contains(output, "Verification") {
		t.Error("expected Verification section")
	}
}
