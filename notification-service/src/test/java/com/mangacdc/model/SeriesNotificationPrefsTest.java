package com.mangacdc.model;

import com.fasterxml.jackson.databind.ObjectMapper;
import org.junit.jupiter.api.Test;

import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

class SeriesNotificationPrefsTest {

    private final ObjectMapper mapper = new ObjectMapper();

    @Test
    void fromJson_parsesNotifyEveryAndGroups() {
        SeriesNotificationPrefs prefs = SeriesNotificationPrefs.fromJson(
            "{\"notify_every\": 5, \"preferred_groups\": [\"group-a\"], \"blocked_groups\": [\"bad-group\"]}", mapper);
        assertEquals(5, prefs.notifyEvery());
        assertEquals(List.of("group-a"), prefs.preferredGroups());
        assertEquals(List.of("bad-group"), prefs.blockedGroups());
        assertTrue(prefs.bingeEnabled());
    }

    @Test
    void fromJson_defaultsWhenMissing() {
        SeriesNotificationPrefs prefs = SeriesNotificationPrefs.fromJson("{}", mapper);
        assertEquals(0, prefs.notifyEvery());
        assertTrue(prefs.preferredGroups().isEmpty());
        assertTrue(prefs.blockedGroups().isEmpty());
        assertFalse(prefs.bingeEnabled());
    }

    @Test
    void fromJson_clampsNegativeValues() {
        SeriesNotificationPrefs prefs = SeriesNotificationPrefs.fromJson("{\"notify_every\": -2}", mapper);
        assertEquals(0, prefs.notifyEvery());
    }

    @Test
    void fromJson_invalidJsonReturnsEmpty() {
        SeriesNotificationPrefs prefs = SeriesNotificationPrefs.fromJson("{not json", mapper);
        assertEquals(SeriesNotificationPrefs.empty(), prefs);
    }

    @Test
    void allowsGroup_blocksDeniedGroups() {
        SeriesNotificationPrefs prefs = new SeriesNotificationPrefs(List.of(), List.of("Machine TL"), 0, false);
        assertFalse(prefs.allowsGroup("Machine TL"));
        assertTrue(prefs.allowsGroup("Official TL"));
    }

    @Test
    void allowsGroup_requiresPreferredMatch() {
        SeriesNotificationPrefs prefs = new SeriesNotificationPrefs(List.of("Official TL"), List.of(), 0, false);
        assertTrue(prefs.allowsGroup("official tl"));
        assertFalse(prefs.allowsGroup("Machine TL"));
        assertFalse(prefs.allowsGroup(""));
    }

    @Test
    void fromJson_parsesBlockEarlyWeek() {
        SeriesNotificationPrefs prefs = SeriesNotificationPrefs.fromJson(
            "{\"block_early_week\": true}", mapper);
        assertTrue(prefs.blockEarlyWeek());
    }
}
