type CacheEntry = {
  body: unknown;
  status: number;
  expiresAt: number;
};

const cache = new Map<string, CacheEntry>();

export const UPSTREAM_CACHE_TTL_MS = 5 * 60 * 1000;
export const UPSTREAM_CACHE_CONTROL = 'private, max-age=300, stale-while-revalidate=600';

export function getCached(key: string): CacheEntry | null {
  const entry = cache.get(key);
  if (!entry) {
    return null;
  }
  if (Date.now() >= entry.expiresAt) {
    cache.delete(key);
    return null;
  }
  return entry;
}

export function setCached(
  key: string,
  body: unknown,
  status: number,
  ttlMs = UPSTREAM_CACHE_TTL_MS,
): void {
  cache.set(key, { body, status, expiresAt: Date.now() + ttlMs });
}
