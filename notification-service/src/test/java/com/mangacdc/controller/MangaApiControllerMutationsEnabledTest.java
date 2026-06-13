package com.mangacdc.controller;

import com.mangacdc.config.MutationConfig;
import com.mangacdc.config.MutationGuard;
import com.mangacdc.support.WebMvcSecurityTestSupport;
import com.mangacdc.model.MangaSeries;
import com.mangacdc.repository.ChapterRepository;
import com.mangacdc.repository.SeriesRepository;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.context.annotation.Import;
import org.springframework.http.MediaType;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.test.context.TestPropertySource;
import org.springframework.test.web.servlet.MockMvc;

import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@WebMvcTest(MangaApiController.class)
@AutoConfigureMockMvc(addFilters = false)
@Import({MutationConfig.class, MutationGuard.class, WebMvcSecurityTestSupport.class})
@TestPropertySource(properties = {
        "ADMIN_MUTATIONS_ENABLED=true",
        "ADMIN_API_KEY=test-admin-key"
})
class MangaApiControllerMutationsEnabledTest {

    @Autowired
    private MockMvc mockMvc;

    @MockBean
    private SeriesRepository seriesRepository;

    @MockBean
    private ChapterRepository chapterRepository;

    @MockBean
    private JdbcTemplate jdbcTemplate;

    @Test
    void addSeries_rejectsInvalidPayload() throws Exception {
        mockMvc.perform(post("/api/series")
                        .header("X-Admin-Key", "test-admin-key")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "sourceId": "md-1",
                                  "title": "404",
                                  "status": "ONGOING",
                                  "sourceUrl": "https://example.com/title/1"
                                }
                                """))
                .andExpect(status().isBadRequest());

        verify(seriesRepository, never()).save(any(MangaSeries.class));
    }

    @Test
    void addSeries_acceptsValidPayload() throws Exception {
        mockMvc.perform(post("/api/series")
                        .header("X-Admin-Key", "test-admin-key")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "sourceId": "md-1",
                                  "title": "One Piece",
                                  "status": "ONGOING",
                                  "sourceUrl": "https://example.com/title/1",
                                  "coverUrl": "https://example.com/cover.jpg"
                                }
                                """))
                .andExpect(status().isOk());

        verify(seriesRepository).save(any(MangaSeries.class));
    }
}
