package com.mangacdc.service;

import com.mangacdc.config.NotificationProperties;
import org.junit.jupiter.api.Test;

import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

class ChapterNotificationBatcherTest {

    @Test
    void enqueue_withZeroWindow_flushesImmediately() {
        ChapterNotificationBatcher batcher = new ChapterNotificationBatcher(new NotificationProperties(0));
        List<ChapterNotificationBatcher.PendingChapter> flushed = new ArrayList<>();

        batcher.enqueue(sample("ch1", "s1", "1"), flushed::addAll);

        assertEquals(1, flushed.size());
        assertEquals("ch1", flushed.get(0).chapterId());
    }

    @Test
    void enqueue_batchesChaptersWithinWindow() throws InterruptedException {
        ChapterNotificationBatcher batcher = new ChapterNotificationBatcher(new NotificationProperties(1));
        CountDownLatch latch = new CountDownLatch(1);
        List<ChapterNotificationBatcher.PendingChapter> flushed = new ArrayList<>();

        batcher.enqueue(sample("ch1", "s1", "1"), chapters -> {
            flushed.addAll(chapters);
            latch.countDown();
        });
        batcher.enqueue(sample("ch2", "s1", "2"), chapters -> {
            flushed.addAll(chapters);
            latch.countDown();
        });

        assertTrue(latch.await(3, TimeUnit.SECONDS));
        assertEquals(2, flushed.size());
        assertEquals("ch1", flushed.get(0).chapterId());
        assertEquals("ch2", flushed.get(1).chapterId());
    }

    private static ChapterNotificationBatcher.PendingChapter sample(String chapterId, String seriesId, String num) {
        return new ChapterNotificationBatcher.PendingChapter(
            chapterId, seriesId, "Series", num, "Title", "https://example.com/" + num
        );
    }
}
