package generate

import (
	"strings"
	"testing"

	"github.com/aeswibon/manga-cdc/configure/manifest"
)

func TestGuideLocalSections(t *testing.T) {
	m := manifest.DefaultLocal()
	m.Notifiers = []string{"discord"}
	output, err := renderGuide(m)
	if err != nil {
		t.Fatalf("renderGuide error: %v", err)
	}
	if !strings.Contains(output, "Redpanda") {
		t.Error("expected Redpanda mention for local tier")
	}
	if strings.Contains(output, "Terraform") {
		t.Error("unexpected terraform section for local tier")
	}
}

func TestGuideProductionSections(t *testing.T) {
	m := manifest.Manifest{
		Version: manifest.CurrentVersion,
		Tier:    manifest.TierProduction,
		Database: manifest.DatabaseConfig{
			Mode: manifest.DatabaseExternal,
			URL:  "postgres://user:pass@db.example.com:5432/mangacdc",
		},
		Eventing:  manifest.EventingConfig{Backend: manifest.EventingKafka, Kafka: manifest.KafkaConfig{Brokers: "kafka:9092"}},
		Notifiers: []string{"discord"},
		Deploy:    manifest.DeployConfig{Targets: []string{"docker-compose-prod", "helm"}},
	}
	output, err := renderGuide(m)
	if err != nil {
		t.Fatalf("renderGuide error: %v", err)
	}
	if !strings.Contains(output, "docker-compose.prod.yml") {
		t.Error("expected production compose instructions")
	}
	if !strings.Contains(output, "Helm") {
		t.Error("expected helm section")
	}
	if !strings.Contains(output, "future release") {
		t.Error("expected terraform deferred note")
	}
}
