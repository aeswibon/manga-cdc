package com.mangacdc.controller;

import com.mangacdc.service.ChapterEventService;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api")
public class WebhookController {

    private final ChapterEventService chapterEventService;

    public WebhookController(ChapterEventService chapterEventService) {
        this.chapterEventService = chapterEventService;
    }

    @PostMapping("/webhook")
    public ResponseEntity<String> handleWebhook(@RequestBody String message) {
        chapterEventService.processChapterEvent(message);
        return ResponseEntity.ok("OK");
    }
}
