export const config = { runtime: 'edge' };
import {
  getCached,
  setCached,
  UPSTREAM_CACHE_CONTROL,
  UPSTREAM_CACHE_TTL_MS,
} from './_upstream-cache.js';

const OVERVIEW_LOG_LIMIT = 5;
const FULL_LOG_LIMIT = 20;

type BootstrapScope = 'overview' | 'watchlist' | 'full';

async function upstreamGet(baseUrl: string, apiKey: string | undefined, path: string) {
  const headers = new Headers({ Accept: 'application/json' });
  if (apiKey) {
    headers.set('X-Api-Key', apiKey);
  }

  const response = await fetch(`${baseUrl}${path}`, {
    headers,
    signal: AbortSignal.timeout(20_000),
  });
  if (!response.ok) {
    throw new Error(`${path} returned HTTP ${response.status}`);
  }
  return response.json();
}

function parseScope(req: Request): BootstrapScope {
  const url = new URL(req.url);
  const value = url.searchParams.get('scope');
  if (value === 'overview' || value === 'watchlist') {
    return value;
  }
  return 'full';
}

function sendJson(body: unknown, status = 200) {
  return Response.json(body, {
    status,
    headers: { 'Cache-Control': UPSTREAM_CACHE_CONTROL },
  });
}

function sendError(message: string, status = 502) {
  return Response.json({ error: message }, { status });
}

export default async function handler(req: Request) {
  const method = (req.method ?? 'GET').toUpperCase();
  if (method !== 'GET' && method !== 'HEAD') {
    return sendError('Method not allowed', 405);
  }

  const baseUrl = process.env.NOTIFIER_URL?.replace(/\/$/, '');
  const apiKey = process.env.NOTIFIER_API_KEY?.trim();
  if (!baseUrl) {
    return sendError('NOTIFIER_URL is not configured', 503);
  }

  const scope = parseScope(req);

  const cacheKey = `bootstrap:${scope}`;

  try {
    const cached = getCached(cacheKey);
    if (cached) {
      return sendJson(cached.body, cached.status);
    }

    if (scope === 'overview') {
      const [stats, logs] = await Promise.all([
        upstreamGet(baseUrl, apiKey, '/api/stats'),
        upstreamGet(baseUrl, apiKey, `/api/logs?limit=${OVERVIEW_LOG_LIMIT}`),
      ]);
      const body = { stats, logs };
      setCached(cacheKey, body, 200, UPSTREAM_CACHE_TTL_MS);
      return sendJson(body, 200);
    }

    if (scope === 'watchlist') {
      const series = await upstreamGet(baseUrl, apiKey, '/api/series');
      const body = { series };
      setCached(cacheKey, body, 200, UPSTREAM_CACHE_TTL_MS);
      return sendJson(body, 200);
    }

    const [stats, series, logs] = await Promise.all([
      upstreamGet(baseUrl, apiKey, '/api/stats'),
      upstreamGet(baseUrl, apiKey, '/api/series'),
      upstreamGet(baseUrl, apiKey, `/api/logs?limit=${FULL_LOG_LIMIT}`),
    ]);

    const body = { stats, series, logs };
    setCached(cacheKey, body, 200, UPSTREAM_CACHE_TTL_MS);
    return sendJson(body, 200);
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Upstream request failed';
    return sendError(message, 502);
  }
}
