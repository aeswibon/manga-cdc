import type { VercelRequest, VercelResponse } from '@vercel/node';

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

function buildTargetPath(segments: string | string[] | undefined): string {
  const pathParts = Array.isArray(segments)
    ? segments
    : typeof segments === 'string' && segments.length > 0
      ? segments.split('/').filter(Boolean)
      : [];
  return `/api/${pathParts.map(encodeURIComponent).join('/')}`.replace(/\/+$/, '') || '/api';
}

function proxySegments(req: VercelRequest): string | string[] | undefined {
  const pathname = (req.url ?? '').split('?')[0];
  const prefix = '/api/notifier/';
  if (pathname.startsWith(prefix)) {
    const remainder = pathname.slice(prefix.length);
    if (remainder.length > 0) {
      return remainder.split('/').filter(Boolean);
    }
  }

  const queryPath = req.query.path ?? req.query['...path'];
  return Array.isArray(queryPath) ? queryPath : typeof queryPath === 'string' ? queryPath : undefined;
}

function isAllowedGetPath(path: string): boolean {
  if (ALLOWED_GET.has(path)) {
    return true;
  }
  if (/^\/api\/series\/[^/]+\/chapters$/.test(path)) {
    return true;
  }
  return false;
}

export default async function handler(req: VercelRequest, res: VercelResponse) {
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

  const targetPath = buildTargetPath(proxySegments(req));
  if (!isAllowedGetPath(targetPath)) {
    res.status(404).json({ error: 'Not found' });
    return;
  }

  const queryIndex = req.url?.indexOf('?') ?? -1;
  const query = queryIndex >= 0 ? req.url!.slice(queryIndex) : '';
  const targetUrl = `${baseUrl}${targetPath}${query}`;

  const headers = new Headers({ Accept: 'application/json' });
  if (apiKey) {
    headers.set('X-Api-Key', apiKey);
  }

  const upstream = await fetch(targetUrl, {
    method,
    headers,
  });

  res.status(upstream.status);
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
