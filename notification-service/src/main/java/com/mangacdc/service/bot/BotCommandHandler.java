package com.mangacdc.service.bot;

import com.mangacdc.model.Chapter;
import com.mangacdc.model.MangaSeries;
import com.mangacdc.repository.ChapterRepository;
import com.mangacdc.repository.SeriesRepository;
import org.springframework.stereotype.Service;

import java.time.ZoneId;
import java.time.format.DateTimeFormatter;
import java.util.List;

@Service
public class BotCommandHandler {

    private final ChapterRepository chapterRepository;
    private final SeriesRepository seriesRepository;

    private static final DateTimeFormatter FORMATTER = DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm")
            .withZone(ZoneId.systemDefault());

    public BotCommandHandler(ChapterRepository chapterRepository, SeriesRepository seriesRepository) {
        this.chapterRepository = chapterRepository;
        this.seriesRepository = seriesRepository;
    }

    public String handleLatestCommand() {
        List<Chapter> recent = chapterRepository.findRecentChapters(5);
        if (recent.isEmpty()) {
            return "No recent chapters found.";
        }
        
        StringBuilder sb = new StringBuilder("**Latest Chapters:**\n\n");
        for (Chapter c : recent) {
            String date = c.releaseDate() != null ? FORMATTER.format(c.releaseDate()) : "Unknown";
            sb.append(String.format("• **%s** - Ch. %s\n  %s\n  *Released: %s*\n\n", 
                c.title(), c.chapterNum(), c.url(), date));
        }
        return sb.toString();
    }

    public String handleWatchlistCommand() {
        List<MangaSeries> active = seriesRepository.findAllActive();
        if (active.isEmpty()) {
            return "Watchlist is empty.";
        }

        StringBuilder sb = new StringBuilder("**Current Watchlist:**\n\n");
        for (MangaSeries s : active) {
            sb.append(String.format("• %s (Status: %s, Latest Ch: %s)\n", 
                s.title(), s.status(), s.latestChapter() != null ? s.latestChapter() : "N/A"));
        }
        return sb.toString();
    }

    public String handleStatsCommand(String prefix) {
        int totalSeries = seriesRepository.countAll();
        int activeSeries = seriesRepository.countActive();
        
        return String.format("**Manga-CDC Stats:**\n\n" +
                "• **Total Series:** %d\n" +
                "• **Active Tracking:** %d\n" +
                "• **Recent Releases:** Run `%slatest`\n", 
                totalSeries, activeSeries, prefix);
    }

    public String handleHelpCommand(String prefix) {
        return String.format("**Available Commands:**\n\n" +
                "• `%slatest` - Show the 5 most recently released chapters\n" +
                "• `%swatchlist` - Show all manga series currently being tracked\n" +
                "• `%sstats` - Show database statistics\n" +
                "• `%shelp` - Show this help message\n", 
                prefix, prefix, prefix, prefix);
    }
}
