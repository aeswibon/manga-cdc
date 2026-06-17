package com.mangacdc.service;

public interface Notifier {
    boolean isConfigured();
    boolean sendChapterAlert(String seriesTitle, String chapterNum, String chapterTitle, String url);
    default boolean sendSeriesAlert(String seriesTitle, String alertTitle, String message, String url) {
        return false;
    }
    String name();
}
