package com.mangacdc.config;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import java.util.Arrays;
import java.util.List;

@Component
public class SecurityProperties {

    private final boolean requireApiKey;
    private final boolean requireWebhookAuth;
    private final String apiReadKey;
    private final String webhookSecret;
    private final String qstashCurrentSigningKey;
    private final String qstashNextSigningKey;
    private final List<String> allowedOrigins;
    private final int readRateLimitPerMinute;
    private final int webhookRateLimitPerMinute;
    private final int maxSseConnectionsPerIp;

    public SecurityProperties(
            @Value("${security.require-api-key:false}") boolean requireApiKey,
            @Value("${security.require-webhook-auth:true}") boolean requireWebhookAuth,
            @Value("${API_READ_KEY:}") String apiReadKey,
            @Value("${WEBHOOK_SECRET:}") String webhookSecret,
            @Value("${QSTASH_CURRENT_SIGNING_KEY:}") String qstashCurrentSigningKey,
            @Value("${QSTASH_NEXT_SIGNING_KEY:}") String qstashNextSigningKey,
            @Value("${ALLOWED_ORIGINS:}") String allowedOrigins,
            @Value("${security.read-rate-limit-per-minute:120}") int readRateLimitPerMinute,
            @Value("${security.webhook-rate-limit-per-minute:30}") int webhookRateLimitPerMinute,
            @Value("${security.max-sse-connections-per-ip:5}") int maxSseConnectionsPerIp) {
        this.requireApiKey = requireApiKey;
        this.requireWebhookAuth = requireWebhookAuth;
        this.apiReadKey = apiReadKey == null ? "" : apiReadKey.trim();
        this.webhookSecret = webhookSecret == null ? "" : webhookSecret.trim();
        this.qstashCurrentSigningKey = qstashCurrentSigningKey == null ? "" : qstashCurrentSigningKey.trim();
        this.qstashNextSigningKey = qstashNextSigningKey == null ? "" : qstashNextSigningKey.trim();
        this.allowedOrigins = parseOrigins(allowedOrigins);
        this.readRateLimitPerMinute = readRateLimitPerMinute;
        this.webhookRateLimitPerMinute = webhookRateLimitPerMinute;
        this.maxSseConnectionsPerIp = maxSseConnectionsPerIp;
    }

    public boolean isRequireApiKey() {
        return requireApiKey;
    }

    public boolean isRequireWebhookAuth() {
        return requireWebhookAuth;
    }

    public String getApiReadKey() {
        return apiReadKey;
    }

    public String getWebhookSecret() {
        return webhookSecret;
    }

    public String getQstashCurrentSigningKey() {
        return qstashCurrentSigningKey;
    }

    public String getQstashNextSigningKey() {
        return qstashNextSigningKey;
    }

    public List<String> getAllowedOrigins() {
        return allowedOrigins;
    }

    public int getReadRateLimitPerMinute() {
        return readRateLimitPerMinute;
    }

    public int getWebhookRateLimitPerMinute() {
        return webhookRateLimitPerMinute;
    }

    public int getMaxSseConnectionsPerIp() {
        return maxSseConnectionsPerIp;
    }

    public boolean hasQstashSigningKeys() {
        return !qstashCurrentSigningKey.isBlank() || !qstashNextSigningKey.isBlank();
    }

    public boolean hasWebhookSecret() {
        return !webhookSecret.isBlank();
    }

    private static List<String> parseOrigins(String raw) {
        if (raw == null || raw.isBlank()) {
            return List.of();
        }
        return Arrays.stream(raw.split(","))
                .map(String::trim)
                .filter(value -> !value.isEmpty())
                .toList();
    }
}
