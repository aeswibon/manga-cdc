package com.mangacdc.model;

import java.time.Instant;

public record Chapter(
    String id,
    String seriesId,
    double chapterNum,
    String title,
    String url,
    Instant releaseDate,
    boolean isNew
) {}
