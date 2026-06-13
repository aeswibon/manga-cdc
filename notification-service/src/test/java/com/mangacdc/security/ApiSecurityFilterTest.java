package com.mangacdc.security;

import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.http.MediaType;
import org.springframework.test.context.TestPropertySource;
import org.springframework.test.web.servlet.MockMvc;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@SpringBootTest(properties = {
        "cdc.enabled=false",
        "spring.kafka.listener.auto-startup=false"
})
@AutoConfigureMockMvc
@TestPropertySource(properties = {
        "security.require-api-key=true",
        "API_READ_KEY=read-key",
        "security.require-webhook-auth=true",
        "WEBHOOK_SECRET=hook-secret"
})
class ApiSecurityFilterTest {

    @Autowired
    private MockMvc mockMvc;

    @Test
    void readApi_requiresApiKey() throws Exception {
        mockMvc.perform(get("/api/stats"))
                .andExpect(status().isUnauthorized());

        mockMvc.perform(get("/api/stats").header("X-Api-Key", "read-key"))
                .andExpect(status().isOk());
    }

    @Test
    void webhook_requiresSharedSecret() throws Exception {
        mockMvc.perform(post("/api/webhook")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"op\":\"c\"}"))
                .andExpect(status().isUnauthorized());

        mockMvc.perform(post("/api/webhook")
                        .header("X-Webhook-Secret", "hook-secret")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"op\":\"c\"}"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$").value("OK"));
    }

    @Test
    void healthEndpoint_staysPublic() throws Exception {
        mockMvc.perform(get("/actuator/health"))
                .andExpect(status().isOk());
    }
}
