#!/usr/bin/env bash
# Render docker-compose.prod.yml .env from required environment variables.
set -euo pipefail

required=(
  SCRAPER_IMAGE
  NOTIFICATION_IMAGE
  DATABASE_URL
  KAFKA_BROKERS
  KAFKA_TOPIC
  KAFKA_USERNAME
  KAFKA_PASSWORD
  CDC_ENABLED
)

for key in "${required[@]}"; do
  if [ -z "${!key:-}" ]; then
    echo "error: missing required env var $key" >&2
    exit 1
  fi
done

python3 - <<'PY'
import os
import re
from urllib.parse import quote, unquote, urlparse, urlunparse


def parse_database_url(url: str) -> tuple[str, str, str, str]:
    url = re.sub(r"[\x00-\x1f]", "", url.strip())
    parsed = urlparse(url)
    if not parsed.hostname:
        raise ValueError("DATABASE_URL must include a hostname")
    username = parsed.username or ""
    password = unquote(parsed.password or "")
    path = parsed.path or ""
    if not path.startswith("/"):
        path = "/" + path
    host = f"{parsed.hostname}:{parsed.port}" if parsed.port else parsed.hostname
    query = f"?{parsed.query}" if parsed.query else ""
    return username, password, host, f"{path}{query}"


def clean_postgres_url(url: str) -> str:
    username, password, host, path_and_query = parse_database_url(url)
    if not username:
        return url
    encoded_password = quote(password, safe="")
    return f"postgres://{username}:{encoded_password}@{host}{path_and_query}"


username, password, host, path_and_query = parse_database_url(os.environ["DATABASE_URL"])
database_url = clean_postgres_url(os.environ["DATABASE_URL"])
spring_url = f"jdbc:postgresql://{host}{path_and_query}"

optional = {
    "DISCORD_WEBHOOK_URL": "",
    "SLACK_WEBHOOK_URL": "",
    "TELEGRAM_BOT_TOKEN": "",
    "TELEGRAM_CHAT_ID": "",
    "OBSERVABILITY_MODE": "grafana-cloud",
    "GRAFANA_CLOUD_PROMETHEUS_URL": "",
    "GRAFANA_CLOUD_PROMETHEUS_USER": "",
    "GRAFANA_CLOUD_API_KEY": "",
    "GRAFANA_CLOUD_STACK_URL": "",
    "GRAFANA_CLOUD_PROMETHEUS_DATASOURCE_UID": "",
    "ADMIN_MUTATIONS_ENABLED": "false",
    "SECURITY_REQUIRE_API_KEY": "true",
    "SECURITY_REQUIRE_WEBHOOK_AUTH": "true",
    "API_READ_KEY": "",
    "WEBHOOK_SECRET": "",
    "QSTASH_CURRENT_SIGNING_KEY": "",
    "QSTASH_NEXT_SIGNING_KEY": "",
    "ALLOWED_ORIGINS": "",
}

observability_mode = os.environ.get("OBSERVABILITY_MODE", optional["OBSERVABILITY_MODE"])
if observability_mode == "grafana-cloud":
    for key in (
        "GRAFANA_CLOUD_PROMETHEUS_URL",
        "GRAFANA_CLOUD_PROMETHEUS_USER",
        "GRAFANA_CLOUD_API_KEY",
    ):
        if not os.environ.get(key):
            raise SystemExit(f"error: {key} required when OBSERVABILITY_MODE=grafana-cloud")

lines = [
    f"SCRAPER_IMAGE={os.environ['SCRAPER_IMAGE']}",
    f"NOTIFICATION_IMAGE={os.environ['NOTIFICATION_IMAGE']}",
    f"DATABASE_URL={database_url}",
    f"SPRING_DATASOURCE_URL={spring_url}",
    f"SPRING_DATASOURCE_USERNAME={username}",
    f"SPRING_DATASOURCE_PASSWORD={password}",
    f"KAFKA_BROKERS={os.environ['KAFKA_BROKERS']}",
    f"KAFKA_TOPIC={os.environ['KAFKA_TOPIC']}",
    f"KAFKA_USERNAME={os.environ['KAFKA_USERNAME']}",
    f"KAFKA_PASSWORD={os.environ['KAFKA_PASSWORD']}",
    f"CDC_ENABLED={os.environ['CDC_ENABLED']}",
]
for key, default in optional.items():
    lines.append(f"{key}={os.environ.get(key, default)}")

print("\n".join(lines))
PY
