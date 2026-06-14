package com.mangacdc.service;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.mangacdc.config.NotificationProperties;
import com.mangacdc.model.Chapter;
import com.mangacdc.model.SeriesNotificationPrefs;
import com.mangacdc.repository.ChapterRepository;
import io.micrometer.core.instrument.simple.SimpleMeterRegistry;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.time.Instant;
import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

class ChapterEventServiceTest {
 
    private ObjectMapper mapper;
    private SimpleMeterRegistry meterRegistry;
    private com.mangacdc.repository.NotificationLogRepository notificationLogRepo;
    private SseEmitterService sseEmitterService;
    private SeriesNotificationPrefsService prefsService;
 
    @BeforeEach
    void setUp() {
        mapper = new ObjectMapper();
        meterRegistry = new SimpleMeterRegistry();
        notificationLogRepo = mock(com.mangacdc.repository.NotificationLogRepository.class);
        sseEmitterService = mock(SseEmitterService.class);
        prefsService = mock(SeriesNotificationPrefsService.class);
        when(prefsService.getPrefs(anyString())).thenReturn(SeriesNotificationPrefs.empty());
    }
 
    private ChapterEventService newService(NotifierRegistry registry, ChapterRepository repo) {
        ChapterNotificationBatcher batcher = new ChapterNotificationBatcher(new NotificationProperties(0));
        return new ChapterEventService(registry, repo, notificationLogRepo, sseEmitterService, meterRegistry, batcher, prefsService);
    }

    private String cdcEvent(String op, String id, String seriesId, String seriesTitle, String chapterNum, String title, String url, boolean isNew) {
        try {
            var after = mapper.createObjectNode();
            after.put("id", id);
            after.put("series_id", seriesId);
            if (seriesTitle != null) after.put("series_title", seriesTitle);
            after.put("chapter_num", chapterNum);
            after.put("title", title);
            after.put("url", url);
            after.put("is_new", isNew);

            var root = mapper.createObjectNode();
            root.put("op", op);
            root.set("after", after);
            return mapper.writeValueAsString(root);
        } catch (Exception e) {
            throw new RuntimeException(e);
        }
    }

    private String cdcEvent(String op, String id, String seriesId, String chapterNum, String title, String url, boolean isNew) {
        return cdcEvent(op, id, seriesId, null, chapterNum, title, url, isNew);
    }

    @Test
    void processChapterEvent_shouldSkipNonCreateOps() {
        NotifierRegistry registry = mock(NotifierRegistry.class);
        ChapterRepository repo = mock(ChapterRepository.class);
        ChapterEventService service = newService(registry, repo);

        service.processChapterEvent(cdcEvent("r", "ch1", "s1", "1", "Title", "https://ex.com", true));
        service.processChapterEvent(cdcEvent("u", "ch1", "s1", "1", "Title", "https://ex.com", true));
        service.processChapterEvent(cdcEvent("d", "ch1", "s1", "1", "Title", "https://ex.com", true));
        verifyNoInteractions(registry, repo);
    }

    @Test
    void processChapterEvent_shouldSkipWhenIsNewIsFalse() {
        NotifierRegistry registry = mock(NotifierRegistry.class);
        ChapterRepository repo = mock(ChapterRepository.class);
        ChapterEventService service = newService(registry, repo);

        service.processChapterEvent(cdcEvent("c", "ch1", "s1", "1", "Title", "https://ex.com", false));
        verifyNoInteractions(registry, repo);
    }

    @Test
    void processChapterEvent_shouldSkipWhenAfterIsNull() {
        NotifierRegistry registry = mock(NotifierRegistry.class);
        ChapterRepository repo = mock(ChapterRepository.class);
        ChapterEventService service = newService(registry, repo);

        service.processChapterEvent("{\"op\":\"c\"}");
        verifyNoInteractions(registry, repo);
    }

    @Test
    void processChapterEvent_shouldProcessNewChapterSuccessfully() {
        NotifierRegistry registry = mock(NotifierRegistry.class);
        ChapterRepository repo = mock(ChapterRepository.class);
        when(repo.existsNewChapter("ch1")).thenReturn(true);
        when(repo.findChapterUrl("ch1")).thenReturn("https://ex.com/ch/1100");
        when(registry.sendAll(anyString(), anyString(), anyString(), anyString()))
            .thenReturn(Map.of("discord", true));

        ChapterEventService service = newService(registry, repo);
        service.processChapterEvent(cdcEvent("c", "ch1", "s1", "One Piece", "1100", "The Final Chapter", "https://ex.com/ch/1100", true));

        verify(registry).sendAll("One Piece", "1100", "The Final Chapter", "https://ex.com/ch/1100");
        verify(repo).logNotification("ch1", "SENT", "discord", null);
        verify(repo).markNotified("ch1");
    }

    @Test
    void processChapterEvent_shouldHandleMissingSeriesTitle() {
        NotifierRegistry registry = mock(NotifierRegistry.class);
        ChapterRepository repo = mock(ChapterRepository.class);
        when(repo.existsNewChapter("ch1")).thenReturn(true);
        when(repo.findChapterUrl("ch1")).thenReturn("https://ex.com");
        when(registry.sendAll(anyString(), anyString(), anyString(), anyString()))
            .thenReturn(Map.of("discord", true));

        ChapterEventService service = newService(registry, repo);
        service.processChapterEvent(cdcEvent("c", "ch1", "s1", "1", "", "https://ex.com", true));

        verify(registry).sendAll("Unknown", "1", "", "https://ex.com");
        verify(repo).logNotification("ch1", "SENT", "discord", null);
    }

    @Test
    void processChapterEvent_shouldLogFailedNotification() {
        NotifierRegistry registry = mock(NotifierRegistry.class);
        ChapterRepository repo = mock(ChapterRepository.class);
        when(repo.existsNewChapter("ch1")).thenReturn(true);
        when(repo.findChapterUrl("ch1")).thenReturn("https://ex.com");
        when(registry.sendAll(anyString(), anyString(), anyString(), anyString()))
            .thenReturn(Map.of("discord", false));

        ChapterEventService service = newService(registry, repo);
        service.processChapterEvent(cdcEvent("c", "ch1", "s1", "Naruto", "700", "", "https://ex.com", true));

        verify(repo).logNotification("ch1", "FAILED", "discord", "Webhook returned error");
        verify(repo, never()).markNotified(anyString());
    }

    @Test
    void processChapterEvent_shouldHandleMixedResults() {
        NotifierRegistry registry = mock(NotifierRegistry.class);
        ChapterRepository repo = mock(ChapterRepository.class);
        when(repo.existsNewChapter("ch1")).thenReturn(true);
        when(repo.findChapterUrl("ch1")).thenReturn("https://ex.com");
        when(registry.sendAll(anyString(), anyString(), anyString(), anyString()))
            .thenReturn(Map.of("discord", true, "slack", false, "telegram", true));

        ChapterEventService service = newService(registry, repo);
        service.processChapterEvent(cdcEvent("c", "ch1", "s1", "Berserk", "377", "", "https://ex.com", true));

        verify(repo).logNotification("ch1", "SENT", "discord", null);
        verify(repo).logNotification("ch1", "FAILED", "slack", "Webhook returned error");
        verify(repo).logNotification("ch1", "SENT", "telegram", null);
        verify(repo).markNotified("ch1");
    }

    @Test
    void processChapterEvent_allChannelsFailed_shouldNotMarkNotified() {
        NotifierRegistry registry = mock(NotifierRegistry.class);
        ChapterRepository repo = mock(ChapterRepository.class);
        when(repo.existsNewChapter("ch1")).thenReturn(true);
        when(repo.findChapterUrl("ch1")).thenReturn("https://ex.com");
        when(registry.sendAll(anyString(), anyString(), anyString(), anyString()))
            .thenReturn(Map.of("discord", false, "slack", false));

        ChapterEventService service = newService(registry, repo);
        service.processChapterEvent(cdcEvent("c", "ch1", "s1", "Naruto", "700", "", "https://ex.com", true));

        verify(repo, never()).markNotified(anyString());
    }

    @Test
    void processChapterEvent_shouldRecordDeliveryMetrics() {
        NotifierRegistry registry = mock(NotifierRegistry.class);
        ChapterRepository repo = mock(ChapterRepository.class);
        when(repo.existsNewChapter("ch1")).thenReturn(true);
        when(repo.findChapterUrl("ch1")).thenReturn("https://ex.com");
        when(registry.sendAll(anyString(), anyString(), anyString(), anyString()))
            .thenReturn(Map.of("discord", true, "slack", false, "telegram", true));

        ChapterEventService service = newService(registry, repo);
        service.processChapterEvent(cdcEvent("c", "ch1", "s1", "Berserk", "377", "", "https://ex.com", true));

        assertEquals(1.0, meterRegistry.counter("notification_deliveries_total",
            "channel", "discord", "status", "SENT").count());
        assertEquals(1.0, meterRegistry.counter("notification_deliveries_total",
            "channel", "slack", "status", "FAILED").count());
        assertEquals(1.0, meterRegistry.counter("notification_deliveries_total",
            "channel", "telegram", "status", "SENT").count());
    }

    @Test
    void processChapterEvent_skippedEvents_shouldNotRecordDeliveryMetrics() {
        NotifierRegistry registry = mock(NotifierRegistry.class);
        ChapterRepository repo = mock(ChapterRepository.class);
        ChapterEventService service = newService(registry, repo);

        service.processChapterEvent(cdcEvent("r", "ch1", "s1", "1", "Title", "https://ex.com", true));
        service.processChapterEvent(cdcEvent("c", "ch1", "s1", "1", "Title", "https://ex.com", false));

        assertEquals(0, meterRegistry.find("notification_deliveries_total").counters().size());
    }

    @Test
    void processChapterEvent_rejectsUrlMismatch() {
        NotifierRegistry registry = mock(NotifierRegistry.class);
        ChapterRepository repo = mock(ChapterRepository.class);
        when(repo.existsNewChapter("ch1")).thenReturn(true);
        when(repo.findChapterUrl("ch1")).thenReturn("https://ex.com/safe");

        ChapterEventService service = newService(registry, repo);
        service.processChapterEvent(cdcEvent("c", "ch1", "s1", "One Piece", "1100", "Title", "https://evil.example/phish", true));

        verifyNoInteractions(registry);
        verify(repo, never()).logNotification(anyString(), anyString(), anyString(), any());
    }

    @Test
    void formatRangeLabel_formatsSingleAndRange() {
        assertEquals("100", ChapterEventService.formatRangeLabel("100", "100"));
        assertEquals("100–114", ChapterEventService.formatRangeLabel("100", "114"));
    }

    @Test
    void processChapterEvent_shouldHandleMalformedJson() {
        NotifierRegistry registry = mock(NotifierRegistry.class);
        ChapterRepository repo = mock(ChapterRepository.class);
        ChapterEventService service = newService(registry, repo);

        service.processChapterEvent("{invalid json}");
        verifyNoInteractions(registry, repo);
    }

    @Test
    void processChapterEvent_bingeMode_holdsUntilThreshold() {
        NotifierRegistry registry = mock(NotifierRegistry.class);
        ChapterRepository repo = mock(ChapterRepository.class);
        when(repo.existsNewChapter("ch1")).thenReturn(true);
        when(repo.findChapterUrl("ch1")).thenReturn("https://ex.com/1");
        when(prefsService.getPrefs("s1")).thenReturn(new SeriesNotificationPrefs(5));
        when(repo.countNewChaptersForSeries("s1")).thenReturn(3);

        ChapterEventService service = newService(registry, repo);
        service.processChapterEvent(cdcEvent("c", "ch1", "s1", "One Piece", "1", "Title", "https://ex.com/1", true));

        verifyNoInteractions(registry);
        verify(repo, never()).markNotified(anyString());
        verify(repo, never()).logNotification(anyString(), anyString(), anyString(), any());
    }

    @Test
    void processChapterEvent_bingeMode_sendsMassReleaseWhenThresholdMet() {
        NotifierRegistry registry = mock(NotifierRegistry.class);
        ChapterRepository repo = mock(ChapterRepository.class);
        when(repo.existsNewChapter("ch3")).thenReturn(true);
        when(repo.findChapterUrl("ch3")).thenReturn("https://ex.com/3");
        when(prefsService.getPrefs("s1")).thenReturn(new SeriesNotificationPrefs(3));
        when(repo.countNewChaptersForSeries("s1")).thenReturn(3);
        when(repo.findSeriesTitle("s1")).thenReturn("One Piece");
        when(repo.findNewChaptersForSeries("s1")).thenReturn(List.of(
            new Chapter("ch1", "s1", 1.0, "T1", "https://ex.com/1", Instant.now(), true),
            new Chapter("ch2", "s1", 2.0, "T2", "https://ex.com/2", Instant.now(), true),
            new Chapter("ch3", "s1", 3.0, "T3", "https://ex.com/3", Instant.now(), true)
        ));
        when(registry.sendMassRelease(eq("One Piece"), eq("1–3"), eq(3), eq("https://ex.com/3")))
            .thenReturn(Map.of("discord", true));

        ChapterEventService service = newService(registry, repo);
        service.processChapterEvent(cdcEvent("c", "ch3", "s1", "One Piece", "3", "T3", "https://ex.com/3", true));

        verify(registry).sendMassRelease("One Piece", "1–3", 3, "https://ex.com/3");
        verify(repo).markNotified("ch1");
        verify(repo).markNotified("ch2");
        verify(repo).markNotified("ch3");
    }
}
