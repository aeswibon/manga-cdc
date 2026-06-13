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
