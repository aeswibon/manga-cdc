package com.mangacdc.service;
 
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.mangacdc.repository.ChapterRepository;
import com.mangacdc.repository.NotificationLogRepository;
import com.mangacdc.model.NotificationLogEntry;
import com.mangacdc.security.SecurityUtils;
import io.micrometer.core.instrument.MeterRegistry;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;
 
import java.util.Comparator;
import java.util.List;
import java.util.Map;
import java.util.UUID;
 
@Service
public class ChapterEventService {
 
    private static final Logger log = LoggerFactory.getLogger(ChapterEventService.class);
 
    private final NotifierRegistry notifierRegistry;
    private final ChapterRepository chapterRepo;
    private final NotificationLogRepository notificationLogRepo;
    private final SseEmitterService sseEmitterService;
    private final MeterRegistry meterRegistry;
    private final ChapterNotificationBatcher batcher;
    private final ObjectMapper mapper;
 
    public ChapterEventService(NotifierRegistry notifierRegistry,
                                ChapterRepository chapterRepo,
                                NotificationLogRepository notificationLogRepo,
                                SseEmitterService sseEmitterService,
                                MeterRegistry meterRegistry,
                                ChapterNotificationBatcher batcher) {
        this.notifierRegistry = notifierRegistry;
        this.chapterRepo = chapterRepo;
        this.notificationLogRepo = notificationLogRepo;
        this.sseEmitterService = sseEmitterService;
        this.meterRegistry = meterRegistry;
        this.batcher = batcher;
        this.mapper = new ObjectMapper();
    }

    public void processChapterEvent(String message) {
        try {
            JsonNode root = mapper.readTree(message);

            String op = root.path("op").asText();
            if (!"c".equals(op)) {
                return;
            }

            JsonNode after = root.path("after");
            if (after.isMissingNode() || after.isNull()) {
                return;
            }

            String chapterId = after.path("id").asText();
            String seriesId = after.path("series_id").asText();
            String chapterNum = after.path("chapter_num").asText();
            String title = after.path("title").asText("");
            String seriesTitle = after.path("series_title").asText("");
            String url = after.path("url").asText();
            boolean isNew = after.path("is_new").asBoolean(false);

            if (!isNew) {
                return;
            }

            if (!SecurityUtils.isHttpUrl(url)) {
                log.warn("Rejected chapter event with invalid URL for chapter {}", chapterId);
                return;
            }

            if (!chapterRepo.existsNewChapter(chapterId)) {
                log.warn("Rejected chapter event for unknown or already-notified chapter {}", chapterId);
                return;
            }

            String storedUrl = chapterRepo.findChapterUrl(chapterId);
            if (storedUrl == null || storedUrl.isBlank() || !storedUrl.equals(url)) {
                log.warn("Rejected chapter event with URL mismatch for chapter {}", chapterId);
                return;
            }

            String resolvedTitle = seriesTitle.isEmpty() ? "Unknown" : seriesTitle;
            ChapterNotificationBatcher.PendingChapter pending = new ChapterNotificationBatcher.PendingChapter(
                chapterId, seriesId, resolvedTitle, chapterNum, title, url
            );
            batcher.enqueue(pending, chapters -> deliverChapters(chapters));

        } catch (Exception e) {
            log.error("Failed to process chapter event: {}", e.getMessage(), e);
        }
    }

    private void deliverChapters(List<ChapterNotificationBatcher.PendingChapter> chapters) {
        if (chapters.isEmpty()) {
            return;
        }
        if (chapters.size() == 1) {
            deliverSingle(chapters.get(0));
            return;
        }
        deliverBatch(chapters);
    }

    private void deliverSingle(ChapterNotificationBatcher.PendingChapter chapter) {
        Map<String, Boolean> results = notifierRegistry.sendAll(
            chapter.seriesTitle(), chapter.chapterNum(), chapter.title(), chapter.url()
        );
        recordResults(chapter.chapterId(), chapter.seriesTitle(), chapter.chapterNum(), results);
    }

    private void deliverBatch(List<ChapterNotificationBatcher.PendingChapter> chapters) {
        chapters.sort(Comparator.comparingDouble(ch -> parseChapterNum(ch.chapterNum())));

        ChapterNotificationBatcher.PendingChapter first = chapters.get(0);
        ChapterNotificationBatcher.PendingChapter last = chapters.get(chapters.size() - 1);
        String rangeLabel = formatRangeLabel(first.chapterNum(), last.chapterNum());
        String url = last.url();

        Map<String, Boolean> results = notifierRegistry.sendMassRelease(
            first.seriesTitle(), rangeLabel, chapters.size(), url
        );

        for (ChapterNotificationBatcher.PendingChapter chapter : chapters) {
            recordResults(chapter.chapterId(), chapter.seriesTitle(), rangeLabel, results);
        }
    }

    private void recordResults(String chapterId, String seriesTitle, String chapterLabel, Map<String, Boolean> results) {
        boolean anySuccess = false;
        for (Map.Entry<String, Boolean> entry : results.entrySet()) {
            String channel = entry.getKey();
            boolean success = entry.getValue();
            String status = success ? "SENT" : "FAILED";
            String error = success ? null : "Webhook returned error";
            chapterRepo.logNotification(chapterId, status, channel, error);
            recordDelivery(channel, status);
            if (success) {
                anySuccess = true;
            }
            try {
                NotificationLogEntry logEntry = notificationLogRepo.findRecentForChapterAndChannel(
                    UUID.fromString(chapterId), channel
                );
                sseEmitterService.publishLog(logEntry);
            } catch (Exception ex) {
                log.error("Failed to publish log event to SSE: {}", ex.getMessage());
            }
        }

        if (anySuccess) {
            chapterRepo.markNotified(chapterId);
        }

        log.info("Processed chapter {} for series {}: {} channel(s)",
            chapterLabel, seriesTitle, results.size());
    }

    private static double parseChapterNum(String chapterNum) {
        try {
            return Double.parseDouble(chapterNum);
        } catch (NumberFormatException ex) {
            return 0;
        }
    }

    static String formatRangeLabel(String minChapter, String maxChapter) {
        if (minChapter.equals(maxChapter)) {
            return minChapter;
        }
        return minChapter + "–" + maxChapter;
    }

    private void recordDelivery(String channel, String status) {
        meterRegistry.counter("notification_deliveries_total", "channel", channel, "status", status)
                .increment();
    }
}
