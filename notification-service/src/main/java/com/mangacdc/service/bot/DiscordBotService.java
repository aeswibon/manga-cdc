package com.mangacdc.service.bot;

import discord4j.core.DiscordClientBuilder;
import discord4j.core.GatewayDiscordClient;
import discord4j.core.event.domain.message.MessageCreateEvent;
import discord4j.core.object.entity.Message;
import jakarta.annotation.PostConstruct;
import jakarta.annotation.PreDestroy;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import reactor.core.publisher.Mono;

@Service
public class DiscordBotService {

    private static final Logger log = LoggerFactory.getLogger(DiscordBotService.class);

    private final String botToken;
    private final BotCommandHandler commandHandler;
    private GatewayDiscordClient gateway;

    public DiscordBotService(@Value("${discord.bot-token:}") String botToken, BotCommandHandler commandHandler) {
        this.botToken = botToken;
        this.commandHandler = commandHandler;
    }

    @PostConstruct
    public void init() {
        if (botToken == null || botToken.isBlank()) {
            log.info("Discord bot token not configured. Interactive bot is disabled.");
            return;
        }

        try {
            gateway = DiscordClientBuilder.create(botToken).build()
                    .login()
                    .block();
            
            if (gateway != null) {
                gateway.on(MessageCreateEvent.class, event -> {
                    Message message = event.getMessage();
                    String content = message.getContent();
                    
                    if (content.startsWith("!latest")) {
                        return message.getChannel().flatMap(channel -> channel.createMessage(commandHandler.handleLatestCommand()));
                    } else if (content.startsWith("!watchlist")) {
                        return message.getChannel().flatMap(channel -> channel.createMessage(commandHandler.handleWatchlistCommand()));
                    } else if (content.startsWith("!stats")) {
                        return message.getChannel().flatMap(channel -> channel.createMessage(commandHandler.handleStatsCommand("!")));
                    } else if (content.startsWith("!help")) {
                        return message.getChannel().flatMap(channel -> channel.createMessage(commandHandler.handleHelpCommand("!")));
                    }
                    
                    return Mono.empty();
                }).subscribe();
                
                log.info("Discord Interactive Bot started.");
            }
        } catch (Exception e) {
            log.error("Failed to start Discord Interactive Bot: {}", e.getMessage());
        }
    }

    @PreDestroy
    public void destroy() {
        if (gateway != null) {
            gateway.logout().block();
        }
    }
}
