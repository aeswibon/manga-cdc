package com.mangacdc.service;

import com.mangacdc.config.NotificationProperties;
import jakarta.annotation.PreDestroy;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;

import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledFuture;
import java.util.concurrent.TimeUnit;
import java.util.function.Consumer;

@Component
public class ChapterNotificationBatcher {

    public record PendingChapter(
        String chapterId,
        String seriesId,
        String seriesTitle,
        String chapterNum,
        String title,
        String url
    ) {}

    private final long batchWindowMs;
    private final ScheduledExecutorService scheduler;
    private final ConcurrentHashMap<String, Batch> batches = new ConcurrentHashMap<>();

    @Autowired
    public ChapterNotificationBatcher(NotificationProperties properties) {
        this.batchWindowMs = properties.batchWindowMillis();
        this.scheduler = Executors.newSingleThreadScheduledExecutor(r -> {
            Thread thread = new Thread(r, "chapter-notification-batcher");
            thread.setDaemon(true);
            return thread;
        });
    }

    public void enqueue(PendingChapter chapter, Consumer<List<PendingChapter>> onFlush) {
        if (batchWindowMs <= 0) {
            onFlush.accept(List.of(chapter));
            return;
        }

        batches.compute(chapter.seriesId(), (seriesId, existing) -> {
            Batch batch = existing != null ? existing : new Batch();
            batch.chapters.add(chapter);
            if (batch.flushTask != null) {
                batch.flushTask.cancel(false);
            }
            batch.flushTask = scheduler.schedule(
                () -> flush(seriesId, onFlush),
                batchWindowMs,
                TimeUnit.MILLISECONDS
            );
            return batch;
        });
    }

    private void flush(String seriesId, Consumer<List<PendingChapter>> onFlush) {
        Batch batch = batches.remove(seriesId);
        if (batch == null || batch.chapters.isEmpty()) {
            return;
        }
        onFlush.accept(List.copyOf(batch.chapters));
    }

    @PreDestroy
    void shutdown() {
        scheduler.shutdownNow();
    }

    static final class Batch {
        final List<PendingChapter> chapters = new ArrayList<>();
        ScheduledFuture<?> flushTask;
    }
}
