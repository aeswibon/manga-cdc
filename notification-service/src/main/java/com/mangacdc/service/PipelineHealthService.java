package com.mangacdc.service;

import org.apache.kafka.clients.admin.AdminClient;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.kafka.core.KafkaAdmin;
import org.springframework.stereotype.Service;
import org.springframework.web.client.RestClientException;
import org.springframework.web.client.RestTemplate;

import java.time.Instant;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.TimeUnit;

@Service
public class PipelineHealthService {

    private final JdbcTemplate jdbc;
    private final RestTemplate healthCheckRestTemplate;
    private final KafkaAdmin kafkaAdmin;
    private final String scraperHealthUrl;
    private final String scraperReadyUrl;
    private final boolean cdcEnabled;

    public PipelineHealthService(
            JdbcTemplate jdbc,
            @Qualifier("healthCheckRestTemplate") RestTemplate healthCheckRestTemplate,
            @Autowired(required = false) KafkaAdmin kafkaAdmin,
            @Value("${pipeline.scraper-health-url:}") String scraperHealthUrl,
            @Value("${pipeline.scraper-ready-url:}") String scraperReadyUrl,
            @Value("${cdc.enabled:false}") boolean cdcEnabled) {
        this.jdbc = jdbc;
        this.healthCheckRestTemplate = healthCheckRestTemplate;
        this.kafkaAdmin = kafkaAdmin;
        this.scraperHealthUrl = scraperHealthUrl == null ? "" : scraperHealthUrl.trim();
        this.scraperReadyUrl = scraperReadyUrl == null ? "" : scraperReadyUrl.trim();
        this.cdcEnabled = cdcEnabled;
    }

    public Map<String, Object> buildHealth() {
        List<Map<String, Object>> components = new ArrayList<>();
        components.add(notifierComponent());
        components.add(checkDatabase());
        components.add(checkKafka());
        if (!scraperHealthUrl.isBlank()) {
            components.add(checkScraper());
        }

        String overall = summarize(components);
        Map<String, Object> payload = new LinkedHashMap<>();
        payload.put("status", overall);
        payload.put("updatedAt", Instant.now().toString());
        payload.put("components", components);
        return payload;
    }

    private Map<String, Object> notifierComponent() {
        Map<String, Object> component = baseComponent("notifier");
        component.put("status", "operational");
        component.put("detail", "Notification API responding");
        return component;
    }

    private Map<String, Object> checkDatabase() {
        Map<String, Object> component = baseComponent("database");
        try {
            Integer result = jdbc.queryForObject("SELECT 1", Integer.class);
            if (result != null && result == 1) {
                component.put("status", "operational");
                component.put("detail", "PostgreSQL reachable");
            } else {
                component.put("status", "down");
                component.put("detail", "Unexpected database response");
            }
        } catch (Exception ex) {
            component.put("status", "down");
            component.put("detail", "Database check failed");
        }
        return component;
    }

    private Map<String, Object> checkKafka() {
        Map<String, Object> component = baseComponent("kafka");
        if (!cdcEnabled) {
            component.put("status", "operational");
            component.put("detail", "CDC disabled");
            return component;
        }
        if (kafkaAdmin == null) {
            component.put("status", "degraded");
            component.put("detail", "CDC enabled but Kafka is not configured");
            return component;
        }
        try (AdminClient admin = AdminClient.create(kafkaAdmin.getConfigurationProperties())) {
            admin.describeCluster().clusterId().get(3, TimeUnit.SECONDS);
            component.put("status", "operational");
            component.put("detail", "Broker reachable");
        } catch (Exception ex) {
            component.put("status", "degraded");
            component.put("detail", "Kafka check failed");
        }
        return component;
    }

    private Map<String, Object> checkScraper() {
        Map<String, Object> component = baseComponent("scraper");
        String readyUrl = scraperReadyUrl.isBlank() ? scraperHealthUrl : scraperReadyUrl;
        try {
            var response = healthCheckRestTemplate.getForEntity(readyUrl, Map.class);
            if (response.getStatusCode().is2xxSuccessful()) {
                component.put("status", "operational");
                Object bodyStatus = response.getBody() != null ? response.getBody().get("status") : null;
                component.put("detail", bodyStatus != null ? bodyStatus.toString() : "Reachable");
            } else {
                component.put("status", "degraded");
                component.put("detail", "HTTP " + response.getStatusCode().value());
            }
        } catch (RestClientException ex) {
            if (!scraperHealthUrl.equals(readyUrl)) {
                try {
                    healthCheckRestTemplate.getForEntity(scraperHealthUrl, Map.class);
                    component.put("status", "degraded");
                    component.put("detail", "Liveness OK, readiness failed");
                    return component;
                } catch (RestClientException ignored) {
                    // fall through
                }
            }
            component.put("status", "down");
            component.put("detail", "Scraper check failed");
        }
        return component;
    }

    private Map<String, Object> baseComponent(String name) {
        Map<String, Object> component = new LinkedHashMap<>();
        component.put("name", name);
        component.put("status", "unknown");
        component.put("detail", "");
        return component;
    }

    private String summarize(List<Map<String, Object>> components) {
        boolean anyDown = components.stream().anyMatch(c -> "down".equals(c.get("status")));
        if (anyDown) {
            return "down";
        }
        boolean anyDegraded = components.stream().anyMatch(c -> "degraded".equals(c.get("status")));
        if (anyDegraded) {
            return "degraded";
        }
        return "operational";
    }
}
