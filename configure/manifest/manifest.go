package manifest

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const CurrentVersion = 1

type Tier string

const (
	TierLocal       Tier = "local"
	TierProduction  Tier = "production"
)

type DatabaseMode string

const (
	DatabaseEmbedded DatabaseMode = "embedded"
	DatabaseExternal DatabaseMode = "external"
)

type EventingBackend string

const (
	EventingKafka  EventingBackend = "kafka"
	EventingQStash EventingBackend = "qstash"
	EventingNone   EventingBackend = "none"
)

type Manifest struct {
	Version   int             `yaml:"version"`
	Tier      Tier            `yaml:"tier"`
	Database  DatabaseConfig  `yaml:"database"`
	Eventing  EventingConfig  `yaml:"eventing"`
	Notifiers []string        `yaml:"notifiers"`
	Deploy    DeployConfig    `yaml:"deploy"`
}

type DatabaseConfig struct {
	Mode   DatabaseMode `yaml:"mode"`
	Preset string       `yaml:"preset,omitempty"`
	URL    string       `yaml:"url,omitempty"`
}

type EventingConfig struct {
	Backend EventingBackend `yaml:"backend"`
	Preset  string          `yaml:"preset,omitempty"`
	Kafka   KafkaConfig     `yaml:"kafka,omitempty"`
	QStash  QStashConfig    `yaml:"qstash,omitempty"`
}

type KafkaConfig struct {
	Brokers  string `yaml:"brokers,omitempty"`
	Topic    string `yaml:"topic,omitempty"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

type QStashConfig struct {
	Token          string `yaml:"token,omitempty"`
	DestinationURL string `yaml:"destination_url,omitempty"`
}

type DeployConfig struct {
	Targets     []string `yaml:"targets"`
	ComputeSize string   `yaml:"compute_size,omitempty"`
}

func DefaultLocal() Manifest {
	return Manifest{
		Version: CurrentVersion,
		Tier:    TierLocal,
		Database: DatabaseConfig{
			Mode: DatabaseEmbedded,
		},
		Eventing: EventingConfig{
			Backend: EventingKafka,
			Kafka: KafkaConfig{
				Topic: "mangacdc.public.chapters",
			},
		},
		Deploy: DeployConfig{
			Targets: []string{"docker-compose"},
		},
	}
}

func Load(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest: %w", err)
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("parse manifest: %w", err)
	}
	return m, nil
}

func Save(path string, m Manifest) error {
	if err := m.Validate(); err != nil {
		return err
	}
	data, err := yaml.Marshal(&m)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}

func (m Manifest) Validate() error {
	if m.Version != CurrentVersion {
		return fmt.Errorf("version: unsupported manifest version %d (want %d)", m.Version, CurrentVersion)
	}
	switch m.Tier {
	case TierLocal:
		return m.validateLocal()
	case TierProduction:
		return m.validateProduction()
	default:
		return fmt.Errorf("tier: must be %q or %q", TierLocal, TierProduction)
	}
}

func (m Manifest) validateLocal() error {
	if m.Database.Mode != DatabaseEmbedded {
		return fmt.Errorf("database.mode: local tier requires %q", DatabaseEmbedded)
	}
	if m.Eventing.Backend != EventingKafka {
		return fmt.Errorf("eventing.backend: local tier requires %q", EventingKafka)
	}
	for _, t := range m.Deploy.Targets {
		if t != "docker-compose" {
			return fmt.Errorf("deploy.targets: local tier only supports docker-compose, got %q", t)
		}
	}
	return nil
}

func (m Manifest) validateProduction() error {
	if m.Database.Mode != DatabaseExternal {
		return fmt.Errorf("database.mode: production tier requires %q", DatabaseExternal)
	}
	if err := validatePostgresURL(m.Database.URL); err != nil {
		return fmt.Errorf("database.url: %w", err)
	}
	switch m.Eventing.Backend {
	case EventingKafka:
		if strings.TrimSpace(m.Eventing.Kafka.Brokers) == "" {
			return fmt.Errorf("eventing.kafka.brokers: required when backend is kafka")
		}
		if m.Eventing.Kafka.Topic == "" {
			m.Eventing.Kafka.Topic = "mangacdc.public.chapters"
		}
	case EventingQStash:
		if strings.TrimSpace(m.Eventing.QStash.Token) == "" {
			return fmt.Errorf("eventing.qstash.token: required when backend is qstash")
		}
		if err := validateHTTPSURL(m.Eventing.QStash.DestinationURL); err != nil {
			return fmt.Errorf("eventing.qstash.destination_url: %w", err)
		}
	case EventingNone:
	default:
		return fmt.Errorf("eventing.backend: must be kafka, qstash, or none")
	}
	if len(m.Deploy.Targets) == 0 {
		return fmt.Errorf("deploy.targets: at least one target required")
	}
	for _, t := range m.Deploy.Targets {
		switch t {
		case "docker-compose-prod", "helm", "terraform":
		default:
			return fmt.Errorf("deploy.targets: unsupported production target %q", t)
		}
	}
	return nil
}

func validatePostgresURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("required for external database")
	}
	if !strings.HasPrefix(raw, "postgres://") && !strings.HasPrefix(raw, "postgresql://") {
		return fmt.Errorf("must start with postgres:// or postgresql://")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Host == "" {
		return fmt.Errorf("invalid URL: missing host")
	}
	return nil
}

func validateHTTPSURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("must use https")
	}
	return nil
}

func (m Manifest) HasDeployTarget(name string) bool {
	for _, t := range m.Deploy.Targets {
		if t == name {
			return true
		}
	}
	return false
}
