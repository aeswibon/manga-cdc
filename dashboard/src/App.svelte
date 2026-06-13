<script lang="ts">
  import { onMount } from 'svelte';
  import { 
    type Series, 
    type LogEntry,
    type Chapter,
    filterSeries, 
    filterLogs,
    calculateSuccessRate,
    WATCHLIST_GITHUB_URL,
    STATUS_PAGE_URL,
    parseSourceDisplay,
    duplicateTitleKeys,
    formatRelativeTime,
    readOnSourceLabel,
    seriesStatusLabel,
    seriesStatusVariant,
    parsePipelineHealth,
    healthLabel,
    healthVariant,
    type PipelineHealth,
  } from './utils';

  const API_BASE = import.meta.env.VITE_API_URL ?? '';

  // State Management (Svelte 5 runes)
  let activeTab = $state('overview');
  let isDemoMode = $state(false);
  let apiStatus = $state('connecting');
  let isDarkMode = $state(true);

  function toggleTheme() {
    isDarkMode = !isDarkMode;
    if (isDarkMode) {
      document.documentElement.classList.remove('light');
      localStorage.setItem('theme', 'dark');
    } else {
      document.documentElement.classList.add('light');
      localStorage.setItem('theme', 'light');
    }
  }
  
  // Dashboard stats state
  let stats = $state({
    total_series: 0,
    active_series: 0,
    total_chapters: 0,
    total_logs: 0,
    successful_deliveries: 0,
    failed_deliveries: 0
  });

  let seriesList: Series[] = $state([]);
  let logList: LogEntry[] = $state([]);
  let searchQuery = $state('');
  let statusFilter = $state('ALL');

  let sseEventCount = $state(0);
  let selectedLogForModal = $state<LogEntry | null>(null);

  // Logs filters state
  let logSearchQuery = $state('');
  let logChannelFilter = $state('ALL');
  let logStatusFilter = $state('ALL');

  let expandedSeriesId = $state<string | null>(null);
  let chaptersBySeries = $state<Record<string, Chapter[]>>({});
  let chaptersLoadingId = $state<string | null>(null);

  let pipelineHealth = $state<PipelineHealth | null>(null);
  let pipelineHealthState = $state<'idle' | 'loading' | 'ok' | 'error'>('idle');
  let pipelineHealthPolledAt = $state('');

  let duplicateTitles = $derived(duplicateTitleKeys(seriesList));
  let pipelineOverallVariant = $derived(healthVariant(pipelineHealth?.status ?? 'unknown'));
  let pipelineOverallLabel = $derived(healthLabel(pipelineHealth?.status ?? 'unknown'));

  const NAV_TABS = [
    { id: 'overview', label: 'Overview', shortLabel: 'Overview', emoji: '📊' },
    { id: 'watchlist', label: 'Community Watchlist', shortLabel: 'Watchlist', emoji: '📖' },
    { id: 'logs', label: 'Activity Logs', shortLabel: 'Logs', emoji: '🔔' },
  ] as const;

  // Computed state (Svelte 5 runes)
  let currentPage = $state(1);
  const itemsPerPage = 12;

  let filteredSeries = $derived(filterSeries(seriesList, searchQuery, statusFilter));
  let totalPages = $derived(Math.max(1, Math.ceil(filteredSeries.length / itemsPerPage)));
  let paginatedSeries = $derived(filteredSeries.slice((currentPage - 1) * itemsPerPage, currentPage * itemsPerPage));

  let filteredLogs = $derived(filterLogs(logList, logSearchQuery, logChannelFilter, logStatusFilter));
  let successRate = $derived(calculateSuccessRate(stats.successful_deliveries, stats.total_logs));

  $effect(() => {
    // Reset to page 1 on search or filter change
    searchQuery;
    statusFilter;
    currentPage = 1;
  });

  $effect(() => {
    // Auto-scroll to top when page or active tab changes
    if (currentPage || activeTab) {
      window.scrollTo({ top: 0, behavior: 'smooth' });
    }
  });

  // Mock data definitions
  const MOCK_SERIES: Series[] = [
    {
      id: "1",
      sourceId: "md-1",
      title: "One Piece",
      author: "Eiichiro Oda",
      artist: "Eiichiro Oda",
      description: "Gol D. Roger, a man referred to as the King of the Pirates, is set to be executed by the World Government...",
      coverUrl: "https://mangadex.org/covers/a1c3b275-c93f-4279-a17d-2b4742e47444/92330a10-2440-410a-8bf7-4632875f10b2.jpg",
      status: "ONGOING",
      sourceUrl: "https://mangadex.org/title/a1c3b275-c93f-4279-a17d-2b4742e47444/one-piece",
      latestChapter: 1115.0,
      lastChecked: new Date().toISOString(),
      isActive: true
    },
    {
      id: "2",
      sourceId: "md-2",
      title: "Solo Leveling",
      author: "Chugong",
      artist: "DUBU (REDICE STUDIO)",
      description: "In a world where hunters must battle deadly monsters to protect mankind, Sung Jin-Woo, the weakest hunter...",
      coverUrl: "https://mangadex.org/covers/321e481a-641e-40d9-93b5-74c055272a5a/d32f418b-4b11-477d-bb62-43d92ccb7cd8.jpg",
      status: "COMPLETED",
      sourceUrl: "https://mangadex.org/title/321e481a-641e-40d9-93b5-74c055272a5a/solo-leveling",
      latestChapter: 200.0,
      lastChecked: new Date().toISOString(),
      isActive: false
    },
    {
      id: "3",
      sourceId: "as-1",
      title: "The Beginning After the End",
      author: "TurtleMe",
      artist: "Fuyuki 23",
      description: "King Grey has unrivaled strength, wealth, and prestige in a world governed by martial ability. However...",
      coverUrl: "https://mangadex.org/covers/3331828f-7c15-46a1-a672-2d12e698889a/9903b412-2439-440a-91ff-2f63812d1b09.jpg",
      status: "ONGOING",
      sourceUrl: "https://asuracomics.com/manga/the-beginning-after-the-end",
      latestChapter: 185.0,
      lastChecked: new Date().toISOString(),
      isActive: true
    }
  ];

  const MOCK_LOGS: LogEntry[] = [
    {
      id: "l1",
      chapterId: "c1",
      status: "SENT",
      channel: "discord",
      errorMessage: "",
      createdAt: new Date(Date.now() - 5 * 60000).toISOString(),
      seriesTitle: "One Piece",
      chapterNum: 1115.0,
      chapterTitle: "The Message of Void"
    },
    {
      id: "l2",
      chapterId: "c2",
      status: "SENT",
      channel: "telegram",
      errorMessage: "",
      createdAt: new Date(Date.now() - 12 * 60000).toISOString(),
      seriesTitle: "The Beginning After the End",
      chapterNum: 185.0,
      chapterTitle: "Training Arc Commences"
    },
    {
      id: "l3",
      chapterId: "c3",
      status: "FAILED",
      channel: "slack",
      errorMessage: "Webhook returned status 404 Not Found",
      createdAt: new Date(Date.now() - 60 * 60000).toISOString(),
      seriesTitle: "Solo Leveling",
      chapterNum: 200.0,
      chapterTitle: "Epilogue — The Eternal Monarch"
    },
    {
      id: "l4",
      chapterId: "c4",
      status: "SENT",
      channel: "discord",
      errorMessage: "",
      createdAt: new Date(Date.now() - 3 * 3600000).toISOString(),
      seriesTitle: "Chainsaw Man",
      chapterNum: 182.0,
      chapterTitle: "A Way Back"
    },
    {
      id: "l5",
      chapterId: "c5",
      status: "SENT",
      channel: "slack",
      errorMessage: "",
      createdAt: new Date(Date.now() - 6 * 3600000).toISOString(),
      seriesTitle: "Jujutsu Kaisen",
      chapterNum: 271.0,
      chapterTitle: "Hidden Inventory, Part 4"
    },
    {
      id: "l6",
      chapterId: "c6",
      status: "FAILED",
      channel: "telegram",
      errorMessage: "Bot token unauthorized: chat_id not found",
      createdAt: new Date(Date.now() - 9 * 3600000).toISOString(),
      seriesTitle: "Blue Lock",
      chapterNum: 298.0,
      chapterTitle: "World's Best Striker"
    },
    {
      id: "l7",
      chapterId: "c7",
      status: "SENT",
      channel: "telegram",
      errorMessage: "",
      createdAt: new Date(Date.now() - 24 * 3600000).toISOString(),
      seriesTitle: "One Piece",
      chapterNum: 1114.0,
      chapterTitle: "The Wings of Icarus"
    },
    {
      id: "l8",
      chapterId: "c8",
      status: "FAILED",
      channel: "discord",
      errorMessage: "Rate limited: retry after 30s",
      createdAt: new Date(Date.now() - 36 * 3600000).toISOString(),
      seriesTitle: "The Beginning After the End",
      chapterNum: 184.0,
      chapterTitle: "A New Horizon"
    },
  ];

  const MOCK_STATS = {
    total_series: 3,
    active_series: 2,
    total_chapters: 1500,
    total_logs: 8,
    successful_deliveries: 5,
    failed_deliveries: 3
  };

  async function toggleChapterHistory(series: Series) {
    if (expandedSeriesId === series.id) {
      expandedSeriesId = null;
      return;
    }
    expandedSeriesId = series.id;
    if (chaptersBySeries[series.id]) return;

    chaptersLoadingId = series.id;
    try {
      const res = await fetch(`${API_BASE}/api/series/${series.id}/chapters?limit=15`);
      if (!res.ok) throw new Error(`chapters fetch failed: ${res.status}`);
      const chapters = await res.json() as Chapter[];
      chaptersBySeries = { ...chaptersBySeries, [series.id]: chapters };
    } catch (err) {
      console.warn('Failed to load chapter history', err);
      chaptersBySeries = { ...chaptersBySeries, [series.id]: [] };
    } finally {
      chaptersLoadingId = null;
    }
  }

  async function fetchPipelineHealth() {
    if (isDemoMode) {
      pipelineHealth = null;
      pipelineHealthState = 'error';
      return;
    }
    try {
      pipelineHealthState = 'loading';
      const res = await fetch(`${API_BASE}/api/pipeline/health`, { cache: 'no-store' });
      if (!res.ok) throw new Error(`pipeline health ${res.status}`);
      const parsed = parsePipelineHealth(await res.json());
      if (!parsed) throw new Error('Invalid pipeline health payload');
      pipelineHealth = parsed;
      pipelineHealthPolledAt = new Date().toISOString();
      pipelineHealthState = 'ok';
    } catch (err) {
      console.warn('Pipeline health unavailable.', err);
      pipelineHealthState = 'error';
    }
  }

  async function fetchBackendData() {
    try {
      apiStatus = 'connecting';
      const statsRes = await fetch(`${API_BASE}/api/stats`);
      if (!statsRes.ok) throw new Error('API unreachable');
      stats = await statsRes.json();

      const seriesRes = await fetch(`${API_BASE}/api/series`);
      seriesList = await seriesRes.json();

      const logsRes = await fetch(`${API_BASE}/api/logs?limit=50`);
      logList = await logsRes.json();

      isDemoMode = false;
      apiStatus = 'online';
      await fetchPipelineHealth();
    } catch (err) {
      console.warn("Backend API offline.", err);
      isDemoMode = true;
      apiStatus = 'offline';
      stats = { ...MOCK_STATS };
      seriesList = [...MOCK_SERIES];
      logList = [...MOCK_LOGS];
      pipelineHealth = null;
      pipelineHealthState = 'error';
    }
  }

  onMount(() => {
    const savedTheme = localStorage.getItem('theme');
    const prefersLight = window.matchMedia('(prefers-color-scheme: light)').matches;
    if (savedTheme === 'light' || (!savedTheme && prefersLight)) {
      isDarkMode = false;
      document.documentElement.classList.add('light');
    } else {
      isDarkMode = true;
      document.documentElement.classList.remove('light');
    }

    fetchBackendData().then(() => {
      if (!isDemoMode) {
        connectSSE();
      }
    });
    const interval = setInterval(fetchBackendData, 30000);
    const pipelineHealthInterval = setInterval(fetchPipelineHealth, 60000);

    let eventSource: EventSource | null = null;

    function connectSSE() {
      if (isDemoMode) return;
      eventSource?.close();
      eventSource = new EventSource(`${API_BASE}/api/logs/stream`);

      eventSource.addEventListener('log', (event: MessageEvent) => {
        try {
          const newLog: LogEntry = JSON.parse(event.data);
          sseEventCount++;
          const idx = logList.findIndex(l => l.id === newLog.id);
          if (idx !== -1) {
            logList[idx] = newLog;
          } else {
            logList = [newLog, ...logList];
          }

          // Update stats count in-place
          stats.total_logs = logList.length;
          stats.successful_deliveries = logList.filter(l => l.status === 'SENT').length;
          stats.failed_deliveries = logList.filter(l => l.status === 'FAILED').length;
        } catch (e) {
          console.error("Failed to parse log from SSE", e);
        }
      });

      eventSource.onerror = (err) => {
        console.warn("SSE connection error. Reconnecting in 5 seconds...");
        eventSource?.close();
        setTimeout(connectSSE, 5000);
      };
    }

    return () => {
      clearInterval(interval);
      clearInterval(pipelineHealthInterval);
      eventSource?.close();
    };
  });</script>

<div class="app-shell min-h-screen flex flex-col bg-bg-primary text-gray-100 font-sans">

  <header class="app-topbar">
    <div class="app-topbar__brand">
      <img src="/logo.svg" alt="" class="app-topbar__logo" width="32" height="32" />
      <span class="app-topbar__title">Manga-CDC</span>
    </div>
    <div class="app-topbar__actions">
      {#if isDemoMode}
        <span class="app-topbar__demo">Demo</span>
      {:else}
        <div class="app-topbar__status-wrap" title={pipelineHealth ? pipelineOverallLabel : apiStatus}>
          <span
            class="app-topbar__status"
            class:app-topbar__status--ok={pipelineHealth ? pipelineOverallVariant === 'operational' : apiStatus === 'online'}
            class:app-topbar__status--warn={pipelineHealth ? pipelineOverallVariant === 'degraded' : apiStatus === 'connecting'}
            class:app-topbar__status--down={pipelineHealth ? pipelineOverallVariant === 'down' || pipelineOverallVariant === 'unknown' : apiStatus === 'offline'}
            aria-hidden="true"
          ></span>
          <span class="app-topbar__status-label hidden sm:inline">
            {#if pipelineHealth}
              {pipelineOverallLabel}
            {:else if pipelineHealthState === 'loading'}
              Checking…
            {:else}
              {apiStatus}
            {/if}
          </span>
        </div>
      {/if}
      <button
        onclick={toggleTheme}
        class="app-topbar__theme"
        aria-label="Toggle theme"
      >
        {isDarkMode ? '🌙' : '☀️'}
      </button>
    </div>
  </header>

  <!-- Main Content -->
  <main class="app-main flex-grow p-5 md:p-10 pb-[calc(6.75rem+env(safe-area-inset-bottom))] overflow-y-auto">
    <div class="max-w-6xl xl:max-w-7xl mx-auto w-full flex flex-col">
      {#if isDemoMode}
        <div class="demo-banner">
          <span>⚠️ API Connection Offline. Showing sample tracker data.</span>
          <button onclick={fetchBackendData} class="demo-banner__action">Retry Connection</button>
        </div>
      {/if}

      <header class="page-header">
        <div>
          <h1 class="page-header__title">
            {#if activeTab === 'overview'}System Overview{/if}
            {#if activeTab === 'watchlist'}Community Watchlist{/if}
            {#if activeTab === 'logs'}Notification Logs{/if}
          </h1>
          {#if activeTab === 'watchlist'}
            <p class="page-header__subtitle">Community-curated list of tracked manga series.</p>
            <p class="page-header__hint">
              To add or remove manga, edit
              <a href={WATCHLIST_GITHUB_URL} target="_blank" rel="noopener noreferrer">data/watchlist.yaml</a>
              via pull request.
            </p>
          {:else if activeTab === 'logs'}
            <p class="page-header__subtitle">Delivery history across Discord, Slack, and Telegram.</p>
          {:else}
            <p class="page-header__subtitle">Change Data Capture streaming pipeline at a glance.</p>
          {/if}
        </div>
      </header>

      <!-- OVERVIEW TAB -->
      {#if activeTab === 'overview'}
        <div class="stat-grid">
          <div class="stat-card">
            <span class="stat-card__label">Tracked Series</span>
            <span class="stat-card__value">{stats.total_series}</span>
          </div>
          <div class="stat-card">
            <span class="stat-card__label">Active Watchers</span>
            <span class="stat-card__value stat-card__value--accent">{stats.active_series}</span>
          </div>
          <div class="stat-card">
            <span class="stat-card__label">Chapters Logged</span>
            <span class="stat-card__value">{stats.total_chapters}</span>
          </div>
          <div class="stat-card">
            <span class="stat-card__label">Delivery Success</span>
            <span class="stat-card__value stat-card__value--success">{successRate}%</span>
          </div>
        </div>

        <div class="dash-panel">
          <div class="dash-panel__header">
            <h3 class="dash-panel__title">Recent Alerts</h3>
            <button
              type="button"
              onclick={() => activeTab = 'logs'}
              class="dash-panel__link cursor-pointer"
            >
              View all
            </button>
          </div>
          {#if logList.length === 0}
            <p class="dash-empty">No notifications yet.</p>
          {:else}
            <div class="md:hidden">
              {#each logList.slice(0, 5) as log}
                <article class="alert-row">
                  <div class="alert-row__top">
                    <span class="alert-row__series">{log.seriesTitle}</span>
                    <span class="alert-row__time">{new Date(log.createdAt).toLocaleTimeString()}</span>
                  </div>
                  <div class="alert-row__meta">
                    <span class="meta-badge meta-badge--{log.channel}">{log.channel}</span>
                    <span class="meta-badge meta-badge--{log.status === 'SENT' ? 'sent' : 'failed'}">{log.status}</span>
                    <span class="text-[11px] text-gray-400">Ch. {log.chapterNum}</span>
                  </div>
                </article>
              {/each}
            </div>
            <div class="dash-table-wrap hidden md:block">
              <table class="dash-table">
                <thead>
                  <tr>
                    <th>Time</th>
                    <th>Series</th>
                    <th>Chapter</th>
                    <th>Channel</th>
                    <th>Status</th>
                  </tr>
                </thead>
                <tbody>
                  {#each logList.slice(0, 5) as log}
                    <tr>
                      <td class="text-gray-400">{new Date(log.createdAt).toLocaleTimeString()}</td>
                      <td class="font-semibold text-white">{log.seriesTitle}</td>
                      <td class="text-gray-300">Ch. {log.chapterNum}</td>
                      <td>
                        <span class="meta-badge meta-badge--{log.channel}">{log.channel}</span>
                      </td>
                      <td>
                        <span class="meta-badge meta-badge--{log.status === 'SENT' ? 'sent' : 'failed'}">{log.status}</span>
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          {/if}
        </div>
      {/if}

    <!-- WATCHLIST TAB -->
    {#if activeTab === 'watchlist'}
      {#if duplicateTitles.size > 0}
        <div class="mb-6 bg-warning/10 border border-warning/30 text-warning px-4 py-3 rounded-lg text-sm">
          Duplicate titles detected in the watchlist. Keep one canonical entry per title in <a href={WATCHLIST_GITHUB_URL} target="_blank" rel="noopener noreferrer" class="underline underline-offset-2">watchlist.yaml</a>.
        </div>
      {/if}

      <div class="watchlist-toolbar">
        <label class="watchlist-search">
          <svg class="watchlist-search__icon" viewBox="0 0 20 20" fill="none" aria-hidden="true">
            <path d="M9 3.5a5.5 5.5 0 1 1 0 11 5.5 5.5 0 0 1 0-11Z" stroke="currentColor" stroke-width="1.5"/>
            <path d="m14 14 3.5 3.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
          </svg>
          <input
            class="watchlist-search__input"
            type="search"
            placeholder="Search series or author…"
            bind:value={searchQuery}
          />
        </label>
        <div class="watchlist-segments" role="group" aria-label="Filter by publication status">
          <button
            type="button"
            class="watchlist-segment"
            class:watchlist-segment--active={statusFilter === 'ALL'}
            onclick={() => statusFilter = 'ALL'}
          >All</button>
          <button
            type="button"
            class="watchlist-segment"
            class:watchlist-segment--active={statusFilter === 'ONGOING'}
            onclick={() => statusFilter = 'ONGOING'}
          >
            <span class="watchlist-segment__dot" aria-hidden="true"></span>
            Ongoing
          </button>
          <button
            type="button"
            class="watchlist-segment"
            class:watchlist-segment--active={statusFilter === 'COMPLETED'}
            onclick={() => statusFilter = 'COMPLETED'}
          >
            <svg class="watchlist-segment__icon" viewBox="0 0 12 12" fill="none" aria-hidden="true">
              <path d="M2.5 6.2 4.8 8.5 9.5 3.8" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
            Completed
          </button>
        </div>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
        {#if paginatedSeries.length === 0}
          <div class="col-span-full dash-panel dash-empty">
            {#if isDemoMode}
              Cannot load watchlist — API is offline. Use Retry Connection above.
            {:else}
              No series match your filters, or the watchlist is empty.
            {/if}
          </div>
        {:else}
        {#each paginatedSeries as series}
          {@const source = parseSourceDisplay(series.sourceId)}
          {@const isDuplicateTitle = duplicateTitles.has(series.title.trim().toLowerCase())}
          {@const statusVariant = seriesStatusVariant(series.status)}
          <div
            class="bg-bg-secondary border border-border-color rounded-xl overflow-hidden flex flex-col transition-all duration-300 {isDuplicateTitle ? 'border-warning/40' : ''}"
            class:opacity-50={!series.isActive}
          >
            <div class="h-44 bg-bg-tertiary relative overflow-hidden">
              {#if series.coverUrl}
                <img src={series.coverUrl} alt="{series.title} cover" class="w-full h-full object-cover transition-transform duration-500 hover:scale-105" />
              {:else}
                <div class="w-full h-full flex items-center justify-center text-xs text-gray-500 font-semibold">No Cover</div>
              {/if}
              <div class="cover-top-scrim" aria-hidden="true"></div>
              <span
                class="series-status-pill"
                class:series-status-pill--ongoing={statusVariant === 'ongoing'}
                class:series-status-pill--completed={statusVariant === 'completed'}
                class:series-status-pill--unknown={statusVariant === 'unknown'}
              >
                {#if statusVariant === 'ongoing'}
                  <span class="series-status-pill__dot" aria-hidden="true"></span>
                {:else if statusVariant === 'completed'}
                  <svg class="series-status-pill__icon" viewBox="0 0 12 12" fill="none" aria-hidden="true">
                    <path d="M2.5 6.2 4.8 8.5 9.5 3.8" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
                  </svg>
                {/if}
                {seriesStatusLabel(series.status)}
              </span>
              {#if isDuplicateTitle}
                <span class="absolute top-3 right-3 px-2 py-0.5 rounded text-[9px] font-bold bg-warning/90 text-black">Duplicate title</span>
              {/if}
            </div>
            
            <div class="p-5 flex flex-col gap-2 flex-grow">
              <h3 class="font-heading font-semibold text-base text-white line-clamp-1">{series.title}</h3>
              <p class="text-[11px] text-gray-400">By {series.author || 'Unknown'}</p>
              <p class="text-xs text-gray-400 leading-relaxed line-clamp-3 my-1">{series.description || 'No description provided.'}</p>
              
              <div class="flex flex-wrap items-center justify-between gap-2 border-t border-border-color/60 pt-3 mt-auto">
                <div class="flex flex-col gap-0.5">
                  <span class="text-xs font-semibold text-gray-200">Latest: Ch. {series.latestChapter}</span>
                  <span class="text-[10px] text-gray-500">Polled {formatRelativeTime(series.lastChecked)}</span>
                </div>
                <span class="px-2 py-0.5 rounded text-[9px] font-bold uppercase
                  {series.isActive ? 'bg-success/10 text-success' : 'bg-gray-500/10 text-gray-400'}"
                >{series.isActive ? 'Active' : 'Inactive'}</span>
              </div>

              <div class="flex flex-wrap gap-2 pt-2">
                {#if series.sourceUrl}
                  <a
                    href={series.sourceUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    class="inline-flex items-center justify-center px-3 py-1.5 rounded-md text-[11px] font-semibold bg-accent/15 text-accent border border-accent/30 hover:bg-accent/25 transition-colors"
                  >
                    Read on {readOnSourceLabel(source.source)}
                  </a>
                {/if}
                <button
                  type="button"
                  onclick={() => toggleChapterHistory(series)}
                  class="inline-flex items-center justify-center px-3 py-1.5 rounded-md text-[11px] font-semibold bg-bg-primary text-gray-300 border border-border-color hover:text-white transition-colors cursor-pointer"
                >
                  {expandedSeriesId === series.id ? 'Hide chapters' : 'Chapter history'}
                </button>
              </div>

              {#if expandedSeriesId === series.id}
                <div class="mt-2 border border-border-color rounded-lg bg-bg-primary/60 p-3">
                  {#if chaptersLoadingId === series.id}
                    <p class="text-[11px] text-gray-500">Loading chapters…</p>
                  {:else if (chaptersBySeries[series.id] ?? []).length === 0}
                    <p class="text-[11px] text-gray-500">No chapters logged yet.</p>
                  {:else}
                    <ul class="flex flex-col gap-1.5 max-h-40 overflow-y-auto">
                      {#each chaptersBySeries[series.id] as chapter}
                        <li class="flex items-center justify-between gap-2 text-[11px]">
                          <span class="text-gray-300 font-medium truncate">Ch. {chapter.chapterNum}{chapter.title ? ` — ${chapter.title}` : ''}</span>
                          {#if chapter.url}
                            <a href={chapter.url} target="_blank" rel="noopener noreferrer" class="text-accent shrink-0 hover:underline">Open</a>
                          {/if}
                        </li>
                      {/each}
                    </ul>
                  {/if}
                </div>
              {/if}
            </div>
          </div>
        {/each}
        {/if}
      </div>

      <!-- Pagination Controls -->
      {#if totalPages > 1}
        <div class="dash-pagination">
          <span class="dash-pagination__info">
            Showing {(currentPage - 1) * itemsPerPage + 1} to {Math.min(currentPage * itemsPerPage, filteredSeries.length)} of {filteredSeries.length} series
          </span>
          <div class="dash-pagination__controls">
            <button 
              onclick={() => currentPage = Math.max(1, currentPage - 1)} 
              disabled={currentPage === 1}
              class="dash-pagination__btn"
            >
              Previous
            </button>
            <span class="dash-pagination__page">
              Page {currentPage} of {totalPages}
            </span>
            <button 
              onclick={() => currentPage = Math.min(totalPages, currentPage + 1)} 
              disabled={currentPage === totalPages}
              class="dash-pagination__btn"
            >
              Next
            </button>
          </div>
        </div>
      {/if}
    {/if}

    <!-- LOGS TAB -->
    {#if activeTab === 'logs'}
      <div class="dash-toolbar">
        <label class="dash-search">
          <svg class="dash-search__icon" viewBox="0 0 20 20" fill="none" aria-hidden="true">
            <path d="M9 3.5a5.5 5.5 0 1 1 0 11 5.5 5.5 0 0 1 0-11Z" stroke="currentColor" stroke-width="1.5"/>
            <path d="m14 14 3.5 3.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
          </svg>
          <input
            class="dash-search__input"
            type="search"
            placeholder="Search logs (series, chapter)…"
            bind:value={logSearchQuery}
          />
        </label>
        <div class="flex flex-col sm:flex-row gap-2 sm:gap-3">
          <div class="dash-segments" role="group" aria-label="Filter by channel">
            <button
              type="button"
              class="dash-segment"
              class:dash-segment--active={logChannelFilter === 'ALL'}
              onclick={() => logChannelFilter = 'ALL'}
            >All</button>
            <button
              type="button"
              class="dash-segment"
              class:dash-segment--active={logChannelFilter === 'discord'}
              onclick={() => logChannelFilter = 'discord'}
            >Discord</button>
            <button
              type="button"
              class="dash-segment"
              class:dash-segment--active={logChannelFilter === 'slack'}
              onclick={() => logChannelFilter = 'slack'}
            >Slack</button>
            <button
              type="button"
              class="dash-segment"
              class:dash-segment--active={logChannelFilter === 'telegram'}
              onclick={() => logChannelFilter = 'telegram'}
            >Telegram</button>
          </div>
          <div class="dash-segments" role="group" aria-label="Filter by status">
            <button
              type="button"
              class="dash-segment"
              class:dash-segment--active={logStatusFilter === 'ALL'}
              onclick={() => logStatusFilter = 'ALL'}
            >All</button>
            <button
              type="button"
              class="dash-segment"
              class:dash-segment--active={logStatusFilter === 'SENT'}
              onclick={() => logStatusFilter = 'SENT'}
            >
              <svg class="dash-segment__icon" viewBox="0 0 12 12" fill="none" aria-hidden="true">
                <path d="M2.5 6.2 4.8 8.5 9.5 3.8" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
              Sent
            </button>
            <button
              type="button"
              class="dash-segment"
              class:dash-segment--active={logStatusFilter === 'FAILED'}
              onclick={() => logStatusFilter = 'FAILED'}
            >Failed</button>
          </div>
        </div>
      </div>

      <div class="dash-panel">
        {#if filteredLogs.length === 0}
          <p class="dash-empty">No logs match your filters.</p>
        {:else}
          <div class="lg:hidden">
            {#each filteredLogs as log}
              <article class="log-card">
                <div class="log-card__header">
                  <div>
                    <div class="log-card__title">{log.seriesTitle}</div>
                    <div class="log-card__chapter">Ch. {log.chapterNum}{log.chapterTitle ? ` — ${log.chapterTitle}` : ''}</div>
                  </div>
                  <div class="log-card__timestamp">
                    <div>{new Date(log.createdAt).toLocaleDateString()}</div>
                    <div>{new Date(log.createdAt).toLocaleTimeString()}</div>
                  </div>
                </div>
                <div class="alert-row__meta">
                  <span class="meta-badge meta-badge--{log.channel}">{log.channel}</span>
                  <span class="meta-badge meta-badge--{log.status === 'SENT' ? 'sent' : 'failed'}">{log.status}</span>
                </div>
                {#if log.errorMessage}
                  <p class="log-card__error" title={log.errorMessage}>{log.errorMessage}</p>
                {/if}
                <div class="log-card__footer">
                  <button
                    type="button"
                    onclick={() => selectedLogForModal = log}
                    class="log-card__inspect"
                  >
                    Inspect
                  </button>
                </div>
              </article>
            {/each}
          </div>
          <div class="dash-table-wrap hidden lg:block">
            <table class="dash-table">
              <thead>
                <tr>
                  <th>Time / Date</th>
                  <th>Manga Series</th>
                  <th>Chapter</th>
                  <th>Channel</th>
                  <th>Status</th>
                  <th>Error Details</th>
                  <th>Details</th>
                </tr>
              </thead>
              <tbody>
                {#each filteredLogs as log}
                  <tr>
                    <td>
                      <div class="text-white font-medium">{new Date(log.createdAt).toLocaleDateString()}</div>
                      <div class="text-[10px] text-gray-500 mt-0.5">{new Date(log.createdAt).toLocaleTimeString()}</div>
                    </td>
                    <td class="font-semibold text-white">{log.seriesTitle}</td>
                    <td class="text-gray-300">Ch. {log.chapterNum}</td>
                    <td>
                      <span class="meta-badge meta-badge--{log.channel}">{log.channel}</span>
                    </td>
                    <td>
                      <span class="meta-badge meta-badge--{log.status === 'SENT' ? 'sent' : 'failed'}">{log.status}</span>
                    </td>
                    <td class="text-error font-mono text-[10px] max-w-xs truncate">{log.errorMessage || '—'}</td>
                    <td>
                      <button 
                        onclick={() => selectedLogForModal = log}
                        class="log-card__inspect"
                        title="Inspect Log Payload"
                      >
                        Inspect
                      </button>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      </div>
    {/if}
    </div>
  </main>

  <!-- Floating bottom navigation (all screen sizes) -->
  <nav class="app-dock" aria-label="Main navigation">
    <div class="app-dock__bar" role="tablist">
      {#each NAV_TABS as tab}
        <button
          type="button"
          role="tab"
          aria-selected={activeTab === tab.id}
          class="app-dock__item"
          class:app-dock__item--active={activeTab === tab.id}
          onclick={() => activeTab = tab.id}
        >
          <span class="app-dock__icon" aria-hidden="true">
            {#if tab.id === 'overview'}
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
                <rect x="3" y="3" width="7" height="7" rx="1.75"/>
                <rect x="14" y="3" width="7" height="7" rx="1.75"/>
                <rect x="3" y="14" width="7" height="7" rx="1.75"/>
                <rect x="14" y="14" width="7" height="7" rx="1.75"/>
              </svg>
            {:else if tab.id === 'watchlist'}
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
                <path d="M6 4h9.5a2.5 2.5 0 0 1 2.5 2.5V20a2 2 0 0 0-2-2H6V4z"/>
                <path d="M6 12h12"/>
              </svg>
            {:else}
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
                <path d="M12 4.5a4.5 4.5 0 0 1 4.5 4.5v2.8c0 .5.2 1 .5 1.4l.9 1.2H6.1l.9-1.2c.3-.4.5-.9.5-1.4V9a4.5 4.5 0 0 1 4.5-4.5z"/>
                <path d="M10 18.5a2 2 0 0 0 4 0"/>
              </svg>
            {/if}
          </span>
          <span class="app-dock__label app-dock__label--short">{tab.shortLabel}</span>
          <span class="app-dock__label app-dock__label--full">{tab.label}</span>
        </button>
      {/each}
    </div>
  </nav>
</div>

<!-- LOG DETAILS MODAL -->
{#if selectedLogForModal}
  <div class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-end sm:items-center justify-center p-0 sm:p-4 z-50 animate-fade-in">
    <div class="modal-sheet flex flex-col gap-4">
      <div class="flex justify-between items-center pb-2 border-b border-border-color">
        <h3 class="font-heading font-semibold text-lg text-gray-100">Notification Dispatch Details</h3>
        <button onclick={() => selectedLogForModal = null} class="text-gray-400 hover:text-gray-200 text-lg cursor-pointer" aria-label="Close">✕</button>
      </div>
      <div class="flex flex-col gap-3">
        <div class="grid grid-cols-2 gap-4 bg-bg-primary p-4 rounded border border-border-color text-xs">
          <div class="flex flex-col gap-0.5">
            <span class="text-gray-500 font-semibold">Manga Series</span>
            <span class="text-gray-200 font-medium">{selectedLogForModal.seriesTitle}</span>
          </div>
          <div class="flex flex-col gap-0.5">
            <span class="text-gray-500 font-semibold">Chapter Number</span>
            <span class="text-gray-200 font-medium">Ch. {selectedLogForModal.chapterNum}</span>
          </div>
          <div class="flex flex-col gap-0.5 mt-2">
            <span class="text-gray-500 font-semibold">Target Channel</span>
            <span class="text-gray-200 font-medium uppercase">{selectedLogForModal.channel}</span>
          </div>
          <div class="flex flex-col gap-0.5 mt-2">
            <span class="text-gray-500 font-semibold">Delivery Status</span>
            <span class="font-bold uppercase {selectedLogForModal.status === 'SENT' ? 'text-success' : 'text-error'}">{selectedLogForModal.status}</span>
          </div>
        </div>
        <div class="flex flex-col gap-1.5">
          <label class="text-[10px] uppercase font-semibold text-gray-400">Timestamp</label>
          <div class="bg-bg-primary border border-border-color p-2.5 rounded text-xs text-gray-300 font-mono">
            {new Date(selectedLogForModal.createdAt).toString()}
          </div>
        </div>
        {#if selectedLogForModal.errorMessage}
          <div class="flex flex-col gap-1.5">
            <label class="text-[10px] uppercase font-semibold text-gray-400">Error Payload</label>
            <div class="bg-bg-primary border border-border-color p-3 rounded text-xs text-error font-mono overflow-x-auto whitespace-pre-wrap max-h-40">
              {selectedLogForModal.errorMessage}
            </div>
          </div>
        {/if}
        <div class="flex justify-end gap-3 mt-3">
          <button onclick={() => selectedLogForModal = null} class="px-5.5 py-2 bg-bg-tertiary border border-border-color rounded text-xs text-gray-300 hover:text-gray-100 cursor-pointer">Close</button>
        </div>
      </div>
    </div>
  </div>
{/if}

