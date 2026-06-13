package com.mangacdc.service;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.web.client.RestTemplate;

import java.util.List;
import java.util.Map;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
class PipelineHealthServiceTest {

    @Mock
    private JdbcTemplate jdbc;

    @Mock
    private RestTemplate healthCheckRestTemplate;

    private PipelineHealthService service;

    @BeforeEach
    void setUp() {
        service = new PipelineHealthService(jdbc, healthCheckRestTemplate, null, "", "", false);
    }

    @Test
    void buildHealth_reportsOperationalWhenDatabaseIsHealthy() {
        when(jdbc.queryForObject(anyString(), eq(Integer.class))).thenReturn(1);

        Map<String, Object> health = service.buildHealth();

        assertThat(health.get("status")).isEqualTo("operational");
        @SuppressWarnings("unchecked")
        List<Map<String, Object>> components = (List<Map<String, Object>>) health.get("components");
        assertThat(components).extracting(component -> component.get("name"))
                .contains("notifier", "database", "kafka");
    }

    @Test
    void buildHealth_reportsDownWhenDatabaseFails() {
        when(jdbc.queryForObject(anyString(), eq(Integer.class))).thenThrow(new RuntimeException("connection refused"));

        Map<String, Object> health = service.buildHealth();

        assertThat(health.get("status")).isEqualTo("down");
    }
}
