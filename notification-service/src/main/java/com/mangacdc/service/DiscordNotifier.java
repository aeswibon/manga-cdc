package com.mangacdc.service;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.web.client.RestTemplate;

import java.util.List;
import java.util.Map;

@Service
public class DiscordNotifier {

    private final RestTemplate restTemplate;
    private final String webhookUrl;

    public DiscordNotifier(RestTemplate restTemplate,
                           @Value("${discord.webhook-url:}") String webhookUrl) {
        this.restTemplate = restTemplate;
        this.webhookUrl = webhookUrl;
    }

    public boolean isConfigured() {
        return webhookUrl != null && !webhookUrl.isBlank();
    }

    public boolean sendChapterAlert(String seriesTitle, String chapterNum, String chapterTitle, String url) {
        if (!isConfigured()) {
            return false;
        }

        try {
            String description = String.format("**%s** — Chapter %s", seriesTitle, chapterNum);
            if (chapterTitle != null && !chapterTitle.isBlank()) {
                description += ": " + chapterTitle;
            }

            var embed = Map.of(
                "title", "New Chapter Available!",
                "description", description,
                "url", url,
                "color", 0x5865F2,
                "footer", Map.of("text", "manga-cdc • Change Data Capture Pipeline")
            );

            var payload = Map.of(
                "content", "@everyone",
                "embeds", List.of(embed)
            );

            restTemplate.postForEntity(webhookUrl, payload, String.class);
            return true;
        } catch (Exception e) {
            return false;
        }
    }
}
