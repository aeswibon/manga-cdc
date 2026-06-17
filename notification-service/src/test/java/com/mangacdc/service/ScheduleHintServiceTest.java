package com.mangacdc.service;

import org.junit.jupiter.api.Test;

import java.time.Instant;
import java.time.ZoneOffset;
import java.time.ZonedDateTime;
import java.util.ArrayList;
import java.util.List;
import java.util.Optional;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

class ScheduleHintServiceTest {

    @Test
    void computeHint_returnsEmptyWhenTooFewSamples() {
        List<Instant> dates = List.of(
            instant(2026, 1, 7, 16),
            instant(2026, 1, 14, 16)
        );
        assertTrue(ScheduleHintService.computeHint(dates).isEmpty());
    }

    @Test
    void computeHint_detectsDominantWednesday() {
        List<Instant> dates = new ArrayList<>();
        Instant base = instant(2026, 1, 7, 16);
        for (int week = 0; week < 5; week++) {
            dates.add(base.plusSeconds(7L * 24 * 3600 * week));
        }
        Optional<String> hint = ScheduleHintService.computeHint(dates);
        assertTrue(hint.isPresent());
        assertEquals("Usually drops Wednesday ~16:00 UTC", hint.get());
    }

    private static Instant instant(int year, int month, int day, int hour) {
        return ZonedDateTime.of(year, month, day, hour, 0, 0, 0, ZoneOffset.UTC).toInstant();
    }
}
