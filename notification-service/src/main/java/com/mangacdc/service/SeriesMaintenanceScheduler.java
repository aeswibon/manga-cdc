package com.mangacdc.service;

import com.mangacdc.repository.SeriesRepository;
import com.mangacdc.repository.StaleSeriesCandidate;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

import java.util.List;
import java.util.Map;

@Component
@ConditionalOnProperty(name = "manga.series-maintenance.enabled", havingValue = "true", matchIfMissing = true)
public class SeriesMaintenanceScheduler {

    private static final Logger log = LoggerFactory.getLogger(SeriesMaintenanceScheduler.class);

    private final SeriesRepository seriesRepository;
    private final NotifierRegistry notifierRegistry;
    private final int staleAfterDays;
    private final int alertCooldownDays;

    public SeriesMaintenanceScheduler(
            SeriesRepository seriesRepository,
            NotifierRegistry notifierRegistry,
            @Value("${manga.stale-after-days:30}") int staleAfterDays,
            @Value("${manga.stale-alert-cooldown-days:30}") int alertCooldownDays) {
        this.seriesRepository = seriesRepository;
        this.notifierRegistry = notifierRegistry;
        this.staleAfterDays = staleAfterDays;
        this.alertCooldownDays = alertCooldownDays;
    }

    @Scheduled(fixedDelayString = "${manga.stale-check-interval-ms:7200000}")
    public void checkStaleSeries() {
        List<StaleSeriesCandidate> stale = seriesRepository.findStaleOngoingSeries(staleAfterDays, alertCooldownDays);
        for (StaleSeriesCandidate candidate : stale) {
            if (!seriesRepository.tryMarkStaleAlert(candidate.id())) {
                continue;
            }
            String message = String.format(
                "%s has had no new chapter in at least %d days.",
                candidate.title(),
                staleAfterDays);
            Map<String, Boolean> results = notifierRegistry.sendSeriesAlert(
                candidate.title(),
                "Series may be stale",
                message,
                candidate.sourceUrl());
            log.info("Stale-series alert for {}: {} channel(s)", candidate.title(), results.size());
        }
    }
}
