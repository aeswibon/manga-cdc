import { expect, test, describe } from "bun:test";
import {
  calculateSuccessRate,
  filterSeries,
  filterLogs,
  duplicateTitleKeys,
  formatRelativeTime,
  parseSourceDisplay,
  readOnSourceLabel,
  seriesStatusLabel,
  seriesStatusVariant,
  parsePipelineHealth,
  parseStatusPagePayload,
  pipelineHealthFromStatusPage,
  healthLabel,
  healthShortLabel,
  healthVariant,
  notifierApiUrl,
  type Series,
} from "./utils";

describe("calculateSuccessRate", () => {
  test("returns 100 when totalLogs is 0", () => {
    expect(calculateSuccessRate(0, 0)).toBe(100);
    expect(calculateSuccessRate(5, 0)).toBe(100);
  });

  test("calculates correct percentage", () => {
    expect(calculateSuccessRate(1, 2)).toBe(50);
    expect(calculateSuccessRate(2, 3)).toBe(67);
    expect(calculateSuccessRate(10, 10)).toBe(100);
  });
});

describe("notifierApiUrl", () => {
  test("uses direct /api paths in development", () => {
    expect(notifierApiUrl("/api/stats")).toBe("/api/stats");
    expect(notifierApiUrl("stats")).toBe("/api/stats");
  });

  test("strips /api prefix for the Vercel proxy base", () => {
    expect(notifierApiUrl("/api/stats", "/api/notifier")).toBe("/api/notifier/stats");
    expect(notifierApiUrl("/api/series/abc/chapters", "/api/notifier")).toBe(
      "/api/notifier/series/abc/chapters",
    );
  });
});

describe("filterSeries", () => {
  const mockSeries: Series[] = [
    {
      id: "1",
      sourceId: "md-1",
      title: "One Piece",
      author: "Eiichiro Oda",
      artist: "Eiichiro Oda",
      description: "Pirate king adventure",
      coverUrl: "",
      status: "ONGOING",
      sourceUrl: "",
      latestChapter: 1115,
      lastChecked: "",
      isActive: true,
    },
    {
      id: "2",
      sourceId: "md-2",
      title: "Solo Leveling",
      author: "Chugong",
      artist: "DUBU",
      description: "Weakest hunter leveling up",
      coverUrl: "",
      status: "COMPLETED",
      sourceUrl: "",
      latestChapter: 200,
      lastChecked: "",
      isActive: false,
    },
  ];

  test("returns all series when no query or filter matches", () => {
    const result = filterSeries(mockSeries, "", "ALL");
    expect(result.length).toBe(2);
  });

  test("filters by search query matching title", () => {
    const result = filterSeries(mockSeries, "one", "ALL");
    expect(result.length).toBe(1);
    expect(result[0].title).toBe("One Piece");
  });

  test("filters by search query matching description", () => {
    const result = filterSeries(mockSeries, "hunter", "ALL");
    expect(result.length).toBe(1);
    expect(result[0].title).toBe("Solo Leveling");
  });

  test("filters by status", () => {
    const ongoing = filterSeries(mockSeries, "", "ONGOING");
    expect(ongoing.length).toBe(1);
    expect(ongoing[0].title).toBe("One Piece");

    const completed = filterSeries(mockSeries, "", "COMPLETED");
    expect(completed.length).toBe(1);
    expect(completed[0].title).toBe("Solo Leveling");
  });
});

describe("filterLogs", () => {
  const mockLogs = [
    {
      id: "l1",
      chapterId: "c1",
      status: "SENT",
      channel: "discord",
      errorMessage: "",
      createdAt: "",
      seriesTitle: "One Piece",
      chapterNum: 1115,
      chapterTitle: "Void",
    },
    {
      id: "l2",
      chapterId: "c2",
      status: "FAILED",
      channel: "telegram",
      errorMessage: "Connection Timeout",
      createdAt: "",
      seriesTitle: "Solo Leveling",
      chapterNum: 200,
      chapterTitle: "Epilogue",
    },
  ];

  test("returns all logs when filter parameters are ALL/empty", () => {
    const result = filterLogs(mockLogs, "", "ALL", "ALL");
    expect(result.length).toBe(2);
  });

  test("filters by search query matching series title", () => {
    const result = filterLogs(mockLogs, "piece", "ALL", "ALL");
    expect(result.length).toBe(1);
    expect(result[0].seriesTitle).toBe("One Piece");
  });

  test("filters by channel", () => {
    const result = filterLogs(mockLogs, "", "telegram", "ALL");
    expect(result.length).toBe(1);
    expect(result[0].channel).toBe("telegram");
  });

  test("filters by status", () => {
    const result = filterLogs(mockLogs, "", "ALL", "FAILED");
    expect(result.length).toBe(1);
    expect(result[0].status).toBe("FAILED");
  });
});

describe("watchlist helpers", () => {
  test("parseSourceDisplay splits namespaced ids", () => {
    const parsed = parseSourceDisplay("mangadex:a1c7c817-4e59-43b7-9365-09675a149a6f");
    expect(parsed.source).toBe("mangadex");
    expect(parsed.rawId).toBe("a1c7c817-4e59-43b7-9365-09675a149a6f");
    expect(parsed.shortId).toContain("…");
  });

  test("duplicateTitleKeys finds repeated titles", () => {
    const list: Series[] = [
      { id: "1", sourceId: "mangadex:a", title: "One Piece", author: "", artist: "", description: "", coverUrl: "", status: "ONGOING", sourceUrl: "", latestChapter: 1, lastChecked: "", isActive: true },
      { id: "2", sourceId: "mangadex:b", title: "one piece", author: "", artist: "", description: "", coverUrl: "", status: "ONGOING", sourceUrl: "", latestChapter: 1, lastChecked: "", isActive: true },
      { id: "3", sourceId: "mangadex:c", title: "Solo Leveling", author: "", artist: "", description: "", coverUrl: "", status: "ONGOING", sourceUrl: "", latestChapter: 1, lastChecked: "", isActive: true },
    ];
    const dupes = duplicateTitleKeys(list);
    expect(dupes.has("one piece")).toBe(true);
    expect(dupes.size).toBe(1);
  });

  test("formatRelativeTime returns human-readable value", () => {
    const recent = new Date(Date.now() - 5 * 60 * 1000).toISOString();
    expect(formatRelativeTime(recent)).toMatch(/minute|min/i);
    expect(formatRelativeTime("")).toBe("never");
  });

  test("readOnSourceLabel maps known sources", () => {
    expect(readOnSourceLabel("mangadex")).toBe("MangaDex");
    expect(readOnSourceLabel("custom")).toBe("custom");
  });

  test("seriesStatusLabel and variant map publication status", () => {
    expect(seriesStatusLabel("ONGOING")).toBe("Ongoing");
    expect(seriesStatusLabel("COMPLETED")).toBe("Completed");
    expect(seriesStatusVariant("ONGOING")).toBe("ongoing");
    expect(seriesStatusVariant("COMPLETED")).toBe("completed");
    expect(seriesStatusVariant("HIATUS")).toBe("unknown");
  });
});

describe("pipeline health helpers", () => {
  const sampleHealth = {
    status: "operational",
    updatedAt: "2026-06-13T06:36:33.566Z",
    components: [
      { name: "notifier", status: "operational", detail: "Notification API responding" },
      { name: "database", status: "operational", detail: "PostgreSQL reachable" },
    ],
  };

  test("healthLabel maps pipeline status values", () => {
    expect(healthLabel("operational")).toBe("Operational");
    expect(healthLabel("degraded")).toBe("Degraded");
    expect(healthLabel("down")).toBe("Down");
    expect(healthVariant("maintenance")).toBe("maintenance");
  });

  test("healthShortLabel maps compact labels for header", () => {
    expect(healthShortLabel("operational")).toBe("OK");
    expect(healthShortLabel("degraded")).toBe("Warn");
    expect(healthShortLabel("down")).toBe("Down");
    expect(healthShortLabel("offline")).toBe("Offline");
  });

  test("parsePipelineHealth validates payload shape", () => {
    expect(parsePipelineHealth(sampleHealth)?.status).toBe("operational");
    expect(parsePipelineHealth({})).toBeNull();
  });

  test("parsePipelineHealth keeps component details", () => {
    const parsed = parsePipelineHealth(sampleHealth);
    expect(parsed?.components).toHaveLength(2);
    expect(parsed?.components[0].name).toBe("notifier");
  });

  test("parseStatusPagePayload maps status page API responses", () => {
    const payload = parseStatusPagePayload({
      status: "offline",
      label: "Pipeline Offline",
      checkedAt: "2026-06-13T08:00:00.000Z",
      latencyMs: 120,
      components: [],
      error: "HTTP 404",
    });
    expect(payload?.status).toBe("offline");
    expect(healthLabel(payload!.status)).toBe("Offline");
    const mapped = pipelineHealthFromStatusPage(payload!);
    expect(mapped.updatedAt).toBe("2026-06-13T08:00:00.000Z");
    expect(healthVariant(mapped.status)).toBe("down");
  });
});
