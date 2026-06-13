package com.mangacdc.model;

import java.math.BigDecimal;
import java.time.OffsetDateTime;
import java.util.UUID;

public record NotificationLogEntry(
        UUID id,
        UUID chapterId,
        String status,
        String channel,
        String errorMessage,
        OffsetDateTime sentAt,
        OffsetDateTime createdAt,
        String seriesTitle,
        BigDecimal chapterNum,
        String chapterTitle,
        String chapterUrl
) {}
