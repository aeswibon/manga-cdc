package com.mangacdc.repository;

import com.mangacdc.model.MangaSeries;
import org.springframework.jdbc.core.DataClassRowMapper;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Repository;

import java.util.List;

@Repository
public class SeriesRepository {

    private final JdbcTemplate jdbc;

    public SeriesRepository(JdbcTemplate jdbc) {
        this.jdbc = jdbc;
    }

    public List<MangaSeries> findAll() {
        return jdbc.query(
            "SELECT id, source_id, title, author, artist, description, cover_url, status, source_url, latest_chapter, last_checked, is_active " +
            "FROM manga_series ORDER BY title ASC",
            DataClassRowMapper.newInstance(MangaSeries.class));
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
