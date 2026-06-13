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
        if (mutationConfig.isAdminKeyRequired()) {
            String configuredKey = mutationConfig.getAdminApiKey();
            if (adminKey == null || !configuredKey.equals(adminKey)) {
                throw new ResponseStatusException(
                        HttpStatus.FORBIDDEN,
                        "Invalid or missing X-Admin-Key header.");
            }
        }
    }
}
