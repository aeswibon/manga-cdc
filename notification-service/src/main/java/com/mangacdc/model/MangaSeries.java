package com.mangacdc.model;

import java.time.Instant;

public record MangaSeries(
    String id,
    String sourceId,
    String title,
    String author,
    String artist,
    String description,
    String coverUrl,
    String status,
    String sourceUrl,
    Double latestChapter,
    Instant lastChecked,
    boolean isActive
) {}
