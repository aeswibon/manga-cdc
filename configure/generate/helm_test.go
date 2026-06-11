package generate

import (
	"strings"
	"testing"

	"github.com/aeswibon/manga-cdc/configure/manifest"
)

func TestHelmValuesProductionKafka(t *testing.T) {
	m := manifest.Manifest{
		Version: manifest.CurrentVersion,
		Tier:    manifest.TierProduction,
		Database: manifest.DatabaseConfig{
			Mode: manifest.DatabaseExternal,
			URL:  "postgres://user:pass@db.example.com:5432/mangacdc",
		},
		Eventing: manifest.EventingConfig{
			Backend: manifest.EventingKafka,
			Kafka: manifest.KafkaConfig{
				Brokers: "kafka.example.com:9092",
			},
		},
		Notifiers: []string{"discord"},
		Deploy:    manifest.DeployConfig{Targets: []string{"helm"}},
	}
	output, err := renderHelmValues(m)
	if err != nil {
		t.Fatalf("renderHelmValues error: %v", err)
	}
	if !strings.Contains(output, "kafka:\n    enabled: true") {
		t.Error("expected kafka enabled")
	}
	if !strings.Contains(output, "qstash:\n    enabled: false") {
		t.Error("expected qstash disabled")
	}
	if !strings.Contains(output, "postgres: false") {
		t.Error("expected infra postgres disabled")
	}
	if !strings.Contains(output, "discord: true") {
		t.Error("expected discord enabled")
	}
}
