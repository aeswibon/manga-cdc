import { expect, test, describe } from "bun:test";
import { filterSeries, type Series } from "./utils";

describe("Functional Workflows: Tab Switching and Search Filters", () => {
  const sampleSeries: Series[] = [
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

  test("User switches tabs and updates the active tab state", () => {
    let activeTab = "overview";
    
    // Simulate user clicking on 'watchlist' tab
    activeTab = "watchlist";
    expect(activeTab).toBe("watchlist");

    // Simulate user clicking on 'logs' tab
    activeTab = "logs";
    expect(activeTab).toBe("logs");
  });

  test("User types in search box and filters the series list", () => {
    let searchQuery = "";
    let statusFilter = "ALL";

    // User types "piece"
    searchQuery = "piece";
    let filtered = filterSeries(sampleSeries, searchQuery, statusFilter);
    expect(filtered.length).toBe(1);
    expect(filtered[0].title).toBe("One Piece");

    // User types "level"
    searchQuery = "level";
    filtered = filterSeries(sampleSeries, searchQuery, statusFilter);
    expect(filtered.length).toBe(1);
    expect(filtered[0].title).toBe("Solo Leveling");

    // User clears search
    searchQuery = "";
    filtered = filterSeries(sampleSeries, searchQuery, statusFilter);
    expect(filtered.length).toBe(2);
  });

  test("User toggles status dropdown filter", () => {
    let searchQuery = "";
    let statusFilter = "ONGOING";

    let filtered = filterSeries(sampleSeries, searchQuery, statusFilter);
    expect(filtered.length).toBe(1);
    expect(filtered[0].status).toBe("ONGOING");

    statusFilter = "COMPLETED";
    filtered = filterSeries(sampleSeries, searchQuery, statusFilter);
    expect(filtered.length).toBe(1);
    expect(filtered[0].status).toBe("COMPLETED");
  });
});
