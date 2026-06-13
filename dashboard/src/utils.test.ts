import { expect, test, describe } from "bun:test";
import { calculateSuccessRate, filterSeries, filterLogs, type Series } from "./utils";

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
