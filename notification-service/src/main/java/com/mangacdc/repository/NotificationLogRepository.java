package com.mangacdc.repository;

import com.mangacdc.model.NotificationLogEntry;
import org.springframework.jdbc.core.DataClassRowMapper;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Repository;

import java.util.List;

@Repository
public class NotificationLogRepository {

    private final JdbcTemplate jdbc;

    public NotificationLogRepository(JdbcTemplate jdbc) {
        this.jdbc = jdbc;
    }

    public List<NotificationLogEntry> findRecent(int limit) {
        int capped = Math.min(Math.max(limit, 1), 200);
        return jdbc.query(
                """
                SELECT nl.id,
                       nl.chapter_id,
                       nl.status,
                       nl.channel,
                       nl.error_message,
                       nl.sent_at,
                       nl.created_at,
                       ms.title AS series_title,
                       c.chapter_num,
                       c.title AS chapter_title
                FROM notification_logs nl
                JOIN chapters c ON c.id = nl.chapter_id
                JOIN manga_series ms ON ms.id = c.series_id
                ORDER BY nl.created_at DESC
                LIMIT ?
                """,
                DataClassRowMapper.newInstance(NotificationLogEntry.class),
                capped);
    }
}
