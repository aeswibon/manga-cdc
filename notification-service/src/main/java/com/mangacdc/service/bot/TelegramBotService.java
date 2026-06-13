package com.mangacdc.service.bot;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.telegram.telegrambots.bots.TelegramLongPollingBot;
import org.telegram.telegrambots.meta.api.methods.send.SendMessage;
import org.telegram.telegrambots.meta.api.objects.Update;
import org.telegram.telegrambots.meta.exceptions.TelegramApiException;

@Service
public class TelegramBotService extends TelegramLongPollingBot {

    private static final Logger log = LoggerFactory.getLogger(TelegramBotService.class);

    private final String botUsername;
    private final String botToken;
    private final BotCommandHandler commandHandler;

    public TelegramBotService(@Value("${telegram.bot-username:}") String botUsername,
                              @Value("${telegram.bot-token:}") String botToken,
                              BotCommandHandler commandHandler) {
        super(botToken);
        this.botUsername = botUsername;
        this.botToken = botToken;
        this.commandHandler = commandHandler;
    }

    @Override
    public String getBotUsername() {
        return botUsername;
    }

    @Override
    public String getBotToken() {
        return botToken;
    }

    @Override
    public void onUpdateReceived(Update update) {
        if (botToken == null || botToken.isBlank()) {
            return;
        }

        if (update.hasMessage() && update.getMessage().hasText()) {
            String messageText = update.getMessage().getText();
            long chatId = update.getMessage().getChatId();

            String response = null;
            if (messageText.startsWith("/latest")) {
                response = commandHandler.handleLatestCommand();
            } else if (messageText.startsWith("/watchlist")) {
                response = commandHandler.handleWatchlistCommand();
            } else if (messageText.startsWith("/stats")) {
                response = commandHandler.handleStatsCommand("/");
            } else if (messageText.startsWith("/help")) {
                response = commandHandler.handleHelpCommand("/");
            }

            if (response != null) {
                sendMessage(chatId, response);
            }
        }
    }

    private void sendMessage(long chatId, String text) {
        SendMessage message = new SendMessage();
        message.setChatId(String.valueOf(chatId));
        message.setText(text);
        // Using Markdown format to match Discord formatting generally
        message.setParseMode("Markdown");

        try {
            execute(message);
        } catch (TelegramApiException e) {
            log.error("Failed to send Telegram message: {}", e.getMessage());
        }
    }
}
