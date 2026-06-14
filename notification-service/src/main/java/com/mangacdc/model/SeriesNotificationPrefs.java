package com.mangacdc.model;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;

public record SeriesNotificationPrefs(
    int notifyEvery
) {
    public static SeriesNotificationPrefs empty() {
        return new SeriesNotificationPrefs(0);
    }

    public boolean bingeEnabled() {
        return notifyEvery > 1;
    }

    public static SeriesNotificationPrefs fromJson(String json, ObjectMapper mapper) {
        if (json == null || json.isBlank()) {
            return empty();
        }
        try {
            JsonNode node = mapper.readTree(json);
            int notifyEvery = node.path("notify_every").asInt(0);
            if (notifyEvery < 0) {
                notifyEvery = 0;
            }
            return new SeriesNotificationPrefs(notifyEvery);
        } catch (Exception ex) {
            return empty();
        }
    }
}
