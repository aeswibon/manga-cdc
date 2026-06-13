import { createServer } from 'node:http';
import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const port = Number(process.env.PORT ?? 3001);
const pipelineUrl = process.env.PIPELINE_HEALTH_URL ?? 'http://localhost:8080/api/pipeline/health';
const root = path.dirname(fileURLToPath(import.meta.url));
const timeoutMs = 8000;

const staticAssets = {
  '/logo.svg': { file: 'logo.svg', type: 'image/svg+xml' },
  '/favicon.svg': { file: 'logo.svg', type: 'image/svg+xml' },
  '/favicon.ico': { file: 'logo.svg', type: 'image/svg+xml' },
};

function pathname(req) {
  try {
    return new URL(req.url ?? '/', `http://${req.headers.host ?? 'localhost'}`).pathname;
  } catch {
    return req.url?.split('?')[0] ?? '/';
  }
}

async function serveStatic(res, asset) {
  const svg = await readFile(path.join(root, asset.file));
  res.writeHead(200, {
    'Content-Type': asset.type,
    'Cache-Control': 'public, max-age=86400',
  });
  res.end(svg);
}

function normalizeStatus(status) {
  const value = String(status ?? '').trim().toLowerCase();
  if (value === 'operational' || value === 'success' || value === 'up') return 'operational';
  if (value === 'degraded' || value === 'warn' || value === 'warning') return 'degraded';
  if (value === 'down' || value === 'error' || value === 'failed') return 'down';
  return 'unknown';
}

function statusLabel(status) {
  if (status === 'operational') return 'All Systems Operational';
  if (status === 'degraded') return 'Degraded Performance';
  if (status === 'down') return 'Service Disruption';
  return 'Status Unknown';
}

async function fetchStatus() {
  const started = Date.now();
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    const response = await fetch(pipelineUrl, {
      headers: { Accept: 'application/json' },
      signal: controller.signal,
      cache: 'no-store',
    });
    const latencyMs = Date.now() - started;
    if (!response.ok) {
      return {
        status: 'down',
        label: statusLabel('down'),
        checkedAt: new Date().toISOString(),
        latencyMs,
        components: [],
        error: `HTTP ${response.status}`,
      };
    }
    const health = await response.json();
    const status = normalizeStatus(health.status);
    return {
      status,
      label: statusLabel(status),
      checkedAt: new Date().toISOString(),
      latencyMs,
      sourceUpdatedAt: health.updatedAt,
      components: Array.isArray(health.components) ? health.components : [],
    };
  } catch (error) {
    return {
      status: 'down',
      label: statusLabel('down'),
      checkedAt: new Date().toISOString(),
      latencyMs: Date.now() - started,
      components: [],
      error: error instanceof Error ? error.message : 'Request failed',
    };
  } finally {
    clearTimeout(timer);
  }
}

createServer(async (req, res) => {
  const reqPath = pathname(req);
  const method = req.method ?? 'GET';

  if (reqPath === '/api/status' && (method === 'GET' || method === 'HEAD')) {
    const payload = await fetchStatus();
    res.writeHead(200, {
      'Content-Type': 'application/json',
      'Cache-Control': 'no-store',
      'Access-Control-Allow-Origin': '*',
    });
    res.end(method === 'HEAD' ? undefined : JSON.stringify(payload));
    return;
  }

  if ((reqPath === '/' || reqPath === '/index.html') && (method === 'GET' || method === 'HEAD')) {
    const html = await readFile(path.join(root, 'index.html'));
    res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
    res.end(method === 'HEAD' ? undefined : html);
    return;
  }

  const asset = staticAssets[reqPath];
  if (asset && (method === 'GET' || method === 'HEAD')) {
    try {
      if (method === 'HEAD') {
        res.writeHead(200, { 'Content-Type': asset.type });
        res.end();
        return;
      }
      await serveStatic(res, asset);
    } catch (error) {
      console.error(`failed to serve ${reqPath}:`, error);
      res.writeHead(404).end('Not found');
    }
    return;
  }

  res.writeHead(404).end('Not found');
}).listen(port, () => {
  console.log(`status page: http://localhost:${port}`);
  console.log(`checking: ${pipelineUrl}`);
});
