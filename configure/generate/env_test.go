package generate

import (
	"strings"
	"testing"

	"github.com/aeswibon/manga-cdc/configure/manifest"
)

func TestEnvLocalOmitsManagedKafkaCreds(t *testing.T) {
	m := manifest.DefaultLocal()
	m.Notifiers = []string{"discord"}
	output, err := renderEnv(m)
	if err != nil {
		t.Fatalf("renderEnv error: %v", err)
	}
	if !strings.Contains(output, "DATABASE_URL=postgres://mangacdc:mangacdc@localhost:5432/mangacdc") {
		t.Error("expected local database URL")
	}
	if strings.Contains(output, "QSTASH_TOKEN") {
		t.Error("unexpected QSTASH_TOKEN in local env")
	}
	if !strings.Contains(output, "DISCORD_WEBHOOK_URL") {
		t.Error("expected discord notifier env")
	}
}

func TestEnvProductionKafka(t *testing.T) {
	m := manifest.Manifest{
		Version: manifest.CurrentVersion,
		Tier:    manifest.TierProduction,
		Database: manifest.DatabaseConfig{
			Mode:   manifest.DatabaseExternal,
			Preset: "aiven",
			URL:    "postgres://user:pass@db.example.com:5432/mangacdc?sslmode=require",
		},
		Eventing: manifest.EventingConfig{
			Backend: manifest.EventingKafka,
			Preset:  "aiven",
			Kafka: manifest.KafkaConfig{
				Brokers: "kafka.example.com:9092",
			},
		},
		Notifiers: []string{"discord", "telegram"},
	}
	output, err := renderEnv(m)
	if err != nil {
		t.Fatalf("renderEnv error: %v", err)
	}
	if !strings.Contains(output, "KAFKA_BROKERS=kafka.example.com:9092") {
		t.Error("expected kafka brokers in env")
	}
	if !strings.Contains(output, "Aiven") {
		t.Error("expected aiven preset hint")
	}
	if strings.Contains(output, "QSTASH_TOKEN") {
		t.Error("unexpected qstash env for kafka mode")
	}
}
