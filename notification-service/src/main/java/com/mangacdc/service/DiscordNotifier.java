package com.mangacdc.service;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.web.client.RestTemplate;

import java.util.List;
import java.util.Map;

@Service
public class DiscordNotifier implements Notifier {

    private static final Logger log = LoggerFactory.getLogger(DiscordNotifier.class);

    private final RestTemplate restTemplate;
    private final String webhookUrl;

    public DiscordNotifier(RestTemplate restTemplate,
                           @Value("${discord.webhook-url:}") String webhookUrl) {
        this.restTemplate = restTemplate;
        this.webhookUrl = webhookUrl;
    }

    @Override
    public String name() {
        return "discord";
    }

    @Override
    public boolean isConfigured() {
        return webhookUrl != null && !webhookUrl.isBlank();
    }

    @Override
    public boolean sendChapterAlert(String seriesTitle, String chapterNum, String chapterTitle, String url) {
        return sendChapterAlert(seriesTitle, chapterNum, chapterTitle, url, null);
    }

    public boolean sendChapterAlert(String seriesTitle, String chapterNum, String chapterTitle, String url, String footerExtra) {
        if (!isConfigured()) {
            return false;
        }

        try {
            String description = String.format("**%s** — Chapter %s", seriesTitle, chapterNum);
            if (chapterTitle != null && !chapterTitle.isBlank()) {
                description += ": " + chapterTitle;
            }

            String footerText = "manga-cdc • Change Data Capture Pipeline";
            if (footerExtra != null && !footerExtra.isBlank()) {
                footerText += " • " + footerExtra;
            }

            var embed = Map.of(
                "title", "New Chapter Available!",
                "description", description,
                "url", url,
                "color", 0x5865F2,
                "footer", Map.of("text", footerText)
            );

            var payload = Map.of(
                "content", "@everyone",
                "embeds", List.of(embed)
            );

            restTemplate.postForEntity(webhookUrl, payload, String.class);
            return true;
        } catch (Exception e) {
            log.warn("Discord notification failed: {}", e.getMessage());
            return false;
        }
    }

    @Override
    public boolean sendSeriesAlert(String seriesTitle, String alertTitle, String message, String url) {
        if (!isConfigured()) {
            return false;
        }

        try {
            var embed = Map.of(
                "title", alertTitle,
                "description", message,
                "url", url,
                "color", 0xFEE75C,
                "footer", Map.of("text", "manga-cdc • series alert")
            );
            var payload = Map.of("embeds", List.of(embed));
            restTemplate.postForEntity(webhookUrl, payload, String.class);
            return true;
        } catch (Exception e) {
            log.warn("Discord series alert failed: {}", e.getMessage());
            return false;
        }
    }
}
