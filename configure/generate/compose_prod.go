package generate

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/aeswibon/manga-cdc/configure/manifest"
)

type composeProdData struct {
	IncludeCaddy    bool
	ScraperEnv      string
	NotificationEnv string
}

const composeProdTemplate = `services:
{{ if .IncludeCaddy }}
  caddy:
    image: caddy:alpine
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy_data:/data
      - caddy_config:/config
    depends_on:
      - notification-service

{{ end }}
  scraper:
    image: ${SCRAPER_IMAGE:?err}
    restart: unless-stopped
    environment:
      DATABASE_URL: ${DATABASE_URL:?err}
      SCRAPE_INTERVAL_SECONDS: "300"
      LOG_LEVEL: info
{{ .ScraperEnv }}
      DISCORD_WEBHOOK_URL: ${DISCORD_WEBHOOK_URL:-}
      ZERO_RESULT_ALERT_THRESHOLD: ${ZERO_RESULT_ALERT_THRESHOLD:-3}
    ports:
      - "127.0.0.1:2112:2112"

  notification-service:
    image: ${NOTIFICATION_IMAGE:?err}
    restart: unless-stopped
    environment:
      SPRING_DATASOURCE_URL: ${SPRING_DATASOURCE_URL:?err}
      SPRING_DATASOURCE_USERNAME: ${SPRING_DATASOURCE_USERNAME:?err}
      SPRING_DATASOURCE_PASSWORD: ${SPRING_DATASOURCE_PASSWORD:?err}
{{ .NotificationEnv }}
      DISCORD_WEBHOOK_URL: ${DISCORD_WEBHOOK_URL:-}
      SLACK_WEBHOOK_URL: ${SLACK_WEBHOOK_URL:-}
      TELEGRAM_BOT_TOKEN: ${TELEGRAM_BOT_TOKEN:-}
      TELEGRAM_CHAT_ID: ${TELEGRAM_CHAT_ID:-}
    ports:
      - "127.0.0.1:8080:8080"

{{ if .IncludeCaddy }}
volumes:
  caddy_data:
  caddy_config:
{{ end }}
`

const caddyfileTemplate = `:80 {
    reverse_proxy notification-service:8080
}
`

func renderComposeProd(m manifest.Manifest) (string, error) {
	if m.Tier != manifest.TierProduction {
		return "", fmt.Errorf("compose prod: tier must be %q", manifest.TierProduction)
	}

	data := composeProdData{}
	switch m.Eventing.Backend {
	case manifest.EventingKafka:
		data.ScraperEnv = strings.TrimSpace(`
      KAFKA_BROKERS: ${KAFKA_BROKERS:?err}
      KAFKA_TOPIC: ${KAFKA_TOPIC:-mangacdc.public.chapters}
      KAFKA_USERNAME: ${KAFKA_USERNAME:?err}
      KAFKA_PASSWORD: ${KAFKA_PASSWORD:?err}`)
		data.NotificationEnv = strings.TrimSpace(`
      SPRING_KAFKA_BOOTSTRAP_SERVERS: ${KAFKA_BROKERS:?err}
      SPRING_KAFKA_PROPERTIES_SASL_MECHANISM: SCRAM-SHA-256
      SPRING_KAFKA_PROPERTIES_SASL_JAAS_CONFIG: org.apache.kafka.common.security.scram.ScramLoginModule required username="${KAFKA_USERNAME}" password="${KAFKA_PASSWORD}";
      SPRING_KAFKA_PROPERTIES_SECURITY_PROTOCOL: SASL_SSL
      CDC_ENABLED: "true"`)
	case manifest.EventingQStash:
		data.IncludeCaddy = true
		data.ScraperEnv = strings.TrimSpace(`
      QSTASH_TOKEN: ${QSTASH_TOKEN:?err}
      QSTASH_DESTINATION_URL: ${QSTASH_DESTINATION_URL:?err}`)
		data.NotificationEnv = `      CDC_ENABLED: "false"`
	case manifest.EventingNone:
		data.ScraperEnv = ""
		data.NotificationEnv = `      CDC_ENABLED: "false"`
	default:
		return "", fmt.Errorf("compose prod: unsupported eventing backend %q", m.Eventing.Backend)
	}

	tmpl, err := template.New("compose-prod").Parse(composeProdTemplate)
	if err != nil {
		return "", err
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func writeComposeProd(m manifest.Manifest) error {
	content, err := renderComposeProd(m)
	if err != nil {
		return err
	}
	if err := writeGeneratedFile("docker-compose.prod.yml", content); err != nil {
		return err
	}
	if m.Eventing.Backend == manifest.EventingQStash {
		return writePlainFile("Caddyfile", caddyfileTemplate)
	}
	return nil
}
