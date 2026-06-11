package com.mangacdc.service;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.kafka.annotation.KafkaListener;
import org.springframework.stereotype.Component;

@Component
public class KafkaChapterConsumer {

    private static final Logger log = LoggerFactory.getLogger(KafkaChapterConsumer.class);

    private final ChapterEventService chapterEventService;
    private final boolean cdcEnabled;

    public KafkaChapterConsumer(ChapterEventService chapterEventService,
                                 @Value("${cdc.enabled:false}") boolean cdcEnabled) {
        this.chapterEventService = chapterEventService;
        this.cdcEnabled = cdcEnabled;
    }

    @KafkaListener(topicPattern = "${cdc.topic-pattern:mangacdc.public.chapters}", groupId = "mangacdc-notification")
    public void onChapterEvent(String message) {
        if (!cdcEnabled) {
            return;
        }

        chapterEventService.processChapterEvent(message);
    }
}
