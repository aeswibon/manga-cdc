# manga-cdc status page (Vercel)

Public status page hosted **separately** from Cloud Run. When production is down, this page can still load and report disruption.

## Deploy

1. Import the `status-page/` directory as a new Vercel project (or `vercel --cwd status-page`).
2. Set environment variable:

| Variable | Example |
|----------|---------|
| `PIPELINE_HEALTH_URL` | `https://manga-cdc-notifier-prod-….run.app/api/pipeline/health` |

3. Deploy. Optional: attach a custom domain (e.g. `status.yourdomain.com`).

## How it works

- `index.html` polls `/api/status` every 60 seconds.
- `/api/status` (Vercel serverless) fetches `PIPELINE_HEALTH_URL` from Vercel's network.
- Vercel Cron hits `/api/status` every 5 minutes to keep checks warm.

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

Or without Docker:

```bash
cd status-page
PIPELINE_HEALTH_URL=http://localhost:8080/api/pipeline/health npm run dev
```

## Vercel deploy
