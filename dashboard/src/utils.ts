export interface Series {
  id: string;
  sourceId: string;
  title: string;
  author: string;
  artist: string;
  description: string;
  coverUrl: string;
  status: string;
  sourceUrl: string;
  latestChapter: number;
  lastChecked: string;
  isActive: boolean;
}

export interface Chapter {
  id: string;
  seriesId: string;
  chapterNum: number;
  title: string;
  url: string;
  releaseDate: string;
  isNew: boolean;
}

export const WATCHLIST_GITHUB_URL =
  'https://github.com/aeswibon/manga-cdc/blob/master/data/watchlist.yaml';

export const STATUS_PAGE_URL =
  import.meta.env.VITE_STATUS_PAGE_URL ?? 'http://localhost:3001';

export interface PipelineComponent {
  name: string;
  status: string;
  detail?: string;
}

export interface PipelineHealth {
  status: string;
  updatedAt: string;
  components: PipelineComponent[];
}

export type HealthVariant = 'operational' | 'degraded' | 'down' | 'maintenance' | 'unknown';

export function healthVariant(status: string): HealthVariant {
  const normalized = status.trim().toLowerCase();
  if (normalized === 'success' || normalized === 'operational' || normalized === 'up') return 'operational';
  if (normalized === 'degraded' || normalized === 'warn' || normalized === 'warning') return 'degraded';
  if (normalized === 'offline' || normalized === 'error' || normalized === 'fail' || normalized === 'failed' || normalized === 'down') return 'down';
  if (normalized === 'maintenance') return 'maintenance';
  return 'unknown';
}

export function healthLabel(status: string): string {
  const normalized = status.trim().toLowerCase();
  if (normalized === 'offline') return 'Offline';
  const variant = healthVariant(status);
  switch (variant) {
    case 'operational': return 'Operational';
    case 'degraded': return 'Degraded';
    case 'down': return 'Down';
    case 'maintenance': return 'Maintenance';
    default: return status || 'Unknown';
  }
}

export function healthShortLabel(status: string): string {
  const normalized = status.trim().toLowerCase();
  if (normalized === 'offline') return 'Offline';
  const variant = healthVariant(status);
  switch (variant) {
    case 'operational': return 'OK';
    case 'degraded': return 'Warn';
    case 'down': return 'Down';
    case 'maintenance': return 'Maint';
    default: return 'Unknown';
  }
}

export interface StatusPagePayload {
  status: string;
  label: string;
  checkedAt: string;
  latencyMs: number;
  sourceUpdatedAt?: string;
  components: PipelineComponent[];
  error?: string;
}

export function parseStatusPagePayload(data: unknown): StatusPagePayload | null {
  if (!data || typeof data !== 'object') return null;
  const payload = data as Partial<StatusPagePayload>;
  if (typeof payload.status !== 'string' || typeof payload.checkedAt !== 'string') return null;
  const components = Array.isArray(payload.components) ? payload.components : [];
  return {
    status: payload.status,
    label: typeof payload.label === 'string' ? payload.label : 'Status Unknown',
    checkedAt: payload.checkedAt,
    latencyMs: typeof payload.latencyMs === 'number' ? payload.latencyMs : 0,
    sourceUpdatedAt: typeof payload.sourceUpdatedAt === 'string' ? payload.sourceUpdatedAt : undefined,
    error: typeof payload.error === 'string' ? payload.error : undefined,
    components: components
      .filter((component): component is PipelineComponent => {
        return !!component && typeof component === 'object'
          && typeof (component as PipelineComponent).name === 'string'
          && typeof (component as PipelineComponent).status === 'string';
      })
      .map((component) => ({
        name: component.name,
        status: component.status,
        detail: typeof component.detail === 'string' ? component.detail : undefined,
      })),
  };
}

export function pipelineHealthFromStatusPage(payload: StatusPagePayload): PipelineHealth {
  return {
    status: payload.status,
    updatedAt: payload.sourceUpdatedAt ?? payload.checkedAt,
    components: payload.components,
  };
}

export function parsePipelineHealth(data: unknown): PipelineHealth | null {
  if (!data || typeof data !== 'object') return null;
  const payload = data as Partial<PipelineHealth>;
  if (typeof payload.status !== 'string' || typeof payload.updatedAt !== 'string') return null;
  const components = Array.isArray(payload.components) ? payload.components : [];
  return {
    status: payload.status,
    updatedAt: payload.updatedAt,
    components: components
      .filter((component): component is PipelineComponent => {
        return !!component && typeof component === 'object'
          && typeof (component as PipelineComponent).name === 'string'
          && typeof (component as PipelineComponent).status === 'string';
      })
      .map((component) => ({
        name: component.name,
        status: component.status,
        detail: typeof component.detail === 'string' ? component.detail : undefined,
      })),
  };
}

export function parseSourceDisplay(sourceId: string): { source: string; rawId: string; shortId: string } {
  const [source, ...rest] = sourceId.split(':');
  const rawId = rest.join(':') || sourceId;
  const shortId = rawId.length > 12 ? `${rawId.slice(0, 8)}…${rawId.slice(-4)}` : rawId;
  return {
    source: source || 'unknown',
    rawId,
    shortId,
  };
}

export function duplicateTitleKeys(seriesList: Series[]): Set<string> {
  const counts = new Map<string, number>();
  for (const series of seriesList) {
    const key = series.title.trim().toLowerCase();
    if (!key) continue;
    counts.set(key, (counts.get(key) ?? 0) + 1);
  }
  const duplicates = new Set<string>();
  for (const [key, count] of counts) {
    if (count > 1) duplicates.add(key);
  }
  return duplicates;
}

export function formatRelativeTime(iso: string): string {
  if (!iso) return 'never';
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) return 'unknown';
  const diffSec = Math.round((then - Date.now()) / 1000);
  const abs = Math.abs(diffSec);
  const rtf = new Intl.RelativeTimeFormat('en', { numeric: 'auto' });

  if (abs < 60) return rtf.format(diffSec, 'second');
  const diffMin = Math.round(diffSec / 60);
  if (Math.abs(diffMin) < 60) return rtf.format(diffMin, 'minute');
  const diffHr = Math.round(diffMin / 60);
  if (Math.abs(diffHr) < 48) return rtf.format(diffHr, 'hour');
  const diffDay = Math.round(diffHr / 24);
  return rtf.format(diffDay, 'day');
}

export function readOnSourceLabel(source: string): string {
  switch (source) {
    case 'mangadex': return 'MangaDex';
    case 'mangaplus': return 'Manga Plus';
    case 'mangafire': return 'MangaFire';
    case 'asurascans': return 'Asura Scans';
    case 'mangapill': return 'MangaPill';
    case 'mangatown': return 'MangaTown';
    default: return source;
  }
}

export type SeriesStatusVariant = 'ongoing' | 'completed' | 'unknown';

export function seriesStatusVariant(status: string): SeriesStatusVariant {
  const normalized = status.trim().toUpperCase();
  if (normalized === 'ONGOING') return 'ongoing';
  if (normalized === 'COMPLETED') return 'completed';
  return 'unknown';
}

export function seriesStatusLabel(status: string): string {
  const variant = seriesStatusVariant(status);
  if (variant === 'ongoing') return 'Ongoing';
  if (variant === 'completed') return 'Completed';
  return status || 'Unknown';
}

export function calculateSuccessRate(successfulDeliveries: number, totalLogs: number): number {
  if (totalLogs <= 0) {
    return 100;
  }
  return Math.round((successfulDeliveries / totalLogs) * 100);
}

export function filterSeries(seriesList: Series[], searchQuery: string, statusFilter: string): Series[] {
  const query = searchQuery.trim().toLowerCase();
  return seriesList.filter(s => {
    const matchesSearch = s.title.toLowerCase().includes(query) ||
      (s.author && s.author.toLowerCase().includes(query)) ||
      (s.description && s.description.toLowerCase().includes(query));
    
    const matchesStatus = statusFilter === 'ALL' || s.status === statusFilter;
    return matchesSearch && matchesStatus;
  });
}

export interface LogEntry {
  id: string;
  chapterId: string;
  status: string;
  channel: string;
  errorMessage: string;
  createdAt: string;
  seriesTitle: string;
  chapterNum: number;
  chapterTitle: string;
}

export function filterLogs(
  logs: LogEntry[],
  searchQuery: string,
  channelFilter: string,
  statusFilter: string
): LogEntry[] {
  const query = searchQuery.trim().toLowerCase();
  const channel = channelFilter.trim().toLowerCase();
  const status = statusFilter.trim().toLowerCase();

  return logs.filter(log => {
    const matchesSearch = log.seriesTitle.toLowerCase().includes(query) ||
      (log.chapterTitle && log.chapterTitle.toLowerCase().includes(query));
    
    const matchesChannel = channel === 'all' || log.channel.toLowerCase() === channel;
    const matchesStatus = status === 'all' || log.status.toLowerCase() === status;

    return matchesSearch && matchesChannel && matchesStatus;
  });
}
