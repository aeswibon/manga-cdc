package com.mangacdc.controller;

import com.mangacdc.config.MutationConfig;
import com.mangacdc.config.MutationGuard;
import com.mangacdc.model.NotificationLogEntry;
import com.mangacdc.repository.NotificationLogRepository;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.context.annotation.Import;
import org.springframework.test.web.servlet.MockMvc;

import java.math.BigDecimal;
import java.time.OffsetDateTime;
import java.util.List;
import java.util.UUID;

import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@WebMvcTest(NotificationLogController.class)
@Import({MutationConfig.class, MutationGuard.class})
class NotificationLogControllerTest {

    @Autowired
    private MockMvc mockMvc;

    @MockBean
    private NotificationLogRepository notificationLogRepository;

    @MockBean
    private com.mangacdc.service.SseEmitterService sseEmitterService;

    @MockBean
    private java.util.List<com.mangacdc.service.Notifier> notifiers;

    @Test
    void listLogs_returnsRecentEntries() throws Exception {
        UUID logId = UUID.fromString("00000000-0000-0000-0000-000000000101");
        UUID chapterId = UUID.fromString("00000000-0000-0000-0000-000000000201");
        NotificationLogEntry entry = new NotificationLogEntry(
                logId,
                chapterId,
                "SENT",
                "discord",
                null,
                null,
                OffsetDateTime.parse("2026-06-11T18:00:00Z"),
                "One Piece",
                new BigDecimal("1100"),
                "The Final Chapter",
                "https://mangadex.org/chapter/1"
        );
        when(notificationLogRepository.findRecent(50)).thenReturn(List.of(entry));

        mockMvc.perform(get("/api/logs"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$[0].status").value("SENT"))
                .andExpect(jsonPath("$[0].channel").value("discord"))
                .andExpect(jsonPath("$[0].seriesTitle").value("One Piece"));

        verify(notificationLogRepository).findRecent(50);
    }

    @Test
    void listLogs_honorsLimitParameter() throws Exception {
        when(notificationLogRepository.findRecent(10)).thenReturn(List.of());

        mockMvc.perform(get("/api/logs").param("limit", "10"))
                .andExpect(status().isOk());

        verify(notificationLogRepository).findRecent(eq(10));
    }

    @Test
    void listLogs_capsLimitAt100() throws Exception {
        when(notificationLogRepository.findRecent(100)).thenReturn(List.of());

        mockMvc.perform(get("/api/logs").param("limit", "500"))
                .andExpect(status().isOk());

        verify(notificationLogRepository).findRecent(eq(100));
    }
}
