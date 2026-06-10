package com.mangacdc.service;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.mangacdc.repository.ChapterRepository;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.jdbc.core.JdbcTemplate;

import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

class KafkaChapterConsumerTest {

    private ObjectMapper mapper;

    @BeforeEach
    void setUp() {
        mapper = new ObjectMapper();
    }

    private String cdcEvent(String op, String id, String seriesId, String chapterNum, String title, String url, boolean isNew) {
        try {
            var after = mapper.createObjectNode();
            after.put("id", id);
            after.put("series_id", seriesId);
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

    @Test
    void onChapterEvent_shouldSkipWhenCdcDisabled() {
        KafkaChapterConsumer consumer = new KafkaChapterConsumer(
            mock(DiscordNotifier.class), mock(ChapterRepository.class), mock(JdbcTemplate.class), false);
        consumer.onChapterEvent("{}");
    }

    @Test
    void onChapterEvent_shouldSkipNonCreateOps() {
        DiscordNotifier notifier = mock(DiscordNotifier.class);
        ChapterRepository repo = mock(ChapterRepository.class);
        JdbcTemplate jdbc = mock(JdbcTemplate.class);
        KafkaChapterConsumer consumer = new KafkaChapterConsumer(notifier, repo, jdbc, true);

        consumer.onChapterEvent(cdcEvent("r", "ch1", "s1", "1", "Title", "https://ex.com", true));
        consumer.onChapterEvent(cdcEvent("u", "ch1", "s1", "1", "Title", "https://ex.com", true));
        consumer.onChapterEvent(cdcEvent("d", "ch1", "s1", "1", "Title", "https://ex.com", true));
        verifyNoInteractions(notifier, repo, jdbc);
    }

    @Test
    void onChapterEvent_shouldSkipWhenIsNewIsFalse() {
        KafkaChapterConsumer consumer = new KafkaChapterConsumer(
            mock(DiscordNotifier.class), mock(ChapterRepository.class), mock(JdbcTemplate.class), true);
        consumer.onChapterEvent(cdcEvent("c", "ch1", "s1", "1", "Title", "https://ex.com", false));
    }

    @Test
    void onChapterEvent_shouldSkipWhenAfterIsNull() {
        KafkaChapterConsumer consumer = new KafkaChapterConsumer(
            mock(DiscordNotifier.class), mock(ChapterRepository.class), mock(JdbcTemplate.class), true);
        consumer.onChapterEvent("{\"op\":\"c\"}");
    }

    @Test
    void onChapterEvent_shouldProcessNewChapterSuccessfully() {
        DiscordNotifier notifier = mock(DiscordNotifier.class);
        ChapterRepository repo = mock(ChapterRepository.class);
        JdbcTemplate jdbc = mock(JdbcTemplate.class);
        when(jdbc.queryForObject(anyString(), eq(String.class), anyString()))
            .thenReturn("One Piece");
        when(notifier.sendChapterAlert(anyString(), anyString(), anyString(), anyString()))
            .thenReturn(true);

        KafkaChapterConsumer consumer = new KafkaChapterConsumer(notifier, repo, jdbc, true);
        consumer.onChapterEvent(cdcEvent("c", "ch1", "s1", "1100", "The Final Chapter", "https://ex.com/ch/1100", true));

        verify(notifier).sendChapterAlert("One Piece", "1100", "The Final Chapter", "https://ex.com/ch/1100");
        verify(repo).logNotification("ch1", "SENT", "discord", null);
        verify(repo).markNotified("ch1");
    }

    @Test
    void onChapterEvent_shouldHandleUnknownSeriesTitle() {
        DiscordNotifier notifier = mock(DiscordNotifier.class);
        ChapterRepository repo = mock(ChapterRepository.class);
        JdbcTemplate jdbc = mock(JdbcTemplate.class);
        when(jdbc.queryForObject(anyString(), eq(String.class), anyString()))
            .thenReturn(null);
        when(notifier.sendChapterAlert(anyString(), anyString(), anyString(), anyString()))
            .thenReturn(true);

        KafkaChapterConsumer consumer = new KafkaChapterConsumer(notifier, repo, jdbc, true);
        consumer.onChapterEvent(cdcEvent("c", "ch1", "s1", "1", "", "https://ex.com", true));

        verify(notifier).sendChapterAlert("Unknown", "1", "", "https://ex.com");
        verify(repo).logNotification("ch1", "SENT", "discord", null);
    }

    @Test
    void onChapterEvent_shouldLogFailedNotification() {
        DiscordNotifier notifier = mock(DiscordNotifier.class);
        ChapterRepository repo = mock(ChapterRepository.class);
        JdbcTemplate jdbc = mock(JdbcTemplate.class);
        when(jdbc.queryForObject(anyString(), eq(String.class), anyString()))
            .thenReturn("Naruto");
        when(notifier.sendChapterAlert(anyString(), anyString(), anyString(), anyString()))
            .thenReturn(false);

        KafkaChapterConsumer consumer = new KafkaChapterConsumer(notifier, repo, jdbc, true);
        consumer.onChapterEvent(cdcEvent("c", "ch1", "s1", "700", "", "https://ex.com", true));

        verify(repo).logNotification("ch1", "FAILED", "discord", "Webhook returned error");
        verify(repo, never()).markNotified(anyString());
    }

    @Test
    void onChapterEvent_shouldHandleJdbcException() {
        DiscordNotifier notifier = mock(DiscordNotifier.class);
        ChapterRepository repo = mock(ChapterRepository.class);
        JdbcTemplate jdbc = mock(JdbcTemplate.class);
        when(jdbc.queryForObject(anyString(), eq(String.class), anyString()))
            .thenThrow(new RuntimeException("DB connection lost"));

        KafkaChapterConsumer consumer = new KafkaChapterConsumer(notifier, repo, jdbc, true);
        consumer.onChapterEvent(cdcEvent("c", "ch1", "s1", "1", "", "https://ex.com", true));

        verify(notifier, never()).sendChapterAlert(anyString(), anyString(), anyString(), anyString());
        verify(repo, never()).logNotification(anyString(), anyString(), anyString(), anyString());
    }

    @Test
    void onChapterEvent_shouldHandleMalformedJson() {
        KafkaChapterConsumer consumer = new KafkaChapterConsumer(
            mock(DiscordNotifier.class), mock(ChapterRepository.class), mock(JdbcTemplate.class), true);
        consumer.onChapterEvent("{invalid json}");
    }
}
