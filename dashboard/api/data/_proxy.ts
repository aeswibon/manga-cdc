
import {
  getCached,
  setCached,
  UPSTREAM_CACHE_CONTROL,
  UPSTREAM_CACHE_TTL_MS,
} from './_upstream-cache.js';

const HOP_BY_HOP = new Set([
  'connection',
  'keep-alive',
  'proxy-authenticate',
  'proxy-authorization',
  'te',
  'trailers',
  'transfer-encoding',
  'upgrade',
  'host',
  'content-length',
]);

const STRIP_HEADERS = new Set([
  'x-api-key',
  'x-admin-key',
  'authorization',
  'x-webhook-secret',
  'upstash-signature',
]);

const ALLOWED_GET = new Set([
  '/api/stats',
  '/api/series',
  '/api/logs',
  '/api/logs/stream',
]);

export function buildTargetPath(segments: string | string[] | undefined): string {
  const pathParts = Array.isArray(segments)
    ? segments
    : typeof segments === 'string' && segments.length > 0
      ? segments.split('/').filter(Boolean)
      : [];
  return `/api/${pathParts.map(encodeURIComponent).join('/')}`.replace(/\/+$/, '') || '/api';
}

export function proxySegments(req: Request): string | string[] | undefined {
  const url = new URL(req.url);
  const pathname = url.pathname;
  const prefix = '/api/data/';
  if (pathname.startsWith(prefix)) {
    const remainder = pathname.slice(prefix.length);
    if (remainder.length > 0) {
      return remainder.split('/').filter(Boolean);
    }
  }

  const queryPath = url.searchParams.getAll('path').length > 0
    ? url.searchParams.getAll('path')
    : url.searchParams.getAll('...path');
    
  if (queryPath.length === 1) return queryPath[0];
  if (queryPath.length > 1) return queryPath;
  return undefined;
}

export function isAllowedGetPath(path: string): boolean {
  if (ALLOWED_GET.has(path)) {
    return true;
  }
  if (/^\/api\/series\/[^/]+\/chapters$/.test(path)) {
    return true;
  }
  return false;
}

export async function proxyNotifierGet(
  req: Request,
  targetPath: string,
): Promise<Response> {
  const baseUrl = process.env.NOTIFIER_URL?.replace(/\/$/, '');
  const apiKey = process.env.NOTIFIER_API_KEY?.trim();

  if (!baseUrl) {
    return Response.json({ error: 'NOTIFIER_URL is not configured' }, { status: 503 });
  }

  const method = (req.method ?? 'GET').toUpperCase();
  if (method !== 'GET' && method !== 'HEAD') {
    return Response.json({ error: 'Method not allowed' }, { status: 405 });
  }

  if (!isAllowedGetPath(targetPath)) {
    return Response.json({ error: 'Not found' }, { status: 404 });
  }

  const url = new URL(req.url);
  const query = url.search;
  const targetUrl = `${baseUrl}${targetPath}${query}`;
  const isStream = targetPath === '/api/logs/stream';
  const cacheKey = isStream ? '' : `proxy:${targetPath}${query}`;

  if (cacheKey) {
    const cached = getCached(cacheKey);
    if (cached && typeof cached.body === 'string') {
      return new Response(cached.body, {
        status: cached.status,
        headers: {
          'Cache-Control': UPSTREAM_CACHE_CONTROL,
          'Content-Type': 'application/json',
        },
      });
    }
  }

  const acceptHeader =
    isStream
      ? 'text/event-stream'
      : req.headers.get('accept') || 'application/json';
  const headers = new Headers({ Accept: acceptHeader });
  if (apiKey) {
    headers.set('X-Api-Key', apiKey);
  }

  const upstream = await fetch(targetUrl, {
    method,
    headers,
    signal: AbortSignal.timeout(20_000),
  });

  if (!isStream && upstream.ok) {
    const bodyText = await upstream.text();
    setCached(cacheKey, bodyText, upstream.status, UPSTREAM_CACHE_TTL_MS);
    return new Response(bodyText, {
      status: upstream.status,
      headers: {
        'Cache-Control': UPSTREAM_CACHE_CONTROL,
        'Content-Type': upstream.headers.get('content-type') ?? 'application/json',
      },
    });
  }

  const responseHeaders = new Headers();
  upstream.headers.forEach((value, key) => {
    if (HOP_BY_HOP.has(key.toLowerCase()) || STRIP_HEADERS.has(key.toLowerCase())) {
      return;
    }
    responseHeaders.set(key, value);
  });

  return new Response(upstream.body, {
    status: upstream.status,
    headers: responseHeaders,
  });
}
