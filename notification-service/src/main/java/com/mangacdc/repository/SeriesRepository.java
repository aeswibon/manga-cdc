package com.mangacdc.repository;

import com.mangacdc.model.MangaSeries;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.jdbc.core.RowMapper;
import org.springframework.stereotype.Repository;

import java.util.List;

@Repository
public class SeriesRepository {

    private static final RowMapper<MangaSeries> ROW_MAPPER = (rs, rowNum) -> {
        java.sql.Timestamp lastChecked = rs.getTimestamp("last_checked");
        return new MangaSeries(
                rs.getString("id"),
                rs.getString("source_id"),
                rs.getString("title"),
                rs.getString("author"),
                rs.getString("artist"),
                rs.getString("description"),
                rs.getString("cover_url"),
                rs.getString("status"),
                rs.getString("source_url"),
                rs.getObject("latest_chapter") != null ? rs.getDouble("latest_chapter") : null,
                lastChecked != null ? lastChecked.toInstant() : null,
                rs.getBoolean("is_active"));
    };

    private final JdbcTemplate jdbc;
    private final JdbcTemplate readJdbc;

    public SeriesRepository(
            JdbcTemplate jdbc,
            @org.springframework.beans.factory.annotation.Qualifier("readJdbcTemplate") JdbcTemplate readJdbc) {
        this.jdbc = jdbc;
        this.readJdbc = readJdbc;
    }

    public List<MangaSeries> findAll() {
        return readJdbc.query(
            "SELECT id, source_id, title, author, artist, description, cover_url, status, source_url, latest_chapter, last_checked, is_active " +
            "FROM manga_series ORDER BY title ASC",
            ROW_MAPPER);
    }

    public List<MangaSeries> findAllActive() {
        return readJdbc.query(
            "SELECT id, source_id, title, author, artist, description, cover_url, status, source_url, latest_chapter, last_checked, is_active " +
            "FROM manga_series WHERE is_active = true ORDER BY title ASC",
            ROW_MAPPER);
    }

    public int countAll() {
        Integer count = readJdbc.queryForObject("SELECT COUNT(*) FROM manga_series", Integer.class);
        return count != null ? count : 0;
    }

    public int countActive() {
        Integer count = readJdbc.queryForObject("SELECT COUNT(*) FROM manga_series WHERE is_active = true", Integer.class);
        return count != null ? count : 0;
    }

    public void updateActiveStatus(String id, boolean active) {
        jdbc.update("UPDATE manga_series SET is_active = ?, updated_at = NOW() WHERE id = ?::uuid", active, id);
    }

    public void save(MangaSeries series) {
        jdbc.update(
            "INSERT INTO manga_series (source_id, title, author, artist, description, cover_url, status, source_url, is_active) " +
            "VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
            series.sourceId(), series.title(), series.author(), series.artist(),
            series.description(), series.coverUrl(), series.status(), series.sourceUrl(), series.isActive()
        );
    }

    public void deleteById(String id) {
        jdbc.update("DELETE FROM manga_series WHERE id = ?::uuid", id);
    }
}
