package com.mangacdc.service;

import org.junit.jupiter.api.Test;
import org.springframework.web.client.HttpClientErrorException;
import org.springframework.web.client.RestTemplate;

import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

class DiscordNotifierTest {

    @Test
    void isConfigured_shouldReturnFalseWhenUrlIsNull() {
        DiscordNotifier n = new DiscordNotifier(mock(RestTemplate.class), null);
        assertFalse(n.isConfigured());
    }

    @Test
    void isConfigured_shouldReturnFalseWhenUrlIsBlank() {
        DiscordNotifier n = new DiscordNotifier(mock(RestTemplate.class), "");
        assertFalse(n.isConfigured());
        n = new DiscordNotifier(mock(RestTemplate.class), "   ");
        assertFalse(n.isConfigured());
    }

    @Test
    void isConfigured_shouldReturnTrueWhenUrlIsSet() {
        DiscordNotifier n = new DiscordNotifier(mock(RestTemplate.class), "https://discord.com/api/webhooks/123");
        assertTrue(n.isConfigured());
    }

    @Test
    void sendChapterAlert_shouldReturnFalseWhenNotConfigured() {
        DiscordNotifier n = new DiscordNotifier(mock(RestTemplate.class), null);
        assertFalse(n.sendChapterAlert("Series", "1", "Title", "https://example.com"));
        verifyNoInteractions(mock(RestTemplate.class));
    }

    @Test
    void sendChapterAlert_shouldSendCorrectPayload() {
        RestTemplate restTemplate = mock(RestTemplate.class);
        DiscordNotifier n = new DiscordNotifier(restTemplate, "https://discord.com/api/webhooks/123");
        when(restTemplate.postForEntity(anyString(), any(), eq(String.class)))
            .thenReturn(null);

        boolean result = n.sendChapterAlert("One Piece", "1100", "The Final Chapter", "https://example.com/ch/1100");
        assertTrue(result);

        verify(restTemplate).postForEntity(eq("https://discord.com/api/webhooks/123"), any(), eq(String.class));
    }

    @Test
    void sendChapterAlert_shouldSendPayloadWithoutTitle() {
        RestTemplate restTemplate = mock(RestTemplate.class);
        DiscordNotifier n = new DiscordNotifier(restTemplate, "https://discord.com/api/webhooks/123");
        when(restTemplate.postForEntity(anyString(), any(), eq(String.class)))
            .thenReturn(null);

        boolean result = n.sendChapterAlert("Naruto", "700", "", "https://example.com/ch/700");
        assertTrue(result);
    }

    @Test
    void sendChapterAlert_shouldReturnFalseOnHttpError() {
        RestTemplate restTemplate = mock(RestTemplate.class);
        DiscordNotifier n = new DiscordNotifier(restTemplate, "https://discord.com/api/webhooks/123");
        when(restTemplate.postForEntity(anyString(), any(), eq(String.class)))
            .thenThrow(new HttpClientErrorException(org.springframework.http.HttpStatus.NOT_FOUND));

        boolean result = n.sendChapterAlert("Series", "1", "Title", "https://example.com");
        assertFalse(result);
    }

    @Test
    void sendChapterAlert_shouldReturnFalseOnGenericError() {
        RestTemplate restTemplate = mock(RestTemplate.class);
        DiscordNotifier n = new DiscordNotifier(restTemplate, "https://discord.com/api/webhooks/123");
        when(restTemplate.postForEntity(anyString(), any(), eq(String.class)))
            .thenThrow(new RuntimeException("Connection refused"));

        boolean result = n.sendChapterAlert("Series", "1", "Title", "https://example.com");
        assertFalse(result);
    }
}
