package com.mangacdc.controller;

import com.mangacdc.config.MutationGuard;
import com.mangacdc.model.Chapter;
import com.mangacdc.model.MangaSeries;
import com.mangacdc.repository.ChapterRepository;
import com.mangacdc.repository.SeriesRepository;
import com.mangacdc.validation.SeriesValidator;
import org.springframework.http.HttpStatus;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.server.ResponseStatusException;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

@RestController
@RequestMapping("/api")
public class MangaApiController {

    private final SeriesRepository seriesRepository;
    private final ChapterRepository chapterRepository;
    private final JdbcTemplate jdbc;
    private final MutationGuard mutationGuard;

    public MangaApiController(
            SeriesRepository seriesRepository,
            ChapterRepository chapterRepository,
            JdbcTemplate jdbc,
            MutationGuard mutationGuard) {
        this.seriesRepository = seriesRepository;
        this.chapterRepository = chapterRepository;
        this.jdbc = jdbc;
        this.mutationGuard = mutationGuard;
    }

    @GetMapping("/series")
    public List<MangaSeries> listSeries() {
        return seriesRepository.findAll();
    }

    @GetMapping("/series/{id}/chapters")
    public List<Chapter> listSeriesChapters(
            @PathVariable String id,
            @RequestParam(defaultValue = "20") int limit) {
        return chapterRepository.findBySeriesId(id, limit);
    }

    @PutMapping("/series/{id}/status")
    public Map<String, Object> updateSeriesStatus(
            @PathVariable String id,
            @RequestParam boolean active,
            @RequestHeader(value = "X-Admin-Key", required = false) String adminKey) {
        mutationGuard.requireMutationAccess(adminKey);
        seriesRepository.updateActiveStatus(id, active);
        Map<String, Object> response = new HashMap<>();
        response.put("status", "OK");
        response.put("id", id);
        response.put("is_active", active);
        return response;
    }

    @GetMapping("/stats")
    public Map<String, Object> getStats() {
        Map<String, Object> seriesStats = jdbc.queryForMap(
                "SELECT COUNT(*) as total_series, COUNT(CASE WHEN is_active THEN 1 END) as active_series FROM manga_series"
        );
        Map<String, Object> chapterStats = jdbc.queryForMap(
                "SELECT COUNT(*) as total_chapters FROM chapters"
        );
        Map<String, Object> logStats = jdbc.queryForMap(
                "SELECT COUNT(*) as total_logs, COUNT(CASE WHEN status = 'SENT' THEN 1 END) as successful_deliveries, COUNT(CASE WHEN status = 'FAILED' THEN 1 END) as failed_deliveries FROM notification_logs"
        );

        Map<String, Object> stats = new HashMap<>();
        stats.put("total_series", ((Number) seriesStats.getOrDefault("total_series", 0)).intValue());
        stats.put("active_series", ((Number) seriesStats.getOrDefault("active_series", 0)).intValue());
        stats.put("total_chapters", ((Number) chapterStats.getOrDefault("total_chapters", 0)).intValue());
        stats.put("total_logs", ((Number) logStats.getOrDefault("total_logs", 0)).intValue());
        stats.put("successful_deliveries", ((Number) logStats.getOrDefault("successful_deliveries", 0)).intValue());
        stats.put("failed_deliveries", ((Number) logStats.getOrDefault("failed_deliveries", 0)).intValue());
        return stats;
    }

    @PostMapping("/series")
    public Map<String, Object> addSeries(
            @RequestBody MangaSeries series,
            @RequestHeader(value = "X-Admin-Key", required = false) String adminKey) {
        mutationGuard.requireMutationAccess(adminKey);
        MangaSeries normalized = SeriesValidator.normalize(series);
        List<String> errors = SeriesValidator.validate(normalized);
        if (!errors.isEmpty()) {
            throw new ResponseStatusException(HttpStatus.BAD_REQUEST, String.join("; ", errors));
        }
        seriesRepository.save(normalized);
        Map<String, Object> response = new HashMap<>();
        response.put("status", "CREATED");
        response.put("title", normalized.title());
        return response;
    }

    @DeleteMapping("/series/{id}")
    public Map<String, Object> deleteSeries(
            @PathVariable String id,
            @RequestHeader(value = "X-Admin-Key", required = false) String adminKey) {
        mutationGuard.requireMutationAccess(adminKey);
        seriesRepository.deleteById(id);
        Map<String, Object> response = new HashMap<>();
        response.put("status", "DELETED");
        response.put("id", id);
        return response;
    }
}
