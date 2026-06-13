package com.mangacdc.config;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Configuration;

@Configuration
public class MutationConfig {

    private final boolean mutationsEnabled;
    private final String adminApiKey;

    public MutationConfig(
            @Value("${ADMIN_MUTATIONS_ENABLED:false}") boolean mutationsEnabled,
            @Value("${ADMIN_API_KEY:}") String adminApiKey) {
        this.mutationsEnabled = mutationsEnabled;
        this.adminApiKey = adminApiKey;
    }

    public boolean isMutationsEnabled() {
        return mutationsEnabled;
    }

    public String getAdminApiKey() {
        return adminApiKey;
    }

    public boolean isAdminKeyRequired() {
        return adminApiKey != null && !adminApiKey.isBlank();
    }
}
