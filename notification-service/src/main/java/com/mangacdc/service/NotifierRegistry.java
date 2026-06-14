package com.mangacdc.service;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

@Component
public class NotifierRegistry {

    private static final Logger log = LoggerFactory.getLogger(NotifierRegistry.class);

    private final List<Notifier> notifiers;

    public NotifierRegistry(List<Notifier> notifiers) {
        this.notifiers = notifiers;
    }

    public Map<String, Boolean> sendAll(String seriesTitle, String chapterNum, String chapterTitle, String url) {
        Map<String, Boolean> results = new LinkedHashMap<>();
        for (Notifier notifier : notifiers) {
            if (!notifier.isConfigured()) {
                continue;
            }
            try {
                boolean ok = notifier.sendChapterAlert(seriesTitle, chapterNum, chapterTitle, url);
                results.put(notifier.name(), ok);
                if (!ok) {
                    log.warn("{} notification failed for {} chapter {}", notifier.name(), seriesTitle, chapterNum);
                }
            } catch (Exception e) {
                results.put(notifier.name(), false);
                log.error("{} notification error for {} chapter {}", notifier.name(), seriesTitle, chapterNum, e);
            }
        }
        return results;
    }

    public Map<String, Boolean> sendMassRelease(String seriesTitle, String rangeLabel, int count, String url) {
        String chapterTitle = String.format("Mass release (%d chapters)", count);
        return sendAll(seriesTitle, rangeLabel, chapterTitle, url);
    }
}
