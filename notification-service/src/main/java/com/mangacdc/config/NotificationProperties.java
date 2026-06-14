package com.mangacdc.config;

import org.springframework.boot.context.properties.ConfigurationProperties;

@ConfigurationProperties(prefix = "notifications")
public record NotificationProperties(int batchWindowSeconds) {

    public NotificationProperties {
        if (batchWindowSeconds < 0) {
            batchWindowSeconds = 30;
        }
    }

    public long batchWindowMillis() {
        return batchWindowSeconds * 1000L;
    }
}
