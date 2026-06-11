package com.mangacdc.service;

import org.junit.jupiter.api.Test;

import static org.mockito.Mockito.*;

class KafkaChapterConsumerTest {

    @Test
    void onChapterEvent_shouldSkipWhenCdcDisabled() {
        ChapterEventService eventService = mock(ChapterEventService.class);
        KafkaChapterConsumer consumer = new KafkaChapterConsumer(eventService, false);
        consumer.onChapterEvent("{}");
        verifyNoInteractions(eventService);
    }

    @Test
    void onChapterEvent_shouldDelegateWhenCdcEnabled() {
        ChapterEventService eventService = mock(ChapterEventService.class);
        KafkaChapterConsumer consumer = new KafkaChapterConsumer(eventService, true);
        consumer.onChapterEvent("{\"op\":\"c\"}");
        verify(eventService).processChapterEvent("{\"op\":\"c\"}");
    }
}
