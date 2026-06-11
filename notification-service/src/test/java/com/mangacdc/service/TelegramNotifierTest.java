package com.mangacdc.service;

import org.junit.jupiter.api.Test;
import org.springframework.web.client.HttpClientErrorException;
import org.springframework.web.client.RestTemplate;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

class TelegramNotifierTest {

    @Test
    void name_shouldReturnTelegram() {
        TelegramNotifier n = new TelegramNotifier(mock(RestTemplate.class), null, null);
        assertEquals("telegram", n.name());
    }

    @Test
    void isConfigured_shouldReturnFalseWhenTokenIsNull() {
        TelegramNotifier n = new TelegramNotifier(mock(RestTemplate.class), null, "12345");
        assertFalse(n.isConfigured());
    }

    @Test
    void isConfigured_shouldReturnFalseWhenChatIdIsNull() {
        TelegramNotifier n = new TelegramNotifier(mock(RestTemplate.class), "token:xxx", null);
        assertFalse(n.isConfigured());
    }

    @Test
    void isConfigured_shouldReturnFalseWhenTokenIsBlank() {
        TelegramNotifier n = new TelegramNotifier(mock(RestTemplate.class), "", "12345");
        assertFalse(n.isConfigured());
    }

    @Test
    void isConfigured_shouldReturnTrueWhenBothSet() {
        TelegramNotifier n = new TelegramNotifier(mock(RestTemplate.class), "token:xxx", "12345");
        assertTrue(n.isConfigured());
    }

    @Test
    void sendChapterAlert_shouldReturnFalseWhenNotConfigured() {
        TelegramNotifier n = new TelegramNotifier(mock(RestTemplate.class), null, null);
        assertFalse(n.sendChapterAlert("Series", "1", "Title", "https://example.com"));
    }

    @Test
    void sendChapterAlert_shouldSendToCorrectEndpoint() {
        RestTemplate restTemplate = mock(RestTemplate.class);
        TelegramNotifier n = new TelegramNotifier(restTemplate, "bot123:abc", "-10012345");
        when(restTemplate.postForEntity(anyString(), any(), eq(String.class)))
            .thenReturn(null);

        boolean result = n.sendChapterAlert("One Piece", "1100", "The Final Chapter", "https://example.com/ch/1100");
        assertTrue(result);

        verify(restTemplate).postForEntity(
            eq("https://api.telegram.org/botbot123:abc/sendMessage"),
            argThat(payload -> {
                var map = (java.util.Map<String, Object>) payload;
                return "-10012345".equals(map.get("chat_id"))
                    && "HTML".equals(map.get("parse_mode"));
            }),
            eq(String.class));
    }

    @Test
    void sendChapterAlert_shouldSendPayloadWithoutTitle() {
        RestTemplate restTemplate = mock(RestTemplate.class);
        TelegramNotifier n = new TelegramNotifier(restTemplate, "bot123:abc", "-10012345");
        when(restTemplate.postForEntity(anyString(), any(), eq(String.class)))
            .thenReturn(null);

        boolean result = n.sendChapterAlert("Naruto", "700", "", "https://example.com/ch/700");
        assertTrue(result);
    }

    @Test
    void sendChapterAlert_shouldReturnFalseOnHttpError() {
        RestTemplate restTemplate = mock(RestTemplate.class);
        TelegramNotifier n = new TelegramNotifier(restTemplate, "bot123:abc", "-10012345");
        when(restTemplate.postForEntity(anyString(), any(), eq(String.class)))
            .thenThrow(HttpClientErrorException.class);

        boolean result = n.sendChapterAlert("Series", "1", "Title", "https://example.com");
        assertFalse(result);
    }

    @Test
    void sendChapterAlert_shouldReturnFalseOnGenericError() {
        RestTemplate restTemplate = mock(RestTemplate.class);
        TelegramNotifier n = new TelegramNotifier(restTemplate, "bot123:abc", "-10012345");
        when(restTemplate.postForEntity(anyString(), any(), eq(String.class)))
            .thenThrow(new RuntimeException("Connection refused"));

        boolean result = n.sendChapterAlert("Series", "1", "Title", "https://example.com");
        assertFalse(result);
    }
}
