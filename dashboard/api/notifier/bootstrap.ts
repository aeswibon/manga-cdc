import type { VercelRequest, VercelResponse } from '@vercel/node';

const LOG_LIMIT = 20;

async function upstreamGet(baseUrl: string, apiKey: string | undefined, path: string) {
  const headers = new Headers({ Accept: 'application/json' });
  if (apiKey) {
    headers.set('X-Api-Key', apiKey);
  }

  const response = await fetch(`${baseUrl}${path}`, { headers });
  if (!response.ok) {
    throw new Error(`${path} returned HTTP ${response.status}`);
  }
  return response.json();
}

export default async function handler(req: VercelRequest, res: VercelResponse) {
  const method = (req.method ?? 'GET').toUpperCase();
  if (method !== 'GET' && method !== 'HEAD') {
    res.status(405).json({ error: 'Method not allowed' });
    return;
  }

  const baseUrl = process.env.NOTIFIER_URL?.replace(/\/$/, '');
  const apiKey = process.env.NOTIFIER_API_KEY?.trim();
  if (!baseUrl) {
    res.status(503).json({ error: 'NOTIFIER_URL is not configured' });
    return;
  }

  try {
    const [stats, series, logs] = await Promise.all([
      upstreamGet(baseUrl, apiKey, '/api/stats'),
      upstreamGet(baseUrl, apiKey, '/api/series'),
      upstreamGet(baseUrl, apiKey, `/api/logs?limit=${LOG_LIMIT}`),
    ]);

    res.setHeader('Cache-Control', 'private, max-age=15, stale-while-revalidate=30');
    res.status(200).json({ stats, series, logs });
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Upstream request failed';
    res.status(502).json({ error: message });
  }
}
