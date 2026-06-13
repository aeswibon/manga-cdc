<script lang="ts">
  import { onMount } from 'svelte';
  import { 
    type Series, 
    type LogEntry,
    filterSeries, 
    filterLogs,
    calculateSuccessRate 
  } from './utils';

  const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';

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

  // Logs filters state
  let logSearchQuery = $state('');
  let logChannelFilter = $state('ALL');
  let logStatusFilter = $state('ALL');
  let retryingLogs = $state<Record<string, boolean>>({});

  // Computed state (Svelte 5 runes)
  let filteredSeries = $derived(filterSeries(seriesList, searchQuery, statusFilter));
  let filteredLogs = $derived(filterLogs(logList, logSearchQuery, logChannelFilter, logStatusFilter));
  let successRate = $derived(calculateSuccessRate(stats.successful_deliveries, stats.total_logs));

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
    }
  ];

  const MOCK_STATS = {
    total_series: 3,
    active_series: 2,
    total_chapters: 1500,
    total_logs: 3,
    successful_deliveries: 2,
    failed_deliveries: 1
  };

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
    } catch (err) {
      console.warn("Backend API offline. Falling back to Demo Mode.");
      isDemoMode = true;
      apiStatus = 'offline';
      stats = MOCK_STATS;
      seriesList = MOCK_SERIES;
      logList = MOCK_LOGS;
    }
  }

  async function toggleSeries(series: Series) {
    const updatedStatus = !series.isActive;
    series.isActive = updatedStatus;
    
    if (updatedStatus) {
      stats.active_series++;
    } else {
      stats.active_series--;
    }

    if (!isDemoMode) {
      try {
        const res = await fetch(`${API_BASE}/api/series/${series.id}/status?active=${updatedStatus}`, {
          method: 'PUT'
        });
        if (!res.ok) throw new Error('Failed to update status');
      } catch (err) {
        console.error("Failed to update status on server:", err);
        series.isActive = !updatedStatus;
        if (updatedStatus) {
          stats.active_series--;
        } else {
          stats.active_series++;
        }
      }
    }
  }

  async function retryNotification(logEntry: LogEntry) {
    if (retryingLogs[logEntry.id]) return;
    retryingLogs[logEntry.id] = true;

    try {
      const res = await fetch(`${API_BASE}/api/logs/${logEntry.id}/retry`, {
        method: 'POST'
      });
      if (!res.ok) throw new Error('Retry request failed');
      const updated: LogEntry = await res.json();

      // Update in logList
      const idx = logList.findIndex(l => l.id === updated.id);
      if (idx !== -1) {
        logList[idx] = updated;
      }

      // Refresh stats
      const statsRes = await fetch(`${API_BASE}/api/stats`);
      if (statsRes.ok) {
        stats = await statsRes.json();
      }
    } catch (err) {
      console.error("Failed to retry notification:", err);
      alert("Failed to retry notification. Check backend integration.");
    } finally {
      retryingLogs[logEntry.id] = false;
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

    fetchBackendData();
    const interval = setInterval(fetchBackendData, 30000);

    let eventSource: EventSource | null = null;

    function connectSSE() {
      if (isDemoMode) return;
      eventSource = new EventSource(`${API_BASE}/api/logs/stream`);

      eventSource.addEventListener('log', (event: MessageEvent) => {
        try {
          const newLog: LogEntry = JSON.parse(event.data);
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

    connectSSE();

    return () => {
      clearInterval(interval);
      eventSource?.close();
    };
  });</script>

<div class="min-h-screen flex flex-col md:flex-row bg-bg-primary text-gray-100 font-sans">
  
  <!-- Sidebar -->
  <aside class="w-full md:w-64 bg-bg-secondary border-b md:border-b-0 md:border-r border-border-color p-6 flex flex-col justify-between gap-6">
    <div class="flex flex-col gap-6 w-full">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="w-8 h-8 rounded-lg bg-gradient-to-tr from-accent to-amber-600 shadow-[0_0_15px_rgba(139,92,246,0.3)]"></div>
          <span class="font-heading font-semibold text-lg tracking-wide text-gray-100">Manga-CDC</span>
        </div>
        <!-- Theme Toggle for Mobile -->
        <button 
          onclick={toggleTheme} 
          class="md:hidden p-1.5 rounded-lg border border-border-color hover:bg-bg-tertiary transition-colors cursor-pointer text-gray-400"
          aria-label="Toggle theme"
        >
          {isDarkMode ? '🌙' : '☀️'}
        </button>
      </div>
      
      <nav class="flex flex-row md:flex-col gap-2 overflow-x-auto md:overflow-visible pb-2 md:pb-0">
        <button 
          class="flex items-center gap-3 px-4 py-2.5 rounded-lg text-sm transition-all whitespace-nowrap cursor-pointer hover:bg-bg-tertiary hover:text-gray-50"
          class:bg-bg-tertiary={activeTab === 'overview'}
          class:text-gray-50={activeTab === 'overview'}
          class:text-gray-400={activeTab !== 'overview'}
          onclick={() => activeTab = 'overview'}
        >
          <span>📊</span> Overview
        </button>
        <button 
          class="flex items-center gap-3 px-4 py-2.5 rounded-lg text-sm transition-all whitespace-nowrap cursor-pointer hover:bg-bg-tertiary hover:text-gray-50"
          class:bg-bg-tertiary={activeTab === 'watchlist'}
          class:text-gray-50={activeTab === 'watchlist'}
          class:text-gray-400={activeTab !== 'watchlist'}
          onclick={() => activeTab = 'watchlist'}
        >
          <span>📖</span> Watchlist
        </button>
        <button 
          class="flex items-center gap-3 px-4 py-2.5 rounded-lg text-sm transition-all whitespace-nowrap cursor-pointer hover:bg-bg-tertiary hover:text-gray-50"
          class:bg-bg-tertiary={activeTab === 'logs'}
          class:text-gray-50={activeTab === 'logs'}
          class:text-gray-400={activeTab !== 'logs'}
          onclick={() => activeTab = 'logs'}
        >
          <span>🔔</span> Activity Logs
        </button>
      </nav>
    </div>
    
    <div class="hidden md:flex flex-col gap-3 pt-4 border-t border-border-color">
      <!-- Theme Toggle for Desktop -->
      <button 
        onclick={toggleTheme} 
        class="flex items-center justify-between w-full px-3 py-2 rounded-lg text-xs font-medium border border-border-color hover:bg-bg-tertiary transition-colors cursor-pointer text-gray-300 hover:text-gray-50 animate-fade-in"
      >
        <span>Theme</span>
        <span>{isDarkMode ? '🌙 Dark' : '☀️ Light'}</span>
      </button>

      <div class="flex items-center gap-2.5">
        <span 
          class="w-2.5 h-2.5 rounded-full" 
          class:bg-success={apiStatus === 'online'} 
          class:bg-warning={apiStatus === 'connecting'} 
          class:bg-error={apiStatus === 'offline'}
        ></span>
        <span class="text-xs text-gray-400">Backend: {apiStatus}</span>
      </div>
      {#if isDemoMode}
        <span class="text-[10px] text-warning bg-warning/10 px-2 py-0.5 rounded border border-warning/20 self-start font-semibold">Demo Mode</span>
      {/if}
    </div>
  </aside>

  <!-- Main Content -->
  <main class="flex-grow p-6 md:p-10 overflow-y-auto">
    <div class="max-w-6xl xl:max-w-7xl mx-auto w-full flex flex-col">
      {#if isDemoMode}
        <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 bg-warning/10 border border-warning/30 text-warning px-5 py-3.5 rounded-lg text-sm mb-6">
          <span>⚠️ API Connection Offline. Showing sample tracker data.</span>
          <button onclick={fetchBackendData} class="bg-warning text-black font-semibold text-xs px-3.5 py-1.5 rounded-md hover:opacity-90 transition-opacity cursor-pointer">Retry Connection</button>
        </div>
      {/if}

      <header class="mb-8">
        <h1 class="font-heading font-bold text-3xl tracking-tight mb-1">
          {#if activeTab === 'overview'}System Overview{/if}
          {#if activeTab === 'watchlist'}Manga Watchlist{/if}
          {#if activeTab === 'logs'}Notification Logs{/if}
        </h1>
        <span class="text-xs text-gray-400">Change Data Capture Streaming Pipeline</span>
      </header>

      <!-- OVERVIEW TAB -->
      {#if activeTab === 'overview'}
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5 mb-8">
          <div class="bg-bg-secondary border border-border-color p-6 rounded-xl flex flex-col gap-2 hover:-translate-y-0.5 transition-transform duration-200">
            <span class="text-xs font-semibold text-gray-400 uppercase tracking-wider">Tracked Series</span>
            <span class="font-heading font-bold text-3xl">{stats.total_series}</span>
          </div>
          <div class="bg-bg-secondary border border-border-color p-6 rounded-xl flex flex-col gap-2 hover:-translate-y-0.5 transition-transform duration-200">
            <span class="text-xs font-semibold text-gray-400 uppercase tracking-wider">Active Watchers</span>
            <span class="font-heading font-bold text-3xl text-accent">{stats.active_series}</span>
          </div>
          <div class="bg-bg-secondary border border-border-color p-6 rounded-xl flex flex-col gap-2 hover:-translate-y-0.5 transition-transform duration-200">
            <span class="text-xs font-semibold text-gray-400 uppercase tracking-wider">Chapters Logged</span>
            <span class="font-heading font-bold text-3xl">{stats.total_chapters}</span>
          </div>
          <div class="bg-bg-secondary border border-border-color p-6 rounded-xl flex flex-col gap-2 hover:-translate-y-0.5 transition-transform duration-200">
            <span class="text-xs font-semibold text-gray-400 uppercase tracking-wider">Delivery Success</span>
            <span class="font-heading font-bold text-3xl text-success">{successRate}%</span>
          </div>
        </div>

        <div class="bg-bg-secondary border border-border-color p-6 rounded-xl mb-8">
          <h3 class="font-heading font-semibold text-lg mb-1">CDC Event Flow</h3>
          <p class="text-xs text-gray-400 mb-6">Visual pipeline representing PostgreSQL logs stream to Slack, Discord, and Telegram hooks.</p>
          <div class="flex flex-col lg:flex-row items-center justify-between gap-4 bg-bg-primary p-5 rounded-lg border border-border-color">
            <div class="flex flex-col items-center gap-1.5 p-3 rounded-lg bg-bg-tertiary border border-accent/20 shadow-[0_0_10px_rgba(139,92,246,0.1)] w-full lg:w-44 text-center">
              <span class="text-2xl">🕸️</span>
              <span class="text-xs font-medium">Scraper (Go)</span>
            </div>
            <span class="text-gray-600 hidden lg:inline">➔</span>
            <div class="flex flex-col items-center gap-1.5 p-3 rounded-lg bg-bg-tertiary border border-accent/20 shadow-[0_0_10px_rgba(139,92,246,0.1)] w-full lg:w-44 text-center">
              <span class="text-2xl">🐘</span>
              <span class="text-xs font-medium">Postgres WAL</span>
            </div>
            <span class="text-gray-600 hidden lg:inline">➔</span>
            <div class="flex flex-col items-center gap-1.5 p-3 rounded-lg bg-bg-tertiary border border-accent/20 shadow-[0_0_10px_rgba(139,92,246,0.1)] w-full lg:w-44 text-center">
              <span class="text-2xl">⚡</span>
              <span class="text-xs font-medium">Kafka / QStash</span>
            </div>
            <span class="text-gray-600 hidden lg:inline">➔</span>
            <div class="flex flex-col items-center gap-1.5 p-3 rounded-lg bg-bg-tertiary border border-accent/20 shadow-[0_0_10px_rgba(139,92,246,0.1)] w-full lg:w-44 text-center">
              <span class="text-2xl">🚀</span>
              <span class="text-xs font-medium">Notifier (Java)</span>
            </div>
          </div>
        </div>

        <div class="bg-bg-secondary border border-border-color p-6 rounded-xl">
          <h3 class="font-heading font-semibold text-lg mb-4">Recent Alerts</h3>
          <div class="overflow-x-auto">
            <table class="w-full text-left text-xs border-collapse">
              <thead>
                <tr class="border-b border-border-color">
                  <th class="pb-3 text-gray-400 font-semibold uppercase tracking-wider">Time</th>
                  <th class="pb-3 text-gray-400 font-semibold uppercase tracking-wider">Series</th>
                  <th class="pb-3 text-gray-400 font-semibold uppercase tracking-wider">Chapter</th>
                  <th class="pb-3 text-gray-400 font-semibold uppercase tracking-wider">Channel</th>
                  <th class="pb-3 text-gray-400 font-semibold uppercase tracking-wider">Status</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-border-color/50">
                {#each logList.slice(0, 5) as log}
                  <tr>
                    <td class="py-3.5 text-gray-400">{new Date(log.createdAt).toLocaleTimeString()}</td>
                    <td class="py-3.5 font-semibold text-white">{log.seriesTitle}</td>
                    <td class="py-3.5 text-gray-300">Ch. {log.chapterNum}</td>
                    <td class="py-3.5">
                      <span class="px-2.5 py-0.5 rounded text-[10px] font-bold uppercase 
                        {log.channel === 'discord' ? 'bg-blue-500/10 text-blue-400' : ''}
                        {log.channel === 'slack' ? 'bg-amber-500/10 text-amber-400' : ''}
                        {log.channel === 'telegram' ? 'bg-sky-500/10 text-sky-400' : ''}"
                      >{log.channel}</span>
                    </td>
                    <td class="py-3.5">
                      <span class="px-2.5 py-0.5 rounded text-[10px] font-bold
                        {log.status === 'SENT' ? 'bg-success/10 text-success' : ''}
                        {log.status === 'FAILED' ? 'bg-error/10 text-error' : ''}"
                      >{log.status}</span>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        </div>
      {/if}

    <!-- WATCHLIST TAB -->
    {#if activeTab === 'watchlist'}
      <div class="flex flex-col sm:flex-row gap-4 mb-8">
        <input 
          type="text" 
          placeholder="Filter series title..." 
          bind:value={searchQuery} 
          class="flex-grow bg-bg-secondary border border-border-color text-sm text-gray-200 px-4 py-3 rounded-lg focus:outline-none focus:border-accent"
        />
        <select bind:value={statusFilter} class="bg-bg-secondary border border-border-color text-sm text-gray-200 px-4 py-3 rounded-lg focus:outline-none cursor-pointer">
          <option value="ALL">All Statuses</option>
          <option value="ONGOING">Ongoing</option>
          <option value="COMPLETED">Completed</option>
        </select>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
        {#each filteredSeries as series}
          <div class="bg-bg-secondary border border-border-color rounded-xl overflow-hidden flex flex-col transition-all duration-300" class:opacity-50={!series.isActive} class:border-transparent={!series.isActive}>
            <div class="h-44 bg-bg-tertiary relative overflow-hidden">
              {#if series.coverUrl}
                <img src={series.coverUrl} alt="{series.title} cover" class="w-full h-full object-cover transition-transform duration-500 hover:scale-105" />
              {:else}
                <div class="w-full h-full flex items-center justify-center text-xs text-gray-500 font-semibold">No Cover</div>
              {/if}
              <span class="absolute top-3 left-3 px-2 py-0.5 rounded text-[9px] font-bold text-black" class:bg-success={series.status === 'ONGOING'} class:bg-accent={series.status === 'COMPLETED'} class:text-white={series.status === 'COMPLETED'}>{series.status}</span>
            </div>
            
            <div class="p-5 flex flex-col gap-2 flex-grow">
              <h3 class="font-heading font-semibold text-base text-white line-clamp-1">{series.title}</h3>
              <p class="text-[11px] text-gray-400">By {series.author || 'Unknown'}</p>
              <p class="text-xs text-gray-400 leading-relaxed line-clamp-3 my-2">{series.description || 'No description provided.'}</p>
              
              <div class="flex justify-between items-center border-t border-border-color/60 pt-4 mt-auto">
                <span class="text-xs font-semibold text-gray-200">Latest: Ch. {series.latestChapter}</span>
                <label class="relative inline-block w-11 h-6 cursor-pointer">
                  <input 
                    type="checkbox" 
                    class="sr-only peer"
                    checked={series.isActive} 
                    onchange={() => toggleSeries(series)}
                  />
                  <span class="absolute inset-0 bg-bg-primary rounded-full border border-border-color transition-colors duration-200 peer-checked:bg-accent/15 peer-checked:border-accent"></span>
                  <span class="absolute left-1 bottom-1 w-4 h-4 rounded-full bg-gray-400 transition-transform duration-200 peer-checked:translate-x-5 peer-checked:bg-accent"></span>
                </label>
              </div>
            </div>
          </div>
        {/each}
      </div>
    {/if}

    <!-- LOGS TAB -->
    {#if activeTab === 'logs'}
      <div class="flex flex-col sm:flex-row gap-4 mb-6">
        <input 
          type="text" 
          placeholder="Search logs (series, chapter)..." 
          bind:value={logSearchQuery} 
          class="flex-grow bg-bg-secondary border border-border-color text-sm text-gray-200 px-4 py-3 rounded-lg focus:outline-none focus:border-accent"
        />
        <select bind:value={logChannelFilter} class="bg-bg-secondary border border-border-color text-sm text-gray-200 px-4 py-3 rounded-lg focus:outline-none cursor-pointer">
          <option value="ALL">All Channels</option>
          <option value="discord">Discord</option>
          <option value="slack">Slack</option>
          <option value="telegram">Telegram</option>
        </select>
        <select bind:value={logStatusFilter} class="bg-bg-secondary border border-border-color text-sm text-gray-200 px-4 py-3 rounded-lg focus:outline-none cursor-pointer">
          <option value="ALL">All Statuses</option>
          <option value="SENT">Sent</option>
          <option value="FAILED">Failed</option>
        </select>
      </div>

      <div class="bg-bg-secondary border border-border-color rounded-xl p-6">
        <div class="overflow-x-auto">
          <table class="w-full text-left text-xs border-collapse">
            <thead>
              <tr class="border-b border-border-color">
                <th class="pb-3 text-gray-400 font-semibold uppercase tracking-wider">Time / Date</th>
                <th class="pb-3 text-gray-400 font-semibold uppercase tracking-wider">Manga Series</th>
                <th class="pb-3 text-gray-400 font-semibold uppercase tracking-wider">Chapter</th>
                <th class="pb-3 text-gray-400 font-semibold uppercase tracking-wider">Channel</th>
                <th class="pb-3 text-gray-400 font-semibold uppercase tracking-wider">Status</th>
                <th class="pb-3 text-gray-400 font-semibold uppercase tracking-wider">Error Details</th>
                <th class="pb-3 text-gray-400 font-semibold uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-border-color/50">
              {#each filteredLogs as log}
                <tr>
                  <td class="py-3.5">
                    <div class="text-white font-medium">{new Date(log.createdAt).toLocaleDateString()}</div>
                    <div class="text-[10px] text-gray-500 mt-0.5">{new Date(log.createdAt).toLocaleTimeString()}</div>
                  </td>
                  <td class="py-3.5 font-semibold text-white">{log.seriesTitle}</td>
                  <td class="py-3.5 text-gray-300">Ch. {log.chapterNum}</td>
                  <td class="py-3.5">
                    <span class="px-2.5 py-0.5 rounded text-[10px] font-bold uppercase 
                      {log.channel === 'discord' ? 'bg-blue-500/10 text-blue-400' : ''}
                      {log.channel === 'slack' ? 'bg-amber-500/10 text-amber-400' : ''}
                      {log.channel === 'telegram' ? 'bg-sky-500/10 text-sky-400' : ''}"
                    >{log.channel}</span>
                  </td>
                  <td class="py-3.5">
                    <span class="px-2.5 py-0.5 rounded text-[10px] font-bold
                      {log.status === 'SENT' ? 'bg-success/10 text-success' : ''}
                      {log.status === 'FAILED' ? 'bg-error/10 text-error' : ''}"
                    >{log.status}</span>
                  </td>
                  <td class="py-3.5 text-error font-mono text-[10px] max-w-xs truncate">{log.errorMessage || '—'}</td>
                  <td class="py-3.5">
                    {#if log.status === 'FAILED'}
                      <button 
                        onclick={() => retryNotification(log)} 
                        disabled={retryingLogs[log.id]}
                        class="bg-accent text-white font-semibold text-[10px] px-2.5 py-1 rounded hover:opacity-90 transition-opacity cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        {retryingLogs[log.id] ? 'Retrying...' : '🔁 Retry'}
                      </button>
                    {:else}
                      <span class="text-gray-500">—</span>
                    {/if}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    {/if}
    </div>
  </main>
</div>

