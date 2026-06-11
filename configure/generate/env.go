package generate

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/aeswibon/manga-cdc/configure/manifest"
	"github.com/aeswibon/manga-cdc/configure/presets"
)

type envData struct {
	TierLabel         string
	DatabaseSection   string
	EventingSection   string
	NotifierSection   string
}

const envTemplate = `# ── Tier: {{ .TierLabel }} ──
{{ .DatabaseSection }}
{{ .EventingSection }}
# ── Notifications ──
{{ .NotifierSection }}
`

func renderEnv(m manifest.Manifest) (string, error) {
	data := envData{TierLabel: string(m.Tier)}

	switch m.Tier {
	case manifest.TierLocal:
		data.DatabaseSection = `# ── Database (embedded in Docker Compose) ──
DATABASE_URL=postgres://mangacdc:mangacdc@localhost:5432/mangacdc?sslmode=disable
SPRING_DATASOURCE_URL=jdbc:postgresql://localhost:5432/mangacdc`
		data.EventingSection = `# ── Eventing: Kafka (Redpanda in Docker Compose) ──
# Local compose wires KAFKA_BROKERS=redpanda:9092 automatically.`
	case manifest.TierProduction:
		dbHints := presets.DatabaseHints(m.Database.Preset)
		data.DatabaseSection = fmt.Sprintf(`# ── Database (external) ──
%s
DATABASE_URL=%s
SPRING_DATASOURCE_URL=`,
			commentOrDefault(dbHints.DatabaseComment, "# Paste your JDBC URL derived from DATABASE_URL"),
			placeholder(m.Database.URL))
		data.EventingSection = renderProductionEventing(m)
	default:
		return "", fmt.Errorf("env: unsupported tier %q", m.Tier)
	}

	data.NotifierSection = renderNotifierSection(m.Notifiers)

	tmpl, err := template.New("env").Parse(envTemplate)
	if err != nil {
		return "", err
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()) + "\n", nil
}

func renderProductionEventing(m manifest.Manifest) string {
	switch m.Eventing.Backend {
	case manifest.EventingKafka:
		h := presets.KafkaHints(m.Eventing.Preset)
		return fmt.Sprintf(`# ── Eventing: Kafka ──
%s
KAFKA_BROKERS=%s
KAFKA_TOPIC=%s
KAFKA_USERNAME=%s
KAFKA_PASSWORD=`,
			commentOrDefault(h.KafkaComment, "# Managed Kafka connection"),
			placeholder(m.Eventing.Kafka.Brokers),
			defaultString(m.Eventing.Kafka.Topic, "mangacdc.public.chapters"),
			placeholder(m.Eventing.Kafka.Username))
	case manifest.EventingQStash:
		h := presets.QStashHints(m.Eventing.Preset)
		return fmt.Sprintf(`# ── Eventing: QStash ──
%s
QSTASH_TOKEN=%s
QSTASH_DESTINATION_URL=%s`,
			commentOrDefault(h.QStashComment, "# Upstash QStash"),
			placeholder(m.Eventing.QStash.Token),
			placeholder(m.Eventing.QStash.DestinationURL))
	case manifest.EventingNone:
		return `# ── Eventing: none (scraper writes DB only) ──`
	default:
		return ""
	}
}

func renderNotifierSection(notifiers []string) string {
	lines := []string{}
	if len(notifiers) == 0 {
		lines = append(lines, "# (none selected)")
	}
	for _, n := range notifiers {
		switch n {
		case "discord":
			lines = append(lines, "DISCORD_WEBHOOK_URL=")
		case "slack":
			lines = append(lines, "SLACK_WEBHOOK_URL=")
		case "telegram":
			lines = append(lines, "TELEGRAM_BOT_TOKEN=", "TELEGRAM_CHAT_ID=")
		}
	}
	return strings.Join(lines, "\n")
}

func commentOrDefault(comment, fallback string) string {
	if strings.TrimSpace(comment) == "" {
		return fallback
	}
	return comment
}

func placeholder(v string) string {
	if strings.TrimSpace(v) == "" {
		return ""
	}
	return v
}

func defaultString(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func writeEnvFile(m manifest.Manifest) error {
	content, err := renderEnv(m)
	if err != nil {
		return err
	}
	return writePlainFile(".env.example", content)
}
