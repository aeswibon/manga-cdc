package generate

import (
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type envData struct {
	HasKafka  bool
	HasQStash bool
	Notifiers []string
}

const envTemplate = `# ── Database ──
DATABASE_URL=postgres://mangacdc:mangacdc@localhost:5432/mangacdc?sslmode=disable

{{ if .HasKafka }}
# ── Eventing: Kafka ──
KAFKA_BROKERS=
KAFKA_TOPIC=mangacdc.public.chapters
KAFKA_USERNAME=
KAFKA_PASSWORD=

{{ end }}{{ if .HasQStash }}
# ── Eventing: QStash ──
QSTASH_TOKEN=
QSTASH_DESTINATION_URL=https://your-domain.com/api/webhook

{{ end }}
# ── Notifications {{ if not .Notifiers}}(none selected){{ end }}──
{{- range .Notifiers }}
{{notifierEnv .}}
{{- end }}
`

var notifierEnvMap = map[string]string{
	"discord":  "DISCORD_WEBHOOK_URL=",
	"slack":    "SLACK_WEBHOOK_URL=",
	"telegram": "TELEGRAM_BOT_TOKEN=\nTELEGRAM_CHAT_ID=",
}

func notifierEnv(name string) string {
	return notifierEnvMap[name]
}

func renderEnv(cfg Config) (string, error) {
	tmpl, err := template.New("env").Funcs(template.FuncMap{
		"notifierEnv": notifierEnv,
	}).Parse(envTemplate)
	if err != nil {
		return "", err
	}

	data := envData{
		HasKafka:  cfg.Eventing == EventingKafka || cfg.Eventing == EventingBoth,
		HasQStash: cfg.Eventing == EventingQStash || cfg.Eventing == EventingBoth,
		Notifiers: cfg.Notifiers,
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func writeEnvFile(cfg Config) error {
	content, err := renderEnv(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(RootDir, ".env.example"), []byte(content), 0644)
}
