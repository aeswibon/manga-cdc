package generate

import (
	"strings"
	"testing"

	"github.com/aeswibon/manga-cdc/configure/manifest"
)

func TestComposeLocalHasRedpandaNoCaddy(t *testing.T) {
	output, err := renderComposeLocal(manifest.DefaultLocal())
	if err != nil {
		t.Fatalf("renderComposeLocal error: %v", err)
	}
	if !strings.Contains(output, "redpanda:") {
		t.Error("expected redpanda service")
	}
	if strings.Contains(output, "caddy:") {
		t.Error("unexpected caddy service in local compose")
	}
	if strings.Contains(output, "connect:") {
		t.Error("unexpected debezium connect service in local compose")
	}
	if !strings.Contains(output, "KAFKA_BROKERS: redpanda:9092") {
		t.Error("expected scraper wired to redpanda")
	}
}

func TestComposeProdKafkaAppOnly(t *testing.T) {
	m := manifest.Manifest{
		Version: manifest.CurrentVersion,
		Tier:    manifest.TierProduction,
		Database: manifest.DatabaseConfig{
			Mode: manifest.DatabaseExternal,
			URL:  "postgres://user:pass@db.example.com:5432/mangacdc?sslmode=require",
		},
		Eventing: manifest.EventingConfig{
			Backend: manifest.EventingKafka,
			Kafka:   manifest.KafkaConfig{Brokers: "kafka.example.com:9092"},
		},
		Deploy: manifest.DeployConfig{Targets: []string{"docker-compose-prod"}},
	}
	output, err := renderComposeProd(m)
	if err != nil {
		t.Fatalf("renderComposeProd error: %v", err)
	}
	if strings.Contains(output, "postgres:") {
		t.Error("unexpected postgres in production compose")
	}
	if strings.Contains(output, "redpanda:") {
		t.Error("unexpected redpanda in production compose")
	}
	if !strings.Contains(output, "KAFKA_BROKERS:") {
		t.Error("expected kafka env vars")
	}
}

func TestComposeProdQStashIncludesCaddy(t *testing.T) {
	m := manifest.Manifest{
		Version: manifest.CurrentVersion,
		Tier:    manifest.TierProduction,
		Database: manifest.DatabaseConfig{
			Mode: manifest.DatabaseExternal,
			URL:  "postgres://user:pass@db.example.com:5432/mangacdc",
		},
		Eventing: manifest.EventingConfig{
			Backend: manifest.EventingQStash,
			QStash: manifest.QStashConfig{
				Token:          "token",
				DestinationURL: "https://example.com/api/webhook",
			},
		},
		Deploy: manifest.DeployConfig{Targets: []string{"docker-compose-prod"}},
	}
	output, err := renderComposeProd(m)
	if err != nil {
		t.Fatalf("renderComposeProd error: %v", err)
	}
	if !strings.Contains(output, "caddy:") {
		t.Error("expected caddy for qstash production compose")
	}
	if strings.Contains(output, "KAFKA_BROKERS:") {
		t.Error("unexpected kafka env for qstash mode")
	}
}
