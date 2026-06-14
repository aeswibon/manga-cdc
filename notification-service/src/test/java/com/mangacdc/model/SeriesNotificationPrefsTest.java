package com.mangacdc.model;

import com.fasterxml.jackson.databind.ObjectMapper;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

class SeriesNotificationPrefsTest {

    private final ObjectMapper mapper = new ObjectMapper();

    @Test
    void fromJson_parsesNotifyEvery() {
        SeriesNotificationPrefs prefs = SeriesNotificationPrefs.fromJson(
            "{\"notify_every\": 5, \"preferred_groups\": [\"group-a\"]}", mapper);
        assertEquals(5, prefs.notifyEvery());
        assertTrue(prefs.bingeEnabled());
    }

    @Test
    void fromJson_defaultsWhenMissing() {
        SeriesNotificationPrefs prefs = SeriesNotificationPrefs.fromJson("{}", mapper);
        assertEquals(0, prefs.notifyEvery());
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
}
