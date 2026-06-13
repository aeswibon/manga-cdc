#!/usr/bin/env bash
set -euo pipefail

: "${IMAGE_TAG:?}"
: "${SCRAPER_IMAGE:?}"
: "${NOTIFICATION_IMAGE:?}"
: "${DATABASE_URL:?}"
: "${JDBC_URL:?}"
: "${DB_USER:?}"
: "${DB_PASS:?}"

kafka_enabled="false"
if [ -n "${KAFKA_BROKERS:-}" ]; then
  kafka_enabled="true"
fi

discord_enabled="false"
slack_enabled="false"
telegram_enabled="false"
[ -n "${DISCORD_WEBHOOK_URL:-}" ] && discord_enabled="true"
[ -n "${SLACK_WEBHOOK_URL:-}" ] && slack_enabled="true"
[ -n "${TELEGRAM_BOT_TOKEN:-}" ] && telegram_enabled="true"

helm upgrade --install manga-cdc ./helm/manga-cdc \
  --namespace default \
  --set "images.scraper=${SCRAPER_IMAGE}:${IMAGE_TAG}" \
  --set "images.notification=${NOTIFICATION_IMAGE}:${IMAGE_TAG}" \
  --set-string "database.url=${DATABASE_URL}" \
  --set-string "database.jdbcUrl=${JDBC_URL}" \
  --set-string "database.username=${DB_USER}" \
  --set-string "database.password=${DB_PASS}" \
  --set "eventing.kafka.enabled=${kafka_enabled}" \
  --set "eventing.kafka.brokers=${KAFKA_BROKERS:-}" \
  --set "eventing.kafka.username=${KAFKA_USERNAME:-}" \
  --set "eventing.kafka.password=${KAFKA_PASSWORD:-}" \
  --set "discord.webhookUrl=${DISCORD_WEBHOOK_URL:-}" \
  --set "slack.webhookUrl=${SLACK_WEBHOOK_URL:-}" \
  --set "telegram.botToken=${TELEGRAM_BOT_TOKEN:-}" \
  --set "telegram.chatId=${TELEGRAM_CHAT_ID:-}" \
  --set "notifiers.discord=${discord_enabled}" \
  --set "notifiers.slack=${slack_enabled}" \
  --set "notifiers.telegram=${telegram_enabled}" \
  --set "security.requireApiKey=true" \
  --set "security.requireWebhookAuth=true" \
  --set "security.adminMutationsEnabled=false" \
  --set-string "security.allowedOrigins=${ALLOWED_ORIGINS:-}" \
  --set-string "security.apiReadKey=${API_READ_KEY:-}" \
  --set-string "security.webhookSecret=${WEBHOOK_SECRET:-}" \
  --set-string "security.qstashCurrentSigningKey=${QSTASH_CURRENT_SIGNING_KEY:-}" \
  --set-string "security.qstashNextSigningKey=${QSTASH_NEXT_SIGNING_KEY:-}"
