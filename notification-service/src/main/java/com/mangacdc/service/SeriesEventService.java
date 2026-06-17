package com.mangacdc.service;

import com.fasterxml.jackson.databind.JsonNode;
import com.mangacdc.repository.SeriesRepository;
import com.mangacdc.security.SecurityUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;

import java.util.Map;

@Service
public class SeriesEventService {

    private static final Logger log = LoggerFactory.getLogger(SeriesEventService.class);

    private final NotifierRegistry notifierRegistry;
    private final SeriesRepository seriesRepository;

    public SeriesEventService(NotifierRegistry notifierRegistry, SeriesRepository seriesRepository) {
        this.notifierRegistry = notifierRegistry;
        this.seriesRepository = seriesRepository;
    }

    public void processSeriesEvent(JsonNode root) {
        JsonNode after = root.path("after");
        if (after.isMissingNode() || after.isNull()) {
            return;
        }

        String seriesId = after.path("series_id").asText("");
        String title = after.path("title").asText("Unknown");
        String status = after.path("status").asText("");
        String sourceUrl = after.path("source_url").asText("");
        String alertType = after.path("alert_type").asText("status_change");

        if (seriesId.isBlank() || status.isBlank()) {
            return;
        }
        if (!SecurityUtils.isHttpUrl(sourceUrl)) {
            log.warn("Rejected series alert with invalid URL for series {}", seriesId);
            return;
        }
        if (!seriesRepository.tryMarkStatusAlert(seriesId)) {
            log.info("Skipping duplicate status alert for series {} ({})", title, status);
            return;
        }

        String alertTitle = "HIATUS".equals(status) ? "Series on hiatus" : "Series completed";
        String message = String.format("%s is now **%s**.", title, status);
        if ("stale".equals(alertType)) {
            alertTitle = "Series may be stale";
            message = String.format("%s has had no new chapters recently.", title);
        }

        Map<String, Boolean> results = notifierRegistry.sendSeriesAlert(title, alertTitle, message, sourceUrl);
        log.info("Processed series alert for {} ({}): {} channel(s)", title, status, results.size());
    }
}
