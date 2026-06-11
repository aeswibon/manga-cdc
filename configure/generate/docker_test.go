package generate

import (
	"strings"
	"testing"
)

func TestComposeQStashHasCaddy(t *testing.T) {
	cfg := Config{
		Eventing:    EventingQStash,
		Deployments: []string{"local-compose"},
	}
	output, err := renderDockerCompose(cfg, "local-compose")
	if err != nil {
		t.Fatalf("renderDockerCompose error: %v", err)
	}
	if !strings.Contains(output, "caddy:") {
		t.Error("expected caddy service for QStash mode")
	}
	if strings.Contains(output, "redpanda:") {
		t.Error("unexpected redpanda service for QStash-only mode")
	}
}

func TestComposeKafkaHasRedpanda(t *testing.T) {
	cfg := Config{
		Eventing:    EventingKafka,
		Deployments: []string{"local-compose"},
	}
	output, err := renderDockerCompose(cfg, "local-compose")
	if err != nil {
		t.Fatalf("renderDockerCompose error: %v", err)
	}
	if !strings.Contains(output, "redpanda:") {
		t.Error("expected redpanda service for Kafka mode")
	}
	if strings.Contains(output, "caddy:") {
		t.Error("unexpected caddy service for Kafka-only mode")
	}
}

func TestComposeBothHasAll(t *testing.T) {
	cfg := Config{
		Eventing:    EventingBoth,
		Deployments: []string{"local-compose"},
	}
	output, err := renderDockerCompose(cfg, "local-compose")
	if err != nil {
		t.Fatalf("renderDockerCompose error: %v", err)
	}
	if !strings.Contains(output, "redpanda:") {
		t.Error("expected redpanda for EventingBoth")
	}
	if !strings.Contains(output, "caddy:") {
		t.Error("expected caddy for EventingBoth")
	}
}

func TestComposeNoneOmitsBoth(t *testing.T) {
	cfg := Config{
		Eventing:    EventingNone,
		Deployments: []string{"local-compose"},
	}
	output, err := renderDockerCompose(cfg, "local-compose")
	if err != nil {
		t.Fatalf("renderDockerCompose error: %v", err)
	}
	if strings.Contains(output, "redpanda:") {
		t.Error("unexpected redpanda for EventingNone")
	}
	if strings.Contains(output, "caddy:") {
		t.Error("unexpected caddy for EventingNone")
	}
}
