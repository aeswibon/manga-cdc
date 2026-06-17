package com.mangacdc.service;

import com.mangacdc.repository.ChapterRepository;
import org.springframework.stereotype.Service;

import java.time.DayOfWeek;
import java.time.Instant;
import java.time.ZoneOffset;
import java.time.ZonedDateTime;
import java.time.format.TextStyle;
import java.util.EnumMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Optional;

@Service
public class ScheduleHintService {

    private static final int MIN_SAMPLES = 5;
    private static final double MIN_WEEKDAY_RATIO = 0.6;

    private final ChapterRepository chapterRepository;

    public ScheduleHintService(ChapterRepository chapterRepository) {
        this.chapterRepository = chapterRepository;
    }

    public Optional<String> hintForSeries(String seriesId) {
        List<Instant> releaseDates = chapterRepository.findReleaseDatesForSeries(seriesId, 50);
        return computeHint(releaseDates);
    }

    static Optional<String> computeHint(List<Instant> releaseDates) {
        if (releaseDates == null || releaseDates.size() < MIN_SAMPLES) {
            return Optional.empty();
        }

        Map<DayOfWeek, Integer> byDay = new EnumMap<>(DayOfWeek.class);
        int[] hours = new int[releaseDates.size()];
        int index = 0;
        for (Instant instant : releaseDates) {
            if (instant == null) {
                continue;
            }
            ZonedDateTime zdt = instant.atZone(ZoneOffset.UTC);
            byDay.merge(zdt.getDayOfWeek(), 1, Integer::sum);
            hours[index++] = zdt.getHour();
        }
        if (index < MIN_SAMPLES) {
            return Optional.empty();
        }

        DayOfWeek dominantDay = null;
        int dominantCount = 0;
        for (Map.Entry<DayOfWeek, Integer> entry : byDay.entrySet()) {
            if (entry.getValue() > dominantCount) {
                dominantCount = entry.getValue();
                dominantDay = entry.getKey();
            }
        }
        if (dominantDay == null || (double) dominantCount / index < MIN_WEEKDAY_RATIO) {
            return Optional.empty();
        }

        java.util.Arrays.sort(hours, 0, index);
        int medianHour = hours[index / 2];
        String dayLabel = dominantDay.getDisplayName(TextStyle.FULL, Locale.ENGLISH);
        return Optional.of(String.format("Usually drops %s ~%02d:00 UTC", dayLabel, medianHour));
    }
}
