package com.mangacdc.repository;

import com.mangacdc.model.NotificationLogEntry;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.core.io.ClassPathResource;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.jdbc.datasource.DriverManagerDataSource;
import org.testcontainers.containers.PostgreSQLContainer;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;

import java.math.BigDecimal;
import java.util.List;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.assertEquals;

@Testcontainers
class NotificationLogRepositoryIntegrationTest {

    @Container
    private static final PostgreSQLContainer<?> postgres = new PostgreSQLContainer<>("postgres:16-alpine")
        .withDatabaseName("mangacdc")
        .withUsername("mangacdc")
        .withPassword("mangacdc");

    private static JdbcTemplate jdbc;
    private NotificationLogRepository repository;

    @BeforeAll
    static void migrate() throws Exception {
        var dataSource = new DriverManagerDataSource(
            postgres.getJdbcUrl(),
            postgres.getUsername(),
            postgres.getPassword()
        );
        jdbc = new JdbcTemplate(dataSource);

        var resource = new ClassPathResource("migration.sql");
        var sql = new String(resource.getInputStream().readAllBytes());
        for (var statement : sql.split(";")) {
            var trimmed = statement.trim();
            if (!trimmed.isEmpty()) {
                jdbc.execute(trimmed);
            }
        }
    }

    @BeforeEach
    void setUp() {
        repository = new NotificationLogRepository(jdbc);
        jdbc.execute("TRUNCATE notification_logs, chapters, manga_series CASCADE");
    }

    @Test
    void findRecent_returnsJoinedSeriesAndChapterDataOrderedByCreatedAtDesc() {
        String seriesId = UUID.randomUUID().toString();
        jdbc.update(
            "INSERT INTO manga_series (id, source_id, title, source_url, status, is_active) VALUES (?::uuid, ?, ?, ?, 'ONGOING', true)",
            seriesId, "repo-test", "One Piece", "https://example.com/one-piece"
        );

        String olderChapterId = UUID.randomUUID().toString();
        String newerChapterId = UUID.randomUUID().toString();
        jdbc.update(
            "INSERT INTO chapters (id, series_id, chapter_num, title, url, is_new) VALUES (?::uuid, ?::uuid, ?, ?, ?, true)",
            olderChapterId, seriesId, 1099, "Older Chapter", "https://example.com/ch-1099"
        );
        jdbc.update(
            "INSERT INTO chapters (id, series_id, chapter_num, title, url, is_new) VALUES (?::uuid, ?::uuid, ?, ?, ?, true)",
            newerChapterId, seriesId, 1100, "The Final Chapter", "https://example.com/ch-1100"
        );

        jdbc.update(
            "INSERT INTO notification_logs (chapter_id, status, channel, created_at) VALUES (?::uuid, 'SENT', 'discord', NOW() - INTERVAL '1 hour')",
            olderChapterId
        );
        jdbc.update(
            "INSERT INTO notification_logs (chapter_id, status, channel, error_message, created_at) VALUES (?::uuid, 'FAILED', 'slack', 'timeout', NOW())",
            newerChapterId
        );

        List<NotificationLogEntry> logs = repository.findRecent(50);

        assertEquals(2, logs.size());
        assertEquals("FAILED", logs.get(0).status());
        assertEquals("slack", logs.get(0).channel());
        assertEquals("One Piece", logs.get(0).seriesTitle());
        assertEquals(new BigDecimal("1100"), logs.get(0).chapterNum());
        assertEquals("The Final Chapter", logs.get(0).chapterTitle());
        assertEquals("timeout", logs.get(0).errorMessage());

        assertEquals("SENT", logs.get(1).status());
        assertEquals("discord", logs.get(1).channel());
        assertEquals(new BigDecimal("1099"), logs.get(1).chapterNum());
    }

    @Test
    void findRecent_honorsLimitAndTreatsZeroAsOne() {
        String seriesId = UUID.randomUUID().toString();
        jdbc.update(
            "INSERT INTO manga_series (id, source_id, title, source_url, status, is_active) VALUES (?::uuid, ?, ?, ?, 'ONGOING', true)",
            seriesId, "repo-limit", "Naruto", "https://example.com/naruto"
        );

        for (int i = 0; i < 5; i++) {
            String chapterId = UUID.randomUUID().toString();
            jdbc.update(
                "INSERT INTO chapters (id, series_id, chapter_num, title, url, is_new) VALUES (?::uuid, ?::uuid, ?, ?, ?, true)",
                chapterId, seriesId, i + 1, "Chapter " + (i + 1), "https://example.com/ch-" + (i + 1)
            );
            jdbc.update(
                "INSERT INTO notification_logs (chapter_id, status, channel, created_at) VALUES (?::uuid, 'SENT', 'discord', NOW() - (? || ' minutes')::interval)",
                chapterId, String.valueOf(5 - i)
            );
        }

        assertEquals(2, repository.findRecent(2).size());
        assertEquals(1, repository.findRecent(0).size());
        assertEquals(5, repository.findRecent(50).size());
    }
}
