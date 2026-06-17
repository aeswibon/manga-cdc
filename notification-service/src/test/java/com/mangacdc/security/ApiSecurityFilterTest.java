package com.mangacdc.security;

import com.mangacdc.config.MutationGuard;
import com.mangacdc.config.SecurityProperties;
import com.mangacdc.controller.MangaApiController;
import com.mangacdc.controller.WebhookController;
import com.mangacdc.repository.ChapterRepository;
import com.mangacdc.repository.SeriesRepository;
import com.mangacdc.service.ChapterEventService;
import com.mangacdc.service.ScheduleHintService;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.http.MediaType;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.test.context.TestPropertySource;
import org.springframework.context.annotation.Import;
import org.springframework.test.context.TestPropertySource;
import org.springframework.test.web.servlet.MockMvc;

import java.util.Map;

import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyInt;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@WebMvcTest(controllers = {MangaApiController.class, WebhookController.class})
@Import({ApiSecurityFilter.class, SecurityProperties.class})
@TestPropertySource(properties = {
        "cdc.enabled=false",
        "spring.kafka.listener.auto-startup=false",
        "security.require-api-key=true",
        "API_READ_KEY=read-key",
        "security.require-webhook-auth=true",
        "WEBHOOK_SECRET=hook-secret"
})
class ApiSecurityFilterTest {

    @Autowired
    private MockMvc mockMvc;

    @MockBean
    private SeriesRepository seriesRepository;

    @MockBean
    private ChapterRepository chapterRepository;

    @MockBean(name = "readJdbcTemplate")
    private JdbcTemplate readJdbcTemplate;

    @MockBean
    private MutationGuard mutationGuard;

    @MockBean
    private ScheduleHintService scheduleHintService;

    @MockBean
    private ChapterEventService chapterEventService;

    @MockBean
    private QStashSignatureVerifier qstashSignatureVerifier;

    @MockBean
    private InMemoryRateLimiter rateLimiter;

    @Test
    void readApi_requiresApiKey() throws Exception {
        when(rateLimiter.allow(anyString(), anyInt(), anyInt())).thenReturn(true);
        when(readJdbcTemplate.queryForMap(anyString())).thenReturn(
                Map.of("total_series", 0, "active_series", 0, "total_chapters", 0, "total_logs", 0,
                        "successful_deliveries", 0, "failed_deliveries", 0)
        );

        mockMvc.perform(get("/api/stats"))
                .andExpect(status().isUnauthorized());

        mockMvc.perform(get("/api/stats").header("X-Api-Key", "read-key"))
                .andExpect(status().isOk());
    }

    @Test
    void webhook_requiresSharedSecret() throws Exception {
        when(rateLimiter.allow(anyString(), anyInt(), anyInt())).thenReturn(true);

        mockMvc.perform(post("/api/webhook")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"op\":\"c\"}"))
                .andExpect(status().isUnauthorized());

        mockMvc.perform(post("/api/webhook")
                        .header("X-Webhook-Secret", "hook-secret")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"op\":\"c\"}"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$").value("OK"));
    }

    @Test
    void healthEndpoint_staysPublic() throws Exception {
        mockMvc.perform(get("/actuator/health"))
                .andExpect(status().isNotFound());
    }
}
