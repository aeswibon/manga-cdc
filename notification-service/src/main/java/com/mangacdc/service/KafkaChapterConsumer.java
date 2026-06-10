package com.mangacdc.service;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.mangacdc.repository.ChapterRepository;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.kafka.annotation.KafkaListener;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Component;

@Component
public class KafkaChapterConsumer {

    private static final Logger log = LoggerFactory.getLogger(KafkaChapterConsumer.class);

    private final DiscordNotifier discordNotifier;
    private final ChapterRepository chapterRepo;
    private final JdbcTemplate jdbc;
    private final ObjectMapper mapper;
    private final boolean cdcEnabled;

    public KafkaChapterConsumer(DiscordNotifier discordNotifier,
                                 ChapterRepository chapterRepo,
                                 JdbcTemplate jdbc,
                                 @Value("${cdc.enabled:false}") boolean cdcEnabled) {
        this.discordNotifier = discordNotifier;
        this.chapterRepo = chapterRepo;
        this.jdbc = jdbc;
        this.mapper = new ObjectMapper();
        this.cdcEnabled = cdcEnabled;
    }

    @KafkaListener(topicPattern = "${cdc.topic-pattern:mangacdc.public.chapters}", groupId = "mangacdc-notification")
    public void onChapterEvent(String message) {
        if (!cdcEnabled) {
            return;
        }

        try {
            JsonNode root = mapper.readTree(message);

            String op = root.path("op").asText();
            if (!"c".equals(op) && !"r".equals(op)) {
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

            boolean success = discordNotifier.sendChapterAlert(
                seriesTitle != null ? seriesTitle : "Unknown",
                chapterNum,
                title,
                url
            );

            String status = success ? "SENT" : "FAILED";
            String error = success ? null : "Webhook returned error";
            chapterRepo.logNotification(chapterId, status, "discord", error);

            if (success) {
                chapterRepo.markNotified(chapterId);
            }

            log.info("CDC processed chapter {} for series {}: {}",
                chapterNum, seriesTitle, status);

        } catch (Exception e) {
            log.error("Failed to process CDC message: {}", e.getMessage(), e);
        }
    }
}
