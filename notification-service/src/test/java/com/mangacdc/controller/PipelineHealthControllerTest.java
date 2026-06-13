package com.mangacdc.controller;

import com.mangacdc.service.PipelineHealthService;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.test.web.servlet.MockMvc;

import java.util.List;
import java.util.Map;

import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@WebMvcTest(PipelineHealthController.class)
class PipelineHealthControllerTest {

    @Autowired
    private MockMvc mockMvc;

    @MockBean
    private PipelineHealthService pipelineHealthService;

    @Test
    void health_returnsPipelineStatus() throws Exception {
        when(pipelineHealthService.buildHealth()).thenReturn(Map.of(
                "status", "operational",
                "updatedAt", "2026-06-13T00:00:00Z",
                "components", List.of(Map.of(
                        "name", "notifier",
                        "status", "operational",
                        "detail", "Notification API responding"
                ))
        ));

        mockMvc.perform(get("/api/pipeline/health"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.status").value("operational"))
                .andExpect(jsonPath("$.components[0].name").value("notifier"));
    }
}
