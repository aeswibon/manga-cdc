package com.mangacdc.repository;

import com.mangacdc.model.Chapter;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.jdbc.core.DataClassRowMapper;
import org.springframework.jdbc.core.RowMapper;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Repository;

import java.util.List;

@Repository
public class ChapterRepository {

    private static final RowMapper<Chapter> CHAPTER_ROW_MAPPER = (rs, rowNum) -> {
        java.sql.Timestamp releaseDate = rs.getTimestamp("release_date");
        return new Chapter(
                rs.getString("id"),
                rs.getString("series_id"),
                rs.getDouble("chapter_num"),
                rs.getString("title"),
                rs.getString("url"),
                releaseDate != null ? releaseDate.toInstant() : null,
                rs.getBoolean("is_new"));
    };

    private final JdbcTemplate jdbc;
    private final JdbcTemplate readJdbc;

    public ChapterRepository(JdbcTemplate jdbc, @Qualifier("readJdbcTemplate") JdbcTemplate readJdbc) {
        this.jdbc = jdbc;
        this.readJdbc = readJdbc;
    }

    public List<Chapter> findNewChapters() {
        return readJdbc.query(
            "SELECT id, series_id, chapter_num, title, url, release_date, is_new " +
            "FROM chapters WHERE is_new = true ORDER BY release_date DESC LIMIT 50",
            CHAPTER_ROW_MAPPER);
    }

    public List<Chapter> findRecentChapters(int limit) {
        return readJdbc.query(
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
        return readJdbc.query(
            "SELECT id, series_id, chapter_num, title, url, release_date, is_new " +
            "FROM chapters WHERE series_id = ?::uuid ORDER BY chapter_num DESC LIMIT ?",
            CHAPTER_ROW_MAPPER,
            seriesId,
            capped);
    }

    public void markNotified(String chapterId) {
        jdbc.update("UPDATE chapters SET is_new = false WHERE id = ?::uuid", chapterId);
    }

    public boolean existsNewChapter(String chapterId) {
        Integer count = readJdbc.queryForObject(
                "SELECT COUNT(*) FROM chapters WHERE id = ?::uuid AND is_new = true",
                Integer.class,
                chapterId);
        return count != null && count > 0;
    }

    public String findChapterUrl(String chapterId) {
        return readJdbc.query(
                "SELECT url FROM chapters WHERE id = ?::uuid",
                rs -> rs.next() ? rs.getString("url") : null,
                chapterId);
    }

    public String findScanGroup(String chapterId) {
        return readJdbc.query(
            "SELECT scan_group FROM chapters WHERE id = ?::uuid",
            rs -> rs.next() ? rs.getString("scan_group") : null,
            chapterId);
    }

    public java.time.Instant findReleaseDate(String chapterId) {
        return readJdbc.query(
            "SELECT release_date FROM chapters WHERE id = ?::uuid",
            rs -> {
                if (!rs.next()) {
                    return null;
                }
                java.sql.Timestamp ts = rs.getTimestamp("release_date");
                return ts != null ? ts.toInstant() : null;
            },
            chapterId);
    }

    public void logNotification(String chapterId, String status, String channel, String errorMessage) {
        jdbc.update(
            "INSERT INTO notification_logs (chapter_id, status, channel, error_message) VALUES (?::uuid, ?, ?, ?)",
            chapterId, status, channel, errorMessage);
    }

    public int countNewChaptersForSeries(String seriesId) {
        Integer count = readJdbc.queryForObject(
            "SELECT COUNT(*) FROM chapters WHERE series_id = ?::uuid AND is_new = true",
            Integer.class,
            seriesId);
        return count != null ? count : 0;
    }

    public List<Chapter> findNewChaptersForSeries(String seriesId) {
        return readJdbc.query(
            "SELECT id, series_id, chapter_num, title, url, release_date, is_new " +
            "FROM chapters WHERE series_id = ?::uuid AND is_new = true ORDER BY chapter_num ASC",
            CHAPTER_ROW_MAPPER,
            seriesId);
    }

    public String findSeriesTitle(String seriesId) {
        return readJdbc.query(
            "SELECT title FROM manga_series WHERE id = ?::uuid",
            rs -> rs.next() ? rs.getString("title") : null,
            seriesId);
    }

    public List<java.time.Instant> findReleaseDatesForSeries(String seriesId, int limit) {
        return readJdbc.query(
            "SELECT release_date FROM chapters WHERE series_id = ?::uuid AND release_date IS NOT NULL " +
            "ORDER BY release_date DESC LIMIT ?",
            (rs, rowNum) -> {
                java.sql.Timestamp ts = rs.getTimestamp("release_date");
                return ts != null ? ts.toInstant() : null;
            },
            seriesId,
            limit).stream().filter(java.util.Objects::nonNull).toList();
    }

    public java.time.Instant findLatestReleaseDate(String seriesId) {
        return readJdbc.query(
            "SELECT MAX(release_date) FROM chapters WHERE series_id = ?::uuid",
            rs -> {
                if (!rs.next()) {
                    return null;
                }
                java.sql.Timestamp ts = rs.getTimestamp(1);
                return ts != null ? ts.toInstant() : null;
            },
            seriesId);
    }
}
