package generate

import (
	"strings"
	"testing"
)

func TestEnvContainsSection(t *testing.T) {
	cfg := Config{
		Eventing:  EventingBoth,
		Notifiers: []string{"discord", "telegram"},
	}
	output, err := renderEnv(cfg)
	if err != nil {
		t.Fatalf("renderEnv error: %v", err)
	}
	if !strings.Contains(output, "KAFKA_BROKERS") {
		t.Error("expected KAFKA_BROKERS in env output for EventingBoth")
	}
	if !strings.Contains(output, "QSTASH_TOKEN") {
		t.Error("expected QSTASH_TOKEN in env output for EventingBoth")
	}
	if !strings.Contains(output, "DISCORD_WEBHOOK_URL") {
		t.Error("expected DISCORD_WEBHOOK_URL in env output")
	}
	if !strings.Contains(output, "TELEGRAM_BOT_TOKEN") {
		t.Error("expected TELEGRAM_BOT_TOKEN in env output")
	}
	if strings.Contains(output, "SLACK_WEBHOOK_URL") {
		t.Error("unexpected SLACK_WEBHOOK_URL in env output")
	}
}

func TestEnvNoneMode(t *testing.T) {
	cfg := Config{Eventing: EventingNone, Notifiers: []string{"discord"}}
	output, err := renderEnv(cfg)
	if err != nil {
		t.Fatalf("renderEnv error: %v", err)
	}
	if strings.Contains(output, "KAFKA_BROKERS") {
		t.Error("unexpected KAFKA_BROKERS for EventingNone")
	}
	if strings.Contains(output, "QSTASH_TOKEN") {
		t.Error("unexpected QSTASH_TOKEN for EventingNone")
	}
}
