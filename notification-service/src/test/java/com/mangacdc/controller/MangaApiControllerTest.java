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
import org.springframework.test.web.servlet.MockMvc;

import org.springframework.web.server.ResponseStatusException;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@WebMvcTest(MangaApiController.class)
@AutoConfigureMockMvc(addFilters = false)
@Import({MutationConfig.class, MutationGuard.class, WebMvcSecurityTestSupport.class})
class MangaApiControllerTest {

    @Autowired
    private MockMvc mockMvc;

    @MockBean
    private SeriesRepository seriesRepository;

    @MockBean
    private ChapterRepository chapterRepository;

    @MockBean
    private JdbcTemplate jdbcTemplate;

    @Test
    void addSeries_rejectsInvalidPayloadWhenMutationsDisabled() throws Exception {
        mockMvc.perform(post("/api/series")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "sourceId": "md-1",
                                  "title": "404",
                                  "status": "ONGOING",
                                  "sourceUrl": "https://example.com/title/1"
                                }
                                """))
                .andExpect(status().isForbidden());

        verify(seriesRepository, never()).save(any(MangaSeries.class));
    }

    @Test
    void addSeries_returnsForbiddenWhenMutationsDisabled() throws Exception {
        mockMvc.perform(post("/api/series")
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
                .andExpect(status().isForbidden())
                .andExpect(result -> {
                    Exception ex = result.getResolvedException();
                    assertThat(ex).isInstanceOf(ResponseStatusException.class);
                    assertThat(((ResponseStatusException) ex).getReason())
                            .contains("Admin mutations are disabled");
                });

        verify(seriesRepository, never()).save(any(MangaSeries.class));
    }
}
