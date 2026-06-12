#!/usr/bin/env bash
# Parse DATABASE_URL into JDBC components for GITHUB_OUTPUT.
set -euo pipefail

: "${DATABASE_URL:?DATABASE_URL is required}"

python3 -c "
import urllib.parse, os

url = os.environ.get('DATABASE_URL', '')
if not url:
    raise SystemExit('DATABASE_URL is empty')

u = urllib.parse.urlparse(url)
jdbc_query = f'?{u.query}' if u.query else ''
jdbc_url = f'jdbc:postgresql://{u.hostname}:{u.port or 5432}{u.path}{jdbc_query}'

with open(os.environ['GITHUB_OUTPUT'], 'a') as f:
    f.write(f'db_user={u.username or \"\"}\n')
    f.write(f'db_pass={u.password or \"\"}\n')
    f.write(f'jdbc_url={jdbc_url}\n')
"
