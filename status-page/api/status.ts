import type { VercelRequest, VercelResponse } from '@vercel/node';

type PipelineComponent = {
  name: string;
  status: string;
  detail?: string;
};

type PipelineHealth = {
  status: string;
  updatedAt: string;
  components: PipelineComponent[];
};

type StatusPayload = {
  status: 'operational' | 'degraded' | 'down' | 'offline' | 'unknown';
  label: string;
  checkedAt: string;
  latencyMs: number;
  sourceUpdatedAt?: string;
  components: PipelineComponent[];
  error?: string;
};

const TIMEOUT_MS = 8000;

function normalizeStatus(status: string | undefined): StatusPayload['status'] {
  const value = (status ?? '').trim().toLowerCase();
  if (value === 'operational' || value === 'success' || value === 'up') return 'operational';
  if (value === 'degraded' || value === 'warn' || value === 'warning') return 'degraded';
  if (value === 'down' || value === 'error' || value === 'failed') return 'down';
  return 'unknown';
}

function statusLabel(status: StatusPayload['status']): string {
  switch (status) {
    case 'operational':
      return 'All Systems Operational';
    case 'degraded':
      return 'Degraded Performance';
    case 'down':
      return 'Service Disruption';
    case 'offline':
      return 'Pipeline Offline';
    default:
      return 'Pipeline Offline';
  }
}

function offlinePayload(error: string, latencyMs = 0): StatusPayload {
  return {
    status: 'offline',
    label: 'Pipeline Offline',
    checkedAt: new Date().toISOString(),
    latencyMs,
    components: [],
    error,
  };
}

async function fetchPipelineHealth(url: string): Promise<{ health: PipelineHealth | null; latencyMs: number; error?: string }> {
  const started = Date.now();
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), TIMEOUT_MS);
  const apiKey = process.env.NOTIFIER_API_KEY?.trim();
  const headers: Record<string, string> = { Accept: 'application/json' };
  if (apiKey) {
    headers['X-Api-Key'] = apiKey;
  }

  try {
    const response = await fetch(url, {
      method: 'GET',
      headers,
      signal: controller.signal,
      cache: 'no-store',
    });
    const latencyMs = Date.now() - started;

    if (!response.ok) {
      return {
        health: null,
        latencyMs,
        error: `HTTP ${response.status}`,
      };
    }

    const health = (await response.json()) as PipelineHealth;
    return { health, latencyMs };
  } catch (error) {
    return {
      health: null,
      latencyMs: Date.now() - started,
      error: error instanceof Error ? error.message : 'Request failed',
    };
  } finally {
    clearTimeout(timer);
  }
}

function sendStatus(res: VercelResponse, payload: StatusPayload) {
  res.setHeader('Cache-Control', 's-maxage=60, stale-while-revalidate=30');
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.status(200).json(payload);
}

export default async function handler(_req: VercelRequest, res: VercelResponse) {
  const pipelineUrl = process.env.PIPELINE_HEALTH_URL?.trim();
  if (!pipelineUrl) {
    sendStatus(res, offlinePayload('PIPELINE_HEALTH_URL is not set'));
    return;
  }

  const { health, latencyMs, error } = await fetchPipelineHealth(pipelineUrl);
  if (!health) {
    sendStatus(res, offlinePayload(error ?? 'Production health endpoint unreachable', latencyMs));
    return;
  }

  const status = normalizeStatus(health.status);
  if (status === 'down' || status === 'unknown') {
    sendStatus(res, {
      status: 'offline',
      label: 'Pipeline Offline',
      checkedAt: new Date().toISOString(),
      latencyMs,
      sourceUpdatedAt: health.updatedAt,
      components: health.components ?? [],
      error: error ?? `Pipeline reported ${health.status}`,
    });
    return;
  }

  sendStatus(res, {
    status,
    label: statusLabel(status),
    checkedAt: new Date().toISOString(),
    latencyMs,
    sourceUpdatedAt: health.updatedAt,
    components: health.components ?? [],
  });
}
