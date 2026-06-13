package com.mangacdc.validation;

import com.mangacdc.model.MangaSeries;

import java.net.URI;
import java.net.URISyntaxException;
import java.util.ArrayList;
import java.util.List;
import java.util.Locale;
import java.util.Set;

public final class SeriesValidator {

    private static final Set<String> GARBAGE_TITLES = Set.of(
            "404", "403", "500", "undefined", "null", "loading...", "loading", "not found", "error", "access denied"
    );

    private static final Set<String> ALLOWED_STATUSES = Set.of(
            "ONGOING", "COMPLETED", "HIATUS", "CANCELLED"
    );

    private SeriesValidator() {}

    public static List<String> validate(MangaSeries series) {
        List<String> errors = new ArrayList<>();

        String title = trim(series.title());
        if (title.isEmpty()) {
            errors.add("title is required");
        } else if (GARBAGE_TITLES.contains(title.toLowerCase(Locale.ROOT))) {
            errors.add("title looks like a scrape error");
        }

        if (trim(series.sourceId()).isEmpty()) {
            errors.add("source_id is required");
        }

        String sourceUrl = trim(series.sourceUrl());
        if (sourceUrl.isEmpty()) {
            errors.add("source_url is required");
        } else if (!isHttpUrl(sourceUrl)) {
            errors.add("source_url must be http or https");
        }

        String coverUrl = trim(series.coverUrl());
        if (!coverUrl.isEmpty() && !isHttpUrl(coverUrl)) {
            errors.add("cover_url must be http or https when set");
        }

        String status = trim(series.status()).toUpperCase(Locale.ROOT);
        if (status.isEmpty()) {
            errors.add("status is required");
        } else if (!ALLOWED_STATUSES.contains(status)) {
            errors.add("status must be ONGOING, COMPLETED, HIATUS, or CANCELLED");
        }

        return errors;
    }

    public static MangaSeries normalize(MangaSeries series) {
        String status = trim(series.status()).toUpperCase(Locale.ROOT);
        if (status.isEmpty()) {
            status = "ONGOING";
        }
        return new MangaSeries(
                series.id(),
                trim(series.sourceId()),
                trim(series.title()),
                trim(series.author()),
                trim(series.artist()),
                trim(series.description()),
                trim(series.coverUrl()),
                status,
                trim(series.sourceUrl()),
                series.latestChapter(),
                series.lastChecked(),
                series.isActive()
        );
    }

    private static String trim(String value) {
        return value == null ? "" : value.trim();
    }

    private static boolean isHttpUrl(String raw) {
        try {
            URI uri = new URI(raw);
            String scheme = uri.getScheme();
            return scheme != null
                    && ("http".equalsIgnoreCase(scheme) || "https".equalsIgnoreCase(scheme))
                    && uri.getHost() != null
                    && !uri.getHost().isBlank();
        } catch (URISyntaxException e) {
            return false;
        }
    }
}
