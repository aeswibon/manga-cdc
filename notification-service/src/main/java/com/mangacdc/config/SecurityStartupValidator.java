package com.mangacdc.config;

import jakarta.annotation.PostConstruct;
import org.springframework.stereotype.Component;

@Component
public class SecurityStartupValidator {

    private final SecurityProperties securityProperties;
    private final MutationConfig mutationConfig;

    public SecurityStartupValidator(SecurityProperties securityProperties, MutationConfig mutationConfig) {
        this.securityProperties = securityProperties;
        this.mutationConfig = mutationConfig;
    }

    @PostConstruct
    void validate() {
        if (mutationConfig.isMutationsEnabled() && mutationConfig.getAdminApiKey().isBlank()) {
            throw new IllegalStateException(
                    "ADMIN_API_KEY must be set when ADMIN_MUTATIONS_ENABLED=true");
        }

        if (securityProperties.isRequireApiKey() && securityProperties.getApiReadKey().isBlank()) {
            throw new IllegalStateException(
                    "API_READ_KEY must be set when SECURITY_REQUIRE_API_KEY=true");
        }

        if (securityProperties.isRequireWebhookAuth()
                && !securityProperties.hasQstashSigningKeys()
                && !securityProperties.hasWebhookSecret()) {
            throw new IllegalStateException(
                    "Configure QSTASH_CURRENT_SIGNING_KEY and/or WEBHOOK_SECRET when SECURITY_REQUIRE_WEBHOOK_AUTH=true");
        }
    }
}
