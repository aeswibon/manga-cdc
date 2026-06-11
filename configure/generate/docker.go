package generate

import (
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type composeData struct {
	IncludeRedpanda    bool
	IncludeConnect     bool
	IncludeDebezium    bool
	IncludeCaddy       bool
	KafkaEnv           string
	QStashEnv          string
	NotificationExtras string
	IsProd             bool
}

const composeLocalTemplate = `services:
  postgres:
    image: postgres:16-alpine
    restart: unless-stopped
    ports:
      - "5432:5432"
    environment:
      POSTGRES_DB: mangacdc
      POSTGRES_USER: mangacdc
      POSTGRES_PASSWORD: mangacdc
    volumes:
      - pgdata:/var/lib/postgresql/data
      - ./db/migrations:/docker-entrypoint-initdb.d
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U mangacdc"]
      interval: 5s
      timeout: 5s
      retries: 5
      start_period: 30s

{{ if .IncludeRedpanda }}
  redpanda:
    image: docker.redpanda.com/redpandadata/redpanda:v24.2.7
    restart: unless-stopped
    ports:
      - "9092:9092"
      - "9644:9644"
    command:
      - redpanda
      - start
      - --kafka-addr internal://0.0.0.0:9092
      - --advertise-kafka-addr internal://redpanda:9092
      - --pandaproxy-addr internal://0.0.0.0:8082
      - --advertise-pandaproxy-addr internal://redpanda:8082
      - --schema-registry-addr internal://0.0.0.0:8081
      - --rpc-addr redpanda:33145
      - --advertise-rpc-addr redpanda:33145
      - --mode dev-container
      - --smp 1
      - --memory 512M
    healthcheck:
      test: ["CMD-SHELL", "rpk cluster info | grep -q leader_id"]
      interval: 10s
      timeout: 5s
      retries: 10
      start_period: 30s
{{ end }}

{{ if .IncludeConnect }}
  connect:
    image: debezium/connect:2.7.4.Final
    restart: unless-stopped
    ports:
      - "8083:8083"
    environment:
      BOOTSTRAP_SERVERS: redpanda:9092
      GROUP_ID: 1
      CONFIG_STORAGE_TOPIC: connect_configs
      OFFSET_STORAGE_TOPIC: connect_offsets
      STATUS_STORAGE_TOPIC: connect_statuses
      KEY_CONVERTER: org.apache.kafka.connect.json.JsonConverter
      VALUE_CONVERTER: org.apache.kafka.connect.json.JsonConverter
      CONNECT_KEY_CONVERTER_SCHEMAS_ENABLE: "false"
      CONNECT_VALUE_CONVERTER_SCHEMAS_ENABLE: "false"
    depends_on:
      redpanda:
        condition: service_healthy
      postgres:
        condition: service_healthy
{{ end }}

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
{{ end }}

  scraper:
    build:
      context: ./scraper
      dockerfile: Dockerfile
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      DATABASE_URL: postgres://mangacdc:mangacdc@postgres:5432/mangacdc?sslmode=disable
      SCRAPE_INTERVAL_SECONDS: "300"
      LOG_LEVEL: info
{{ .KafkaEnv }}{{ .QStashEnv }}
    ports:
      - "2112:2112"

  notification-service:
    build:
      context: ./notification-service
      dockerfile: Dockerfile
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      SPRING_DATASOURCE_URL: jdbc:postgresql://postgres:5432/mangacdc
      SPRING_DATASOURCE_USERNAME: mangacdc
      SPRING_DATASOURCE_PASSWORD: mangacdc
{{ .NotificationExtras }}

  prometheus:
    image: prom/prometheus:v2.53.0
    restart: unless-stopped
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml

  grafana:
    image: grafana/grafana:11.1.0
    restart: unless-stopped
    ports:
      - "3000:3000"
    environment:
      GF_AUTH_ANONYMOUS_ENABLED: "true"
      GF_AUTH_ANONYMOUS_ORG_ROLE: "Viewer"
    volumes:
      - grafana-data:/var/lib/grafana

volumes:
  pgdata:
  grafana-data:
{{ if .IncludeCaddy }}  caddy_data:
  caddy_config:
{{ end }}
`

const composeProdTemplate = `services:
  postgres:
    image: postgres:16-alpine
    restart: unless-stopped
    volumes:
      - pgdata:/var/lib/postgresql/data
      - ./db/migrations:/docker-entrypoint-initdb.d
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U mangacdc"]
      interval: 5s
      timeout: 5s
      retries: 5
      start_period: 30s
    environment:
      POSTGRES_DB: mangacdc
      POSTGRES_USER: mangacdc
      POSTGRES_PASSWORD: mangacdc

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
{{ end }}

  scraper:
    image: \${SCRAPER_IMAGE:?err}
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      DATABASE_URL: postgres://mangacdc:mangacdc@postgres:5432/mangacdc?sslmode=disable
      SCRAPE_INTERVAL_SECONDS: "300"
      LOG_LEVEL: info
{{ .KafkaEnv }}{{ .QStashEnv }}
    ports:
      - "2112:2112"

  notification-service:
    image: \${NOTIFICATION_IMAGE:?err}
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      SPRING_DATASOURCE_URL: jdbc:postgresql://postgres:5432/mangacdc
      SPRING_DATASOURCE_USERNAME: mangacdc
      SPRING_DATASOURCE_PASSWORD: mangacdc
{{ .NotificationExtras }}

volumes:
  pgdata:
{{ if .IncludeCaddy }}  caddy_data:
  caddy_config:
{{ end }}
`

func renderDockerCompose(cfg Config, dep string) (string, error) {
	isProd := dep == "prod-compose"

	data := composeData{
		IsProd: isProd,
	}

	switch cfg.Eventing {
	case EventingKafka:
		data.IncludeRedpanda = true
		data.IncludeConnect = true
		data.IncludeDebezium = true
		data.KafkaEnv = "      KAFKA_BROKERS: ${KAFKA_BROKERS}\n      KAFKA_TOPIC: mangacdc.public.chapters\n      KAFKA_USERNAME: ${KAFKA_USERNAME}\n      KAFKA_PASSWORD: ${KAFKA_PASSWORD}\n"
		data.NotificationExtras = "      SPRING_KAFKA_BOOTSTRAP_SERVERS: redpanda:9092\n"
	case EventingQStash:
		data.IncludeCaddy = true
		data.QStashEnv = "      QSTASH_TOKEN: ${QSTASH_TOKEN}\n      QSTASH_DESTINATION_URL: ${QSTASH_DESTINATION_URL}\n"
	case EventingBoth:
		data.IncludeRedpanda = true
		data.IncludeConnect = true
		data.IncludeDebezium = true
		data.IncludeCaddy = true
		data.KafkaEnv = "      KAFKA_BROKERS: ${KAFKA_BROKERS}\n      KAFKA_TOPIC: mangacdc.public.chapters\n      KAFKA_USERNAME: ${KAFKA_USERNAME}\n      KAFKA_PASSWORD: ${KAFKA_PASSWORD}\n"
		data.QStashEnv = "      QSTASH_TOKEN: ${QSTASH_TOKEN}\n      QSTASH_DESTINATION_URL: ${QSTASH_DESTINATION_URL}\n"
		data.NotificationExtras = "      SPRING_KAFKA_BOOTSTRAP_SERVERS: redpanda:9092\n"
	}

	var tmpl *template.Template
	var err error
	if isProd {
		tmpl, err = template.New("compose-prod").Parse(composeProdTemplate)
	} else {
		tmpl, err = template.New("compose-local").Parse(composeLocalTemplate)
	}
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func writeDockerCompose(cfg Config, dep string) error {
	content, err := renderDockerCompose(cfg, dep)
	if err != nil {
		return err
	}
	filename := "docker-compose.yml"
	if dep == "prod-compose" {
		filename = "docker-compose.prod.yml"
	}
	return os.WriteFile(filepath.Join(RootDir, filename), []byte(content), 0644)
}
