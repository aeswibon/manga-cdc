package com.mangacdc.service;

import com.mangacdc.security.SecurityUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.web.client.RestTemplate;

import java.util.Map;

@Service
public class TelegramNotifier implements Notifier {

    private static final Logger log = LoggerFactory.getLogger(TelegramNotifier.class);

    private final RestTemplate restTemplate;
    private final String botToken;
    private final String chatId;

    public TelegramNotifier(RestTemplate restTemplate,
                            @Value("${telegram.bot-token:}") String botToken,
                            @Value("${telegram.chat-id:}") String chatId) {
        this.restTemplate = restTemplate;
        this.botToken = botToken;
        this.chatId = chatId;
    }

    @Override
    public String name() {
        return "telegram";
    }

    @Override
    public boolean isConfigured() {
        return botToken != null && !botToken.isBlank()
            && chatId != null && !chatId.isBlank();
    }

    @Override
    public boolean sendChapterAlert(String seriesTitle, String chapterNum, String chapterTitle, String url) {
        if (!isConfigured()) {
            return false;
        }

        try {
            String safeSeriesTitle = SecurityUtils.escapeTelegramHtml(seriesTitle);
            String safeChapterNum = SecurityUtils.escapeTelegramHtml(chapterNum);
            String safeChapterTitle = chapterTitle == null ? "" : SecurityUtils.escapeTelegramHtml(chapterTitle);
            String text = String.format("New Chapter!%n<b>%s</b> — Chapter %s", safeSeriesTitle, safeChapterNum);
            if (!safeChapterTitle.isBlank()) {
                text += ": " + safeChapterTitle;
            }
            if (SecurityUtils.isHttpUrl(url)) {
                text += "\n" + url.trim();
            }

            var payload = Map.of(
                "chat_id", chatId,
                "text", text,
                "parse_mode", "HTML",
                "disable_web_page_preview", false
            );

            restTemplate.postForEntity(
                "https://api.telegram.org/bot" + botToken + "/sendMessage",
                payload, String.class);
            return true;
        } catch (Exception e) {
            log.warn("Telegram notification failed: {}", e.getMessage());
            return false;
        }
    }
}
