package com.mangacdc.service;

import org.junit.jupiter.api.Test;
import org.springframework.web.client.HttpClientErrorException;
import org.springframework.web.client.RestTemplate;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

class SlackNotifierTest {

    @Test
    void name_shouldReturnSlack() {
        SlackNotifier n = new SlackNotifier(mock(RestTemplate.class), null);
        assertEquals("slack", n.name());
    }

    @Test
    void isConfigured_shouldReturnFalseWhenUrlIsNull() {
        SlackNotifier n = new SlackNotifier(mock(RestTemplate.class), null);
        assertFalse(n.isConfigured());
    }

    @Test
    void isConfigured_shouldReturnFalseWhenUrlIsBlank() {
        SlackNotifier n = new SlackNotifier(mock(RestTemplate.class), "");
        assertFalse(n.isConfigured());
    }

    @Test
    void isConfigured_shouldReturnTrueWhenUrlIsSet() {
        SlackNotifier n = new SlackNotifier(mock(RestTemplate.class), "https://hooks.slack.com/services/xxx");
        assertTrue(n.isConfigured());
    }

    @Test
    void sendChapterAlert_shouldReturnFalseWhenNotConfigured() {
        SlackNotifier n = new SlackNotifier(mock(RestTemplate.class), null);
        assertFalse(n.sendChapterAlert("Series", "1", "Title", "https://example.com"));
    }

    @Test
    void sendChapterAlert_shouldSendCorrectPayload() {
        RestTemplate restTemplate = mock(RestTemplate.class);
        SlackNotifier n = new SlackNotifier(restTemplate, "https://hooks.slack.com/services/xxx");
        when(restTemplate.postForEntity(anyString(), any(), eq(String.class)))
            .thenReturn(null);

        boolean result = n.sendChapterAlert("One Piece", "1100", "The Final Chapter", "https://example.com/ch/1100");
        assertTrue(result);

        verify(restTemplate).postForEntity(eq("https://hooks.slack.com/services/xxx"), any(), eq(String.class));
    }

    @Test
    void sendChapterAlert_shouldSendPayloadWithoutTitle() {
        RestTemplate restTemplate = mock(RestTemplate.class);
        SlackNotifier n = new SlackNotifier(restTemplate, "https://hooks.slack.com/services/xxx");
        when(restTemplate.postForEntity(anyString(), any(), eq(String.class)))
            .thenReturn(null);

        boolean result = n.sendChapterAlert("Naruto", "700", "", "https://example.com/ch/700");
        assertTrue(result);
    }

    @Test
    void sendChapterAlert_shouldReturnFalseOnHttpError() {
        RestTemplate restTemplate = mock(RestTemplate.class);
        SlackNotifier n = new SlackNotifier(restTemplate, "https://hooks.slack.com/services/xxx");
        when(restTemplate.postForEntity(anyString(), any(), eq(String.class)))
            .thenThrow(HttpClientErrorException.class);

        boolean result = n.sendChapterAlert("Series", "1", "Title", "https://example.com");
        assertFalse(result);
    }

    @Test
    void sendChapterAlert_shouldReturnFalseOnGenericError() {
        RestTemplate restTemplate = mock(RestTemplate.class);
        SlackNotifier n = new SlackNotifier(restTemplate, "https://hooks.slack.com/services/xxx");
        when(restTemplate.postForEntity(anyString(), any(), eq(String.class)))
            .thenThrow(new RuntimeException("Connection refused"));

        boolean result = n.sendChapterAlert("Series", "1", "Title", "https://example.com");
        assertFalse(result);
    }
}
