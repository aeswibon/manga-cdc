package generate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aeswibon/manga-cdc/configure/manifest"
)

func TestAllProductionManifestWritesArtifacts(t *testing.T) {
	dir := t.TempDir()
	RootDir = dir

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
				Brokers:  "kafka.example.com:9092",
				Topic:    "mangacdc.public.chapters",
				Username: "kafka-user",
				Password: "kafka-pass",
			},
		},
		Notifiers: []string{"discord", "slack"},
		Deploy: manifest.DeployConfig{
			Targets: []string{"docker-compose-prod", "helm"},
		},
	}

	if err := All(m); err != nil {
		t.Fatalf("All: %v", err)
	}

	for _, rel := range []string{
		".env.example",
		"docker-compose.prod.yml",
		"helm/manga-cdc/values-override.yaml",
		"SETUP.md",
	} {
		path := filepath.Join(dir, rel)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
		if !strings.Contains(string(data), "manga-cdc") && !strings.Contains(string(data), "mangacdc") {
			t.Errorf("%s: expected project content", rel)
		}
	}

	compose, err := os.ReadFile(filepath.Join(dir, "docker-compose.prod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(compose)
	if strings.Contains(body, "postgres:") || strings.Contains(body, "redpanda:") {
		t.Error("production compose must not include postgres or redpanda")
	}
	if !strings.Contains(body, "KAFKA_BROKERS:") {
		t.Error("production compose missing kafka env")
	}
}

func TestAllQStashProductionWritesCaddyfile(t *testing.T) {
	dir := t.TempDir()
	RootDir = dir

	m := manifest.Manifest{
		Version: manifest.CurrentVersion,
		Tier:    manifest.TierProduction,
		Database: manifest.DatabaseConfig{
			Mode: manifest.DatabaseExternal,
			URL:  "postgres://user:pass@db.example.com:5432/mangacdc",
		},
		Eventing: manifest.EventingConfig{
			Backend: manifest.EventingQStash,
			Preset:  "upstash-qstash",
			QStash: manifest.QStashConfig{
				Token:          "test-token",
				DestinationURL: "https://example.com/api/webhook",
			},
		},
		Notifiers: []string{"discord"},
		Deploy:    manifest.DeployConfig{Targets: []string{"docker-compose-prod"}},
	}

	if err := All(m); err != nil {
		t.Fatalf("All: %v", err)
	}

	caddy, err := os.ReadFile(filepath.Join(dir, "Caddyfile"))
	if err != nil {
		t.Fatalf("missing Caddyfile: %v", err)
	}
	if !strings.Contains(string(caddy), "reverse_proxy notification-service:8080") {
		t.Error("Caddyfile missing reverse proxy config")
	}
}
