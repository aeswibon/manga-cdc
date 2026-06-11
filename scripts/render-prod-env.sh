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


def clean_url(url: str) -> str:
    url = re.sub(r"[\x00-\x1f]", "", url.strip())
    parsed = urlparse(url)
    if not parsed.username or not parsed.password:
        return url
    password = quote(unquote(parsed.password), safe="")
    netloc = f"{parsed.username}:{password}@{parsed.hostname}:{parsed.port}"
    return urlunparse((parsed.scheme, netloc, parsed.path, parsed.params, parsed.query, parsed.fragment))


def to_jdbc(database_url: str) -> str:
    fixed = clean_url(database_url)
    if fixed.startswith("jdbc:"):
        return fixed
    if fixed.startswith("postgres://"):
        return "jdbc:postgresql://" + fixed[len("postgres://") :]
    if fixed.startswith("postgresql://"):
        return "jdbc:" + fixed
    raise ValueError("DATABASE_URL must start with postgres:// or postgresql://")


database_url = clean_url(os.environ["DATABASE_URL"])
spring_url = to_jdbc(database_url)

optional = {
    "DISCORD_WEBHOOK_URL": "",
    "SLACK_WEBHOOK_URL": "",
    "TELEGRAM_BOT_TOKEN": "",
    "TELEGRAM_CHAT_ID": "",
}

lines = [
    f"SCRAPER_IMAGE={os.environ['SCRAPER_IMAGE']}",
    f"NOTIFICATION_IMAGE={os.environ['NOTIFICATION_IMAGE']}",
    f"DATABASE_URL={database_url}",
    f"SPRING_DATASOURCE_URL={spring_url}",
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
