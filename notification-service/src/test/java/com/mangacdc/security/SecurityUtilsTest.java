package com.mangacdc.security;

import org.junit.jupiter.api.Test;

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
}
