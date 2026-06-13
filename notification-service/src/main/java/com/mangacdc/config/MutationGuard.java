package com.mangacdc.config;

import org.springframework.http.HttpStatus;
import org.springframework.stereotype.Component;
import org.springframework.web.server.ResponseStatusException;

@Component
public class MutationGuard {

    private final MutationConfig mutationConfig;

    public MutationGuard(MutationConfig mutationConfig) {
        this.mutationConfig = mutationConfig;
    }

    public void requireMutationAccess(String adminKey) {
        if (!mutationConfig.isMutationsEnabled()) {
            throw new ResponseStatusException(
                    HttpStatus.FORBIDDEN,
                    "Admin mutations are disabled. Set ADMIN_MUTATIONS_ENABLED=true to enable write endpoints.");
        }
        String configuredKey = mutationConfig.getAdminApiKey();
        if (configuredKey == null || configuredKey.isBlank()) {
            throw new ResponseStatusException(
                    HttpStatus.FORBIDDEN,
                    "Admin API key is not configured.");
        }
        if (adminKey == null || !constantTimeEquals(configuredKey, adminKey)) {
            throw new ResponseStatusException(
                    HttpStatus.FORBIDDEN,
                    "Invalid or missing X-Admin-Key header.");
        }
    }

    private static boolean constantTimeEquals(String expected, String actual) {
        return com.mangacdc.security.SecurityUtils.constantTimeEquals(expected, actual);
    }
}
