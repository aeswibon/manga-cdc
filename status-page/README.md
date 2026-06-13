# manga-cdc status page (Vercel)

Public status page hosted **separately** from Cloud Run. When production is down, this page can still load and report disruption.

## Deploy

1. Import the `status-page/` directory as a new Vercel project (or `vercel --cwd status-page`).
2. Set environment variable:

| Variable | Example |
|----------|---------|
| `PIPELINE_HEALTH_URL` | `https://<your-notifier-host>/api/pipeline/health` |
| `NOTIFIER_API_KEY` | Same value as production `API_READ_KEY` (server-side only) |

3. Deploy. Optional: attach a custom domain (e.g. `status.yourdomain.com`).

## How it works

- `index.html` lives in `public/` and polls `/api/status` every 60 seconds.
- `/api/status` (Vercel serverless) fetches `PIPELINE_HEALTH_URL` from Vercel's network.
- Vercel Cron hits `/api/status` once daily (Hobby plan limit) to keep the function warm.

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

## Automated deploy (GitHub Actions)

After configuring `VERCEL_TOKEN`, `VERCEL_ORG_ID`, and `VERCEL_PROJECT_ID` repository secrets, the **Deploy status page** workflow runs automatically when the main **Deploy** workflow completes on a release. You can also run it manually from the Actions tab.

Set `PIPELINE_HEALTH_URL` in the Vercel project environment (not in GitHub) so `/api/status` can reach production.
