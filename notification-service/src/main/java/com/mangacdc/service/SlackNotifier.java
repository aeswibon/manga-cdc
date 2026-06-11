package com.mangacdc.service;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.web.client.RestTemplate;

import java.util.Map;

@Service
public class SlackNotifier implements Notifier {

    private static final Logger log = LoggerFactory.getLogger(SlackNotifier.class);

    private final RestTemplate restTemplate;
    private final String webhookUrl;

    public SlackNotifier(RestTemplate restTemplate,
                         @Value("${slack.webhook-url:}") String webhookUrl) {
        this.restTemplate = restTemplate;
        this.webhookUrl = webhookUrl;
    }

    @Override
    public String name() {
        return "slack";
    }

    @Override
    public boolean isConfigured() {
        return webhookUrl != null && !webhookUrl.isBlank();
    }

    @Override
    public boolean sendChapterAlert(String seriesTitle, String chapterNum, String chapterTitle, String url) {
        if (!isConfigured()) {
            return false;
        }

        try {
            String text = String.format("*%s* — Chapter %s", seriesTitle, chapterNum);
            if (chapterTitle != null && !chapterTitle.isBlank()) {
                text += ": " + chapterTitle;
            }
            text += "\n" + url;

            var payload = Map.of("text", text);
            restTemplate.postForEntity(webhookUrl, payload, String.class);
            return true;
        } catch (Exception e) {
            log.warn("Slack notification failed: {}", e.getMessage());
            return false;
        }
    }
}
