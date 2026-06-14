package com.mangacdc.service;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.mangacdc.model.SeriesNotificationPrefs;
import org.springframework.dao.EmptyResultDataAccessException;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Service;

@Service
public class SeriesNotificationPrefsService {

    private final JdbcTemplate jdbc;
    private final ObjectMapper mapper;

    public SeriesNotificationPrefsService(JdbcTemplate jdbc) {
        this.jdbc = jdbc;
        this.mapper = new ObjectMapper();
    }

    public SeriesNotificationPrefs getPrefs(String seriesId) {
        try {
            String json = jdbc.queryForObject(
                "SELECT notification_prefs::text FROM manga_series WHERE id = ?::uuid",
                String.class,
                seriesId
            );
            return SeriesNotificationPrefs.fromJson(json, mapper);
        } catch (EmptyResultDataAccessException ex) {
            return SeriesNotificationPrefs.empty();
        }
    }
}
