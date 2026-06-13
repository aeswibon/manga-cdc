import type { VercelRequest, VercelResponse } from '@vercel/node';
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

export function proxySegments(req: VercelRequest): string | string[] | undefined {
  const pathname = (req.url ?? '').split('?')[0];
  const prefix = '/api/data/';
  if (pathname.startsWith(prefix)) {
    const remainder = pathname.slice(prefix.length);
    if (remainder.length > 0) {
      return remainder.split('/').filter(Boolean);
    }
  }

  const queryPath = req.query.path ?? req.query['...path'];
  return Array.isArray(queryPath) ? queryPath : typeof queryPath === 'string' ? queryPath : undefined;
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
  req: VercelRequest,
  res: VercelResponse,
  targetPath: string,
): Promise<void> {
  const baseUrl = process.env.NOTIFIER_URL?.replace(/\/$/, '');
  const apiKey = process.env.NOTIFIER_API_KEY?.trim();

  if (!baseUrl) {
    res.status(503).json({ error: 'NOTIFIER_URL is not configured' });
    return;
  }

  const method = (req.method ?? 'GET').toUpperCase();
  if (method !== 'GET' && method !== 'HEAD') {
    res.status(405).json({ error: 'Method not allowed' });
    return;
  }

  if (!isAllowedGetPath(targetPath)) {
    res.status(404).json({ error: 'Not found' });
    return;
  }

  const queryIndex = req.url?.indexOf('?') ?? -1;
  const query = queryIndex >= 0 ? req.url!.slice(queryIndex) : '';
  const targetUrl = `${baseUrl}${targetPath}${query}`;
  const isStream = targetPath === '/api/logs/stream';
  const cacheKey = isStream ? '' : `proxy:${targetPath}${query}`;

  if (cacheKey) {
    const cached = getCached(cacheKey);
    if (cached && typeof cached.body === 'string') {
      res.setHeader('Cache-Control', UPSTREAM_CACHE_CONTROL);
      res.setHeader('Content-Type', 'application/json');
      res.status(cached.status).send(cached.body);
      return;
    }
  }

  const acceptHeader =
    isStream
      ? 'text/event-stream'
      : typeof req.headers.accept === 'string' && req.headers.accept.length > 0
        ? req.headers.accept
        : 'application/json';
  const headers = new Headers({ Accept: acceptHeader });
  if (apiKey) {
    headers.set('X-Api-Key', apiKey);
  }

  const upstream = await fetch(targetUrl, {
    method,
    headers,
    signal: AbortSignal.timeout(20_000),
  });

  res.status(upstream.status);

  if (!isStream && upstream.ok) {
    const bodyText = await upstream.text();
    setCached(cacheKey, bodyText, upstream.status, UPSTREAM_CACHE_TTL_MS);
    res.setHeader('Cache-Control', UPSTREAM_CACHE_CONTROL);
    res.setHeader('Content-Type', upstream.headers.get('content-type') ?? 'application/json');
    res.send(bodyText);
    return;
  }

  upstream.headers.forEach((value, key) => {
    if (HOP_BY_HOP.has(key.toLowerCase()) || STRIP_HEADERS.has(key.toLowerCase())) {
      return;
    }
    res.setHeader(key, value);
  });

  if (!upstream.body) {
    res.end();
    return;
  }

  const reader = upstream.body.getReader();
  while (true) {
    const { done, value } = await reader.read();
    if (done) {
      break;
    }
    res.write(Buffer.from(value));
  }
  res.end();
}
