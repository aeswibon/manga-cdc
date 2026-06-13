package com.mangacdc.controller;

import com.mangacdc.model.MangaSeries;
import com.mangacdc.repository.SeriesRepository;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.web.bind.annotation.*;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

@RestController
@RequestMapping("/api")
@CrossOrigin(origins = "*")
public class MangaApiController {

    private final SeriesRepository seriesRepository;
    private final JdbcTemplate jdbc;

    public MangaApiController(SeriesRepository seriesRepository, JdbcTemplate jdbc) {
        this.seriesRepository = seriesRepository;
        this.jdbc = jdbc;
    }

    @GetMapping("/series")
    public List<MangaSeries> listSeries() {
        return seriesRepository.findAll();
    }

    @PutMapping("/series/{id}/status")
    public Map<String, Object> updateSeriesStatus(@PathVariable String id, @RequestParam boolean active) {
        seriesRepository.updateActiveStatus(id, active);
        Map<String, Object> response = new HashMap<>();
        response.put("status", "OK");
        response.put("id", id);
        response.put("is_active", active);
        return response;
    }

    @GetMapping("/stats")
    public Map<String, Object> getStats() {
        Integer totalSeries = jdbc.queryForObject("SELECT COUNT(*) FROM manga_series", Integer.class);
        Integer activeSeries = jdbc.queryForObject("SELECT COUNT(*) FROM manga_series WHERE is_active = true", Integer.class);
        Integer totalChapters = jdbc.queryForObject("SELECT COUNT(*) FROM chapters", Integer.class);
        Integer totalLogs = jdbc.queryForObject("SELECT COUNT(*) FROM notification_logs", Integer.class);
        Integer successfulDeliveries = jdbc.queryForObject("SELECT COUNT(*) FROM notification_logs WHERE status = 'SENT'", Integer.class);
        Integer failedDeliveries = jdbc.queryForObject("SELECT COUNT(*) FROM notification_logs WHERE status = 'FAILED'", Integer.class);

        Map<String, Object> stats = new HashMap<>();
        stats.put("total_series", totalSeries != null ? totalSeries : 0);
        stats.put("active_series", activeSeries != null ? activeSeries : 0);
        stats.put("total_chapters", totalChapters != null ? totalChapters : 0);
        stats.put("total_logs", totalLogs != null ? totalLogs : 0);
        stats.put("successful_deliveries", successfulDeliveries != null ? successfulDeliveries : 0);
        stats.put("failed_deliveries", failedDeliveries != null ? failedDeliveries : 0);
        return stats;
    }
}
