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
                       c.title AS chapter_title,
                       c.url AS chapter_url
                FROM notification_logs nl
                JOIN chapters c ON c.id = nl.chapter_id
                JOIN manga_series ms ON ms.id = c.series_id
                ORDER BY nl.created_at DESC
                LIMIT ?
                """,
                DataClassRowMapper.newInstance(NotificationLogEntry.class),
                capped);
    }

    public NotificationLogEntry findById(java.util.UUID id) {
        return jdbc.queryForObject(
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
                       c.title AS chapter_title,
                       c.url AS chapter_url
                FROM notification_logs nl
                JOIN chapters c ON c.id = nl.chapter_id
                JOIN manga_series ms ON ms.id = c.series_id
                WHERE nl.id = ?::uuid
                """,
                DataClassRowMapper.newInstance(NotificationLogEntry.class),
                id);
    }

    public void updateStatus(java.util.UUID id, String status, String errorMessage) {
        if ("SENT".equals(status)) {
            jdbc.update("UPDATE notification_logs SET status = ?, error_message = ?, sent_at = NOW() WHERE id = ?::uuid",
                    status, errorMessage, id);
        } else {
            jdbc.update("UPDATE notification_logs SET status = ?, error_message = ? WHERE id = ?::uuid",
                    status, errorMessage, id);
        }
    }

    public NotificationLogEntry findRecentForChapterAndChannel(java.util.UUID chapterId, String channel) {
        return jdbc.queryForObject(
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
                       c.title AS chapter_title,
                       c.url AS chapter_url
                FROM notification_logs nl
                JOIN chapters c ON c.id = nl.chapter_id
                JOIN manga_series ms ON ms.id = c.series_id
                WHERE nl.chapter_id = ?::uuid AND nl.channel = ?
                ORDER BY nl.created_at DESC
                LIMIT 1
                """,
                DataClassRowMapper.newInstance(NotificationLogEntry.class),
                chapterId, channel);
    }
}
