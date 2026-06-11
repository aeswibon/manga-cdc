package com.mangacdc.service;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.mangacdc.repository.ChapterRepository;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Service;

import java.util.Map;

@Service
public class ChapterEventService {

    private static final Logger log = LoggerFactory.getLogger(ChapterEventService.class);

    private final NotifierRegistry notifierRegistry;
    private final ChapterRepository chapterRepo;
    private final JdbcTemplate jdbc;
    private final ObjectMapper mapper;

    public ChapterEventService(NotifierRegistry notifierRegistry,
                                ChapterRepository chapterRepo,
                                JdbcTemplate jdbc) {
        this.notifierRegistry = notifierRegistry;
        this.chapterRepo = chapterRepo;
        this.jdbc = jdbc;
        this.mapper = new ObjectMapper();
    }

    public void processChapterEvent(String message) {
        try {
            JsonNode root = mapper.readTree(message);

            String op = root.path("op").asText();
            if (!"c".equals(op)) {
                return;
            }

            JsonNode after = root.path("after");
            if (after.isMissingNode() || after.isNull()) {
                return;
            }

            String chapterId = after.path("id").asText();
            String seriesId = after.path("series_id").asText();
            String chapterNum = after.path("chapter_num").asText();
            String title = after.path("title").asText("");
            String url = after.path("url").asText();
            boolean isNew = after.path("is_new").asBoolean(false);

            if (!isNew) {
                return;
            }

            String seriesTitle = jdbc.queryForObject(
                "SELECT title FROM manga_series WHERE id = ?::uuid",
                String.class, seriesId);

            String resolvedTitle = seriesTitle != null ? seriesTitle : "Unknown";

            Map<String, Boolean> results = notifierRegistry.sendAll(resolvedTitle, chapterNum, title, url);

            boolean anySuccess = false;
            for (Map.Entry<String, Boolean> entry : results.entrySet()) {
                String channel = entry.getKey();
                boolean success = entry.getValue();
                String status = success ? "SENT" : "FAILED";
                String error = success ? null : "Webhook returned error";
                chapterRepo.logNotification(chapterId, status, channel, error);
                if (success) {
                    anySuccess = true;
                }
            }

            if (anySuccess) {
                chapterRepo.markNotified(chapterId);
            }

            log.info("Processed chapter {} for series {}: {} channel(s)",
                chapterNum, resolvedTitle, results.size());

        } catch (Exception e) {
            log.error("Failed to process chapter event: {}", e.getMessage(), e);
        }
    }
}
