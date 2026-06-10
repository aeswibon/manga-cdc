package com.mangacdc.service;

import com.mangacdc.model.Chapter;
import com.mangacdc.repository.ChapterRepository;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

@Component
public class NotificationPoller {

    private static final Logger log = LoggerFactory.getLogger(NotificationPoller.class);

    private final ChapterRepository chapterRepo;
    private final DiscordNotifier discordNotifier;
    private final JdbcTemplate jdbc;

    public NotificationPoller(ChapterRepository chapterRepo,
                               DiscordNotifier discordNotifier,
                               JdbcTemplate jdbc) {
        this.chapterRepo = chapterRepo;
        this.discordNotifier = discordNotifier;
        this.jdbc = jdbc;
    }

    @Scheduled(fixedDelayString = "${poller.interval-ms:30000}")
    public void pollNewChapters() {
        var chapters = chapterRepo.findNewChapters();
        if (chapters.isEmpty()) {
            return;
        }

        log.info("Found {} new chapters to notify", chapters.size());

        for (Chapter ch : chapters) {
            String seriesTitle = jdbc.queryForObject(
                "SELECT title FROM manga_series WHERE id = ?",
                String.class, ch.seriesId());

            boolean success = discordNotifier.sendChapterAlert(
                seriesTitle != null ? seriesTitle : "Unknown",
                String.valueOf(ch.chapterNum()),
                ch.title(),
                ch.url()
            );

            String status = success ? "SENT" : "FAILED";
            String error = success ? null : "Webhook returned error or not configured";
            chapterRepo.logNotification(ch.id(), status, "discord", error);

            if (success) {
                chapterRepo.markNotified(ch.id());
            }
        }
    }
}
