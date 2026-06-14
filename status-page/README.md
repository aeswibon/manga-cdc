# manga-cdc status page (Vercel)

Public status page hosted **separately** from Cloud Run. When production is down, this page can still load and report disruption.

## Deploy

1. Import the `status-page/` directory as a new Vercel project (or `vercel --cwd status-page`).
2. Set environment variables on the **Vercel project**:

| Variable | Where | Purpose |
|----------|-------|---------|
| `KV_REST_API_URL` | Vercel | Upstash/Vercel KV REST base URL |
| `KV_REST_API_TOKEN` | Vercel | KV read token for `/api/status` |

3. Set GitHub repository secrets for the health poller (not on Vercel):

| Secret | Purpose |
|--------|---------|
| `PIPELINE_HEALTH_URL` | Notifier `/api/pipeline/health` URL |
| `API_READ_KEY` | Authenticated poll from GitHub Actions |
| `KV_REST_API_URL` | Same KV REST URL as Vercel |
| `KV_REST_API_TOKEN` | KV write token for poller sync |

4. Deploy. Optional: attach a custom domain (e.g. `status.yourdomain.com`).

## How it works

- `index.html` in `public/` polls `/api/status` every 120 seconds.
- `/api/status` (Vercel Edge) reads cached pipeline health from **Vercel KV** (`pipeline_health` key).
- **Pipeline Health Poller** (`.github/workflows/health-poller.yml`) runs every 5 minutes: fetches notifier health with `X-Api-Key`, writes JSON to KV.
- Notifier URL and API key never ship to the browser.

## Dashboard link

Set on the dashboard build:

```bash
VITE_STATUS_PAGE_URL=https://your-status-page.vercel.app
```

## Local dev

With docker compose running:

```text
http://localhost:3001
```

Compose wires `serve-local.mjs` to poll `http://notification-service:8080/api/pipeline/health` directly (no KV required locally).

Or without Docker:

```bash
cd status-page
PIPELINE_HEALTH_URL=http://localhost:8080/api/pipeline/health bun run dev
```

## Automated deploy (GitHub Actions)

After configuring `VERCEL_TOKEN`, `VERCEL_ORG_ID`, and `VERCEL_STATUS_PAGE_PROJECT_ID` (or `VERCEL_PROJECT_ID`), the **Deploy status page** job runs when the main **Deploy** workflow completes on a release. You can also run it manually from the Actions tab.

Ensure the Vercel project has `KV_REST_API_URL` and `KV_REST_API_TOKEN` set before expecting live health data.
