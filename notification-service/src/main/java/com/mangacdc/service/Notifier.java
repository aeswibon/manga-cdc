package com.mangacdc.service;

public interface Notifier {
    boolean isConfigured();
    boolean sendChapterAlert(String seriesTitle, String chapterNum, String chapterTitle, String url);
    String name();
}
