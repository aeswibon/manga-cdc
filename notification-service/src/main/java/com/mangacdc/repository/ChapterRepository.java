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

    public List<Chapter> findRecentChapters(int limit) {
        return jdbc.query(
            "SELECT c.id, c.series_id, c.chapter_num, c.title, c.url, c.release_date, c.is_new, s.title as series_title " +
            "FROM chapters c JOIN manga_series s ON c.series_id = s.id " +
            "ORDER BY c.release_date DESC LIMIT ?",
            (rs, rowNum) -> {
                String title = rs.getString("title");
                String fullTitle = rs.getString("series_title") + (title != null && !title.isEmpty() ? " - " + title : "");
                java.sql.Timestamp ts = rs.getTimestamp("release_date");
                return new Chapter(
                    rs.getString("id"), rs.getString("series_id"), rs.getDouble("chapter_num"),
                    fullTitle, rs.getString("url"), ts != null ? ts.toInstant() : null, rs.getBoolean("is_new")
                );
            },
            limit);
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
