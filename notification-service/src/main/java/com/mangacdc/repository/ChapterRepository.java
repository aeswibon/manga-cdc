package com.mangacdc.repository;

import com.mangacdc.model.Chapter;
import org.springframework.jdbc.core.DataClassRowMapper;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Repository;

import java.util.List;

@Repository
public class ChapterRepository {

    private final JdbcTemplate jdbc;

    public ChapterRepository(JdbcTemplate jdbc) {
        this.jdbc = jdbc;
    }

    public List<Chapter> findNewChapters() {
        return jdbc.query(
            "SELECT id, series_id, chapter_num, title, url, release_date, is_new " +
            "FROM chapters WHERE is_new = true ORDER BY release_date DESC LIMIT 50",
            DataClassRowMapper.newInstance(Chapter.class));
    }

    public List<Chapter> findBySeriesId(String seriesId, int limit) {
        int capped = Math.min(Math.max(limit, 1), 100);
        return jdbc.query(
            "SELECT id, series_id, chapter_num, title, url, release_date, is_new " +
            "FROM chapters WHERE series_id = ?::uuid ORDER BY chapter_num DESC LIMIT ?",
            DataClassRowMapper.newInstance(Chapter.class),
            seriesId,
            capped);
    }

    public void markNotified(String chapterId) {
        jdbc.update("UPDATE chapters SET is_new = false WHERE id = ?::uuid", chapterId);
    }

    public boolean existsNewChapter(String chapterId) {
        Integer count = jdbc.queryForObject(
                "SELECT COUNT(*) FROM chapters WHERE id = ?::uuid AND is_new = true",
                Integer.class,
                chapterId);
        return count != null && count > 0;
    }

    public String findChapterUrl(String chapterId) {
        return jdbc.query(
                "SELECT url FROM chapters WHERE id = ?::uuid",
                rs -> rs.next() ? rs.getString("url") : null,
                chapterId);
    }

    public void logNotification(String chapterId, String status, String channel, String errorMessage) {
        jdbc.update(
            "INSERT INTO notification_logs (chapter_id, status, channel, error_message) VALUES (?::uuid, ?, ?, ?)",
            chapterId, status, channel, errorMessage);
    }
}
