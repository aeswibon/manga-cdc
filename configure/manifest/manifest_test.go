package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateLocalRejectsQStash(t *testing.T) {
	m := DefaultLocal()
	m.Eventing.Backend = EventingQStash
	if err := m.Validate(); err == nil {
		t.Fatal("expected validation error for qstash on local tier")
	}
}

func TestValidateProductionRequiresDatabaseURL(t *testing.T) {
	m := Manifest{
		Version: CurrentVersion,
		Tier:    TierProduction,
		Database: DatabaseConfig{
			Mode: DatabaseExternal,
		},
		Eventing: EventingConfig{Backend: EventingKafka, Kafka: KafkaConfig{Brokers: "kafka:9092"}},
		Deploy:   DeployConfig{Targets: []string{"docker-compose-prod"}},
	}
	if err := m.Validate(); err == nil {
		t.Fatal("expected validation error for missing database URL")
	}
}

func TestValidateProductionKafkaRequiresBrokers(t *testing.T) {
	m := Manifest{
		Version: CurrentVersion,
		Tier:    TierProduction,
		Database: DatabaseConfig{
			Mode: DatabaseExternal,
			URL:  "postgres://user:pass@db.example.com:5432/mangacdc?sslmode=require",
		},
		Eventing: EventingConfig{Backend: EventingKafka},
		Deploy:   DeployConfig{Targets: []string{"helm"}},
	}
	if err := m.Validate(); err == nil {
		t.Fatal("expected validation error for missing kafka brokers")
	}
}

func TestValidateProductionQStashRequiresHTTPS(t *testing.T) {
	m := Manifest{
		Version: CurrentVersion,
		Tier:    TierProduction,
		Database: DatabaseConfig{
			Mode: DatabaseExternal,
			URL:  "postgres://user:pass@db.example.com:5432/mangacdc",
		},
		Eventing: EventingConfig{
			Backend: EventingQStash,
			QStash: QStashConfig{
				Token:          "token",
				DestinationURL: "http://insecure.example.com/api/webhook",
			},
		},
		Deploy: DeployConfig{Targets: []string{"docker-compose-prod"}},
	}
	if err := m.Validate(); err == nil {
		t.Fatal("expected validation error for non-https destination URL")
	}
}

func TestLoadSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manga-cdc.config.yaml")

	m := Manifest{
		Version: CurrentVersion,
		Tier:    TierProduction,
		Database: DatabaseConfig{
			Mode:   DatabaseExternal,
			Preset: "aiven",
			URL:    "postgres://user:pass@db.example.com:5432/mangacdc?sslmode=require",
		},
		Eventing: EventingConfig{
			Backend: EventingKafka,
			Preset:  "aiven",
			Kafka: KafkaConfig{
				Brokers:  "kafka.example.com:9092",
				Topic:    "mangacdc.public.chapters",
				Username: "user",
				Password: "pass",
			},
		},
		Notifiers: []string{"discord"},
		Deploy:    DeployConfig{Targets: []string{"docker-compose-prod", "helm"}},
	}

	if err := Save(path, m); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Tier != TierProduction {
		t.Fatalf("tier = %q, want production", loaded.Tier)
	}
	if loaded.Database.Preset != "aiven" {
		t.Fatalf("database preset = %q", loaded.Database.Preset)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("manifest file missing: %v", err)
	}
}
