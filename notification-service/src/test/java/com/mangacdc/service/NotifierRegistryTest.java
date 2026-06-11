package com.mangacdc.service;

import org.junit.jupiter.api.Test;

import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

class NotifierRegistryTest {

    @Test
    void sendAll_shouldOnlyCallConfiguredNotifiers() {
        Notifier n1 = mock(Notifier.class);
        Notifier n2 = mock(Notifier.class);
        when(n1.isConfigured()).thenReturn(true);
        when(n2.isConfigured()).thenReturn(false);
        when(n1.name()).thenReturn("n1");
        when(n1.sendChapterAlert(anyString(), anyString(), anyString(), anyString())).thenReturn(true);

        NotifierRegistry registry = new NotifierRegistry(List.of(n1, n2));
        Map<String, Boolean> results = registry.sendAll("S", "1", "T", "https://ex.com");

        assertEquals(1, results.size());
        assertTrue(results.get("n1"));
        verify(n1).sendChapterAlert("S", "1", "T", "https://ex.com");
        verify(n2, never()).sendChapterAlert(anyString(), anyString(), anyString(), anyString());
    }

    @Test
    void sendAll_shouldReturnResultsForMultipleChannels() {
        Notifier n1 = mock(Notifier.class);
        Notifier n2 = mock(Notifier.class);
        when(n1.isConfigured()).thenReturn(true);
        when(n2.isConfigured()).thenReturn(true);
        when(n1.name()).thenReturn("discord");
        when(n2.name()).thenReturn("slack");
        when(n1.sendChapterAlert(anyString(), anyString(), anyString(), anyString())).thenReturn(true);
        when(n2.sendChapterAlert(anyString(), anyString(), anyString(), anyString())).thenReturn(false);

        NotifierRegistry registry = new NotifierRegistry(List.of(n1, n2));
        Map<String, Boolean> results = registry.sendAll("S", "1", "T", "https://ex.com");

        assertEquals(2, results.size());
        assertTrue(results.get("discord"));
        assertFalse(results.get("slack"));
    }

    @Test
    void sendAll_shouldHandleExceptionFromNotifier() {
        Notifier n1 = mock(Notifier.class);
        when(n1.isConfigured()).thenReturn(true);
        when(n1.name()).thenReturn("failing");
        when(n1.sendChapterAlert(anyString(), anyString(), anyString(), anyString()))
            .thenThrow(new RuntimeException("API error"));

        NotifierRegistry registry = new NotifierRegistry(List.of(n1));
        Map<String, Boolean> results = registry.sendAll("S", "1", "T", "https://ex.com");

        assertEquals(1, results.size());
        assertFalse(results.get("failing"));
    }

    @Test
    void sendAll_shouldReturnEmptyWhenNoneConfigured() {
        Notifier n1 = mock(Notifier.class);
        when(n1.isConfigured()).thenReturn(false);

        NotifierRegistry registry = new NotifierRegistry(List.of(n1));
        Map<String, Boolean> results = registry.sendAll("S", "1", "T", "https://ex.com");

        assertTrue(results.isEmpty());
    }
}
