package com.mangacdc.model;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;

import java.util.ArrayList;
import java.util.List;
import java.util.Locale;

public record SeriesNotificationPrefs(
    List<String> preferredGroups,
    List<String> blockedGroups,
    int notifyEvery,
    boolean blockEarlyWeek
) {
    public static SeriesNotificationPrefs empty() {
        return new SeriesNotificationPrefs(List.of(), List.of(), 0, false);
    }

    public boolean bingeEnabled() {
        return notifyEvery > 1;
    }

    public boolean hasGroupFilters() {
        return !preferredGroups.isEmpty() || !blockedGroups.isEmpty();
    }

    public boolean allowsGroup(String scanGroup) {
        String normalized = normalize(scanGroup);
        for (String blocked : blockedGroups) {
            if (matches(normalized, blocked)) {
                return false;
            }
        }
        if (preferredGroups.isEmpty()) {
            return true;
        }
        if (normalized.isEmpty()) {
            return false;
        }
        for (String preferred : preferredGroups) {
            if (matches(normalized, preferred)) {
                return true;
            }
        }
        return false;
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
            return new SeriesNotificationPrefs(
                readStringList(node.path("preferred_groups")),
                readStringList(node.path("blocked_groups")),
                notifyEvery,
                node.path("block_early_week").asBoolean(false)
            );
        } catch (Exception ex) {
            return empty();
        }
    }

    private static List<String> readStringList(JsonNode node) {
        if (node == null || !node.isArray()) {
            return List.of();
        }
        List<String> values = new ArrayList<>();
        for (JsonNode item : node) {
            if (item.isTextual()) {
                String value = item.asText().trim();
                if (!value.isEmpty()) {
                    values.add(value);
                }
            }
        }
        return List.copyOf(values);
    }

    private static String normalize(String scanGroup) {
        return scanGroup == null ? "" : scanGroup.trim();
    }

    private static boolean matches(String scanGroup, String pattern) {
        return scanGroup.toLowerCase(Locale.ROOT).equals(pattern.trim().toLowerCase(Locale.ROOT));
    }
}
