package com.mangacdc;

import com.fasterxml.jackson.databind.ObjectMapper;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.core.io.ClassPathResource;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.test.context.DynamicPropertyRegistry;
import org.springframework.test.context.DynamicPropertySource;
import org.springframework.web.client.RestTemplate;
import org.testcontainers.containers.KafkaContainer;
import org.testcontainers.containers.PostgreSQLContainer;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;
import org.testcontainers.utility.DockerImageName;

import java.time.Duration;
import java.util.UUID;

import static org.awaitility.Awaitility.await;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

@Testcontainers
@SpringBootTest(webEnvironment = SpringBootTest.WebEnvironment.NONE)
class CdcPipelineIntegrationTest {

    @Container
    private static final KafkaContainer kafka = new KafkaContainer(
        DockerImageName.parse("confluentinc/cp-kafka:7.6.1")
    )
        .withStartupTimeout(Duration.ofSeconds(60));

    @Container
    private static final PostgreSQLContainer<?> postgres = new PostgreSQLContainer<>(
        "postgres:16-alpine"
    )
        .withDatabaseName("mangacdc")
        .withUsername("mangacdc")
        .withPassword("mangacdc")
        .withStartupTimeout(Duration.ofSeconds(60));

    @DynamicPropertySource
    static void properties(DynamicPropertyRegistry registry) {
        registry.add("spring.datasource.url", postgres::getJdbcUrl);
        registry.add("spring.datasource.username", postgres::getUsername);
        registry.add("spring.datasource.password", postgres::getPassword);
        registry.add("spring.kafka.bootstrap-servers", kafka::getBootstrapServers);
        registry.add("cdc.enabled", () -> "true");
        registry.add("discord.webhook-url", () -> "http://localhost:9999/mock-webhook");
    }

    @MockBean
    private RestTemplate restTemplate;

    @Autowired
    private KafkaTemplate<String, String> kafkaTemplate;

    @Autowired
    private JdbcTemplate jdbc;

    private final ObjectMapper mapper = new ObjectMapper();
    private boolean migrated;

    @BeforeEach
    void setUp() throws Exception {
        if (migrated) {
            return;
        }
        var resource = new ClassPathResource("migration.sql");
        var sql = new String(resource.getInputStream().readAllBytes());
        for (var statement : sql.split(";")) {
            var trimmed = statement.trim();
            if (!trimmed.isEmpty()) {
                try {
                    jdbc.execute(trimmed);
                } catch (Exception e) {
                    if (!e.getMessage().contains("already exists")) {
                        throw e;
                    }
                }
            }
        }
        migrated = true;
    }

    @Test
    void fullPipelineDiscordSuccess() throws Exception {
        when(restTemplate.postForEntity(anyString(), any(), eq(String.class)))
            .thenReturn(null);

        String seriesId = UUID.randomUUID().toString();
        jdbc.update(
            "INSERT INTO manga_series (id, source_id, title, source_url, status, is_active) VALUES (?::uuid, ?, ?, ?, 'ONGOING', true)",
            seriesId, "test-source", "Test Series", "https://example.com/test"
        );

        String chapterId = UUID.randomUUID().toString();
        jdbc.update(
            "INSERT INTO chapters (id, series_id, chapter_num, title, url, is_new) VALUES (?::uuid, ?::uuid, ?, ?, ?, true)",
            chapterId, seriesId, 1, "Chapter 1", "https://example.com/ch-1"
        );

        var after = mapper.createObjectNode();
        after.put("id", chapterId);
        after.put("series_id", seriesId);
        after.put("chapter_num", "1");
        after.put("title", "Chapter 1");
        after.put("url", "https://example.com/ch-1");
        after.put("is_new", true);

        var root = mapper.createObjectNode();
        root.put("op", "c");
        root.set("after", after);

        kafkaTemplate.send("mangacdc.public.chapters", mapper.writeValueAsString(root)).get();

        await().atMost(Duration.ofSeconds(10)).untilAsserted(() ->
            verify(restTemplate, atLeastOnce())
                .postForEntity(anyString(), any(), eq(String.class))
        );

        String logStatus = jdbc.queryForObject(
            "SELECT status FROM notification_logs WHERE chapter_id = ?::uuid",
            String.class, chapterId);
        assert "SENT".equals(logStatus) : "Expected SENT, got " + logStatus;

        Boolean isNew = jdbc.queryForObject(
            "SELECT is_new FROM chapters WHERE id = ?::uuid",
            Boolean.class, chapterId);
        assert Boolean.FALSE.equals(isNew) : "Expected is_new=false";
    }

    @Test
    void fullPipelineDiscordFailure() throws Exception {
        when(restTemplate.postForEntity(anyString(), any(), eq(String.class)))
            .thenThrow(new RuntimeException("Webhook unavailable"));

        String seriesId = UUID.randomUUID().toString();
        jdbc.update(
            "INSERT INTO manga_series (id, source_id, title, source_url, status, is_active) VALUES (?::uuid, ?, ?, ?, 'ONGOING', true)",
            seriesId, "test-source-2", "Test Series 2", "https://example.com/test2"
        );

        String chapterId = UUID.randomUUID().toString();
        jdbc.update(
            "INSERT INTO chapters (id, series_id, chapter_num, title, url, is_new) VALUES (?::uuid, ?::uuid, ?, ?, ?, true)",
            chapterId, seriesId, 1, "Chapter 1", "https://example.com/ch-1"
        );

        var after = mapper.createObjectNode();
        after.put("id", chapterId);
        after.put("series_id", seriesId);
        after.put("chapter_num", "1");
        after.put("title", "Chapter 1");
        after.put("url", "https://example.com/ch-1");
        after.put("is_new", true);

        var root = mapper.createObjectNode();
        root.put("op", "c");
        root.set("after", after);

        kafkaTemplate.send("mangacdc.public.chapters", mapper.writeValueAsString(root)).get();

        await().atMost(Duration.ofSeconds(10)).untilAsserted(() -> {
            String status = jdbc.queryForObject(
                "SELECT status FROM notification_logs WHERE chapter_id = ?::uuid",
                String.class, chapterId);
            assert "FAILED".equals(status) : "Expected FAILED, got " + status;
        });

        Boolean isNew = jdbc.queryForObject(
            "SELECT is_new FROM chapters WHERE id = ?::uuid",
            Boolean.class, chapterId);
        assert Boolean.TRUE.equals(isNew) : "Expected is_new=true (not marked notified)";
    }
}
