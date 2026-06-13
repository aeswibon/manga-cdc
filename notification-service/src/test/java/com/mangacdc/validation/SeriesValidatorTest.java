package com.mangacdc.validation;

import com.mangacdc.model.MangaSeries;
import org.junit.jupiter.api.Test;

import java.util.List;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

class SeriesValidatorTest {

    @Test
    void validate_acceptsValidSeries() {
        MangaSeries series = new MangaSeries(
                null,
                "md-1",
                "One Piece",
                "Author",
                "Artist",
                "Description",
                "https://example.com/cover.jpg",
                "ONGOING",
                "https://example.com/title/1",
                null,
                null,
                true
        );

        assertTrue(SeriesValidator.validate(series).isEmpty());
    }

    @Test
    void validate_rejectsGarbageTitle() {
        MangaSeries series = new MangaSeries(
                null,
                "md-1",
                "404",
                null,
                null,
                null,
                "https://example.com/cover.jpg",
                "ONGOING",
                "https://example.com/title/1",
                null,
                null,
                true
        );

        List<String> errors = SeriesValidator.validate(series);
        assertEquals(1, errors.size());
        assertTrue(errors.get(0).contains("scrape error"));
    }

    @Test
    void normalize_defaultsStatusToOngoing() {
        MangaSeries series = new MangaSeries(
                null,
                "md-1",
                "One Piece",
                null,
                null,
                null,
                null,
                "",
                "https://example.com/title/1",
                null,
                null,
                true
        );

        MangaSeries normalized = SeriesValidator.normalize(series);
        assertEquals("ONGOING", normalized.status());
        assertTrue(SeriesValidator.validate(normalized).isEmpty());
    }
}
