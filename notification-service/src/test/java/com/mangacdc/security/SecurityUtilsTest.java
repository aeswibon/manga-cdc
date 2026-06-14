package com.mangacdc.security;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

class SecurityUtilsTest {

    @Test
    void constantTimeEquals_matchesIdenticalValues() {
        assertTrue(SecurityUtils.constantTimeEquals("secret", "secret"));
        assertFalse(SecurityUtils.constantTimeEquals("secret", "other"));
    }

    @Test
    void isHttpUrl_acceptsHttpsOnlyPatterns() {
        assertTrue(SecurityUtils.isHttpUrl("https://example.com/ch/1"));
        assertFalse(SecurityUtils.isHttpUrl("javascript:alert(1)"));
    }

    @Test
    void escapeTelegramHtml_escapesMarkup() {
        assertTrue(SecurityUtils.escapeTelegramHtml("<b>ok</b>").contains("&lt;b&gt;"));
    }

    @Test
    void clientIp_prefersForwardedForHeader() {
        assertEquals("203.0.113.5", SecurityUtils.clientIp("203.0.113.5, 198.51.100.2", "10.0.0.1"));
        assertEquals("10.0.0.1", SecurityUtils.clientIp(null, "10.0.0.1"));
    }
}
