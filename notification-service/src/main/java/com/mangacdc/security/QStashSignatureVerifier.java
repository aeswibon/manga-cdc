package com.mangacdc.security;

import com.mangacdc.config.SecurityProperties;
import org.springframework.stereotype.Component;

import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import java.nio.charset.StandardCharsets;
import java.time.Instant;
import java.util.ArrayList;
import java.util.Base64;
import java.util.List;

@Component
public class QStashSignatureVerifier {

    private static final long MAX_CLOCK_SKEW_SECONDS = 300;

    private final SecurityProperties securityProperties;

    public QStashSignatureVerifier(SecurityProperties securityProperties) {
        this.securityProperties = securityProperties;
    }

    public boolean verify(String signatureHeader, String body) {
        if (signatureHeader == null || signatureHeader.isBlank() || body == null) {
            return false;
        }

        Long timestamp = null;
        List<String> signatures = new ArrayList<>();
        for (String part : signatureHeader.split(",")) {
            String trimmed = part.trim();
            int eq = trimmed.indexOf('=');
            if (eq <= 0) {
                continue;
            }
            String key = trimmed.substring(0, eq).trim();
            String value = trimmed.substring(eq + 1).trim();
            if ("t".equals(key)) {
                try {
                    timestamp = Long.parseLong(value);
                } catch (NumberFormatException ignored) {
                    return false;
                }
            } else if ("v1".equals(key)) {
                signatures.add(value);
            }
        }

        if (timestamp == null || signatures.isEmpty()) {
            return false;
        }

        long age = Math.abs(Instant.now().getEpochSecond() - timestamp);
        if (age > MAX_CLOCK_SKEW_SECONDS) {
            return false;
        }

        String payload = timestamp + "." + body;
        for (String signingKey : signingKeys()) {
            if (signingKey.isBlank()) {
                continue;
            }
            String expected = sign(payload, signingKey);
            for (String provided : signatures) {
                if (SecurityUtils.constantTimeEquals(expected, provided)) {
                    return true;
                }
            }
        }
        return false;
    }

    private List<String> signingKeys() {
        List<String> keys = new ArrayList<>(2);
        keys.add(securityProperties.getQstashCurrentSigningKey());
        keys.add(securityProperties.getQstashNextSigningKey());
        return keys;
    }

    private static String sign(String payload, String signingKey) {
        try {
            Mac mac = Mac.getInstance("HmacSHA256");
            mac.init(new SecretKeySpec(signingKey.getBytes(StandardCharsets.UTF_8), "HmacSHA256"));
            byte[] digest = mac.doFinal(payload.getBytes(StandardCharsets.UTF_8));
            return Base64.getEncoder().encodeToString(digest);
        } catch (Exception ex) {
            return "";
        }
    }
}
