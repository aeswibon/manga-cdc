package com.mangacdc.config;

import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.TestPropertySource;
import org.springframework.web.server.ResponseStatusException;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

@SpringBootTest(classes = {MutationConfig.class, MutationGuard.class})
@TestPropertySource(properties = "ADMIN_MUTATIONS_ENABLED=false")
class MutationGuardTest {

    @Autowired
    private MutationGuard mutationGuard;

    @Test
    void requireMutationAccess_rejectsWhenMutationsDisabled() {
        assertThatThrownBy(() -> mutationGuard.requireMutationAccess(null))
                .isInstanceOf(ResponseStatusException.class)
                .satisfies(ex -> {
                    ResponseStatusException rse = (ResponseStatusException) ex;
                    assertThat(rse.getStatusCode().value()).isEqualTo(403);
                    assertThat(rse.getReason()).contains("Admin mutations are disabled");
                });
    }

    @SpringBootTest(classes = {MutationConfig.class, MutationGuard.class})
    @TestPropertySource(properties = {
            "ADMIN_MUTATIONS_ENABLED=true",
            "ADMIN_API_KEY=secret-key"
    })
    static class WithMutationsEnabledAndApiKey {

        @Autowired
        private MutationGuard mutationGuard;

        @Test
        void requireMutationAccess_rejectsMissingAdminKey() {
            assertThatThrownBy(() -> mutationGuard.requireMutationAccess(null))
                    .isInstanceOf(ResponseStatusException.class)
                    .satisfies(ex -> {
                        ResponseStatusException rse = (ResponseStatusException) ex;
                        assertThat(rse.getStatusCode().value()).isEqualTo(403);
                        assertThat(rse.getReason()).contains("X-Admin-Key");
                    });
        }

        @Test
        void requireMutationAccess_rejectsWrongAdminKey() {
            assertThatThrownBy(() -> mutationGuard.requireMutationAccess("wrong-key"))
                    .isInstanceOf(ResponseStatusException.class)
                    .satisfies(ex -> {
                        ResponseStatusException rse = (ResponseStatusException) ex;
                        assertThat(rse.getStatusCode().value()).isEqualTo(403);
                    });
        }

        @Test
        void requireMutationAccess_allowsValidAdminKey() {
            mutationGuard.requireMutationAccess("secret-key");
        }
    }

    @SpringBootTest(classes = {MutationConfig.class, MutationGuard.class})
    @TestPropertySource(properties = "ADMIN_MUTATIONS_ENABLED=true")
    static class WithMutationsEnabledNoApiKey {

        @Autowired
        private MutationGuard mutationGuard;

        @Test
        void requireMutationAccess_allowsWithoutAdminKey() {
            mutationGuard.requireMutationAccess(null);
        }
    }
}
