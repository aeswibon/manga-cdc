export const config = { runtime: 'edge' };

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
const HEALTH_CACHE_TTL_MS = 2 * 60 * 1000;
const STATUS_CACHE_CONTROL = 's-maxage=120, stale-while-revalidate=300';

type CachedHealth = {
  payload: StatusPayload;
  expiresAt: number;
};

let cachedHealth: CachedHealth | null = null;

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

async function fetchPipelineHealth(kvUrl: string, kvToken: string): Promise<{ health: PipelineHealth | null; latencyMs: number; error?: string }> {
  const started = Date.now();
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), TIMEOUT_MS);

  try {
    const response = await fetch(`${kvUrl}/get/pipeline_health`, {
      method: 'GET',
      headers: { Authorization: `Bearer ${kvToken}` },
      signal: controller.signal,
    });
    const latencyMs = Date.now() - started;

    if (!response.ok) {
      return {
        health: null,
        latencyMs,
        error: `KV HTTP ${response.status}`,
      };
    }

    const json = await response.json();
    if (!json.result) {
       return { health: null, latencyMs, error: 'No health data in KV' };
    }

    const health = JSON.parse(json.result) as PipelineHealth;
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

function sendStatus(payload: StatusPayload): Response {
  return Response.json(payload, {
    status: 200,
    headers: {
      'Cache-Control': STATUS_CACHE_CONTROL,
      'Access-Control-Allow-Origin': '*',
    },
  });
}

export default async function handler(req: Request): Promise<Response> {
  const now = Date.now();
  if (cachedHealth && cachedHealth.expiresAt > now) {
    return sendStatus(cachedHealth.payload);
  }

  const kvUrl = process.env.KV_REST_API_URL?.trim();
  const kvToken = process.env.KV_REST_API_TOKEN?.trim();

  if (!kvUrl || !kvToken) {
    return sendStatus(offlinePayload('KV_REST_API_URL or KV_REST_API_TOKEN is not set'));
  }

  const { health, latencyMs, error } = await fetchPipelineHealth(kvUrl, kvToken);
  if (!health) {
    const payload = offlinePayload(error ?? 'KV health endpoint unreachable', latencyMs);
    cachedHealth = { payload, expiresAt: now + HEALTH_CACHE_TTL_MS };
    return sendStatus(payload);
  }

  const status = normalizeStatus(health.status);
  if (status === 'down' || status === 'unknown') {
    const payload: StatusPayload = {
      status: 'offline',
      label: 'Pipeline Offline',
      checkedAt: new Date().toISOString(),
      latencyMs,
      sourceUpdatedAt: health.updatedAt,
      components: health.components ?? [],
      error: error ?? `Pipeline reported ${health.status}`,
    };
    cachedHealth = { payload, expiresAt: now + HEALTH_CACHE_TTL_MS };
    return sendStatus(payload);
  }

  const payload: StatusPayload = {
    status,
    label: statusLabel(status),
    checkedAt: new Date().toISOString(),
    latencyMs,
    sourceUpdatedAt: health.updatedAt,
    components: health.components ?? [],
  };
  cachedHealth = { payload, expiresAt: now + HEALTH_CACHE_TTL_MS };
  return sendStatus(payload);
}
