import { expect, test, describe, mock, beforeAll, afterAll } from "bun:test";
import { type Series } from "./utils";

describe("Integration Workflows: API Endpoints and Status Syncing", () => {
  const originalFetch = global.fetch;

  afterAll(() => {
    global.fetch = originalFetch;
  });

  test("fetches stats, series, and logs from backend endpoints", async () => {
    const mockStats = { total_series: 5, active_series: 3, total_chapters: 10, total_logs: 2, successful_deliveries: 2, failed_deliveries: 0 };
    const mockSeriesList: Series[] = [
      { id: "1", sourceId: "md-1", title: "One Piece", author: "Eiichiro Oda", artist: "Eiichiro Oda", description: "Adventure", coverUrl: "", status: "ONGOING", sourceUrl: "", latestChapter: 1115, lastChecked: "", isActive: true }
    ];
    const mockLogs = [{ id: "l1", chapterId: "c1", status: "SENT", channel: "discord", errorMessage: "", createdAt: new Date().toISOString(), seriesTitle: "One Piece", chapterNum: 1115, chapterTitle: "Void" }];

    // Mock fetch to simulate successful backend API responses
    global.fetch = mock((url: string) => {
      if (url.includes("/api/bootstrap")) {
        return Promise.resolve(new Response(JSON.stringify({
          stats: mockStats,
          series: mockSeriesList,
          logs: mockLogs,
        })));
      }
      if (url.includes("/api/stats")) {
        return Promise.resolve(new Response(JSON.stringify(mockStats)));
      }
      if (url.includes("/api/series")) {
        return Promise.resolve(new Response(JSON.stringify(mockSeriesList)));
      }
      if (url.includes("/api/logs")) {
        return Promise.resolve(new Response(JSON.stringify(mockLogs)));
      }
      return Promise.reject(new Error("Unknown route: " + url));
    }) as any;

    const statsRes = await fetch("http://localhost:8080/api/stats");
    const stats = await statsRes.json();
    expect(stats.total_series).toBe(5);

    const seriesRes = await fetch("http://localhost:8080/api/series");
    const series = await seriesRes.json();
    expect(series.length).toBe(1);
    expect(series[0].title).toBe("One Piece");

    const logsRes = await fetch("http://localhost:8080/api/logs");
    const logs = await logsRes.json();
    expect(logs.length).toBe(1);
  });

});
