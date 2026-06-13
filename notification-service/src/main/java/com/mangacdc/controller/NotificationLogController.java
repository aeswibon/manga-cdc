package com.mangacdc.controller;

import com.mangacdc.config.MutationGuard;
import com.mangacdc.model.NotificationLogEntry;
import com.mangacdc.repository.NotificationLogRepository;
import com.mangacdc.service.Notifier;
import com.mangacdc.service.SseEmitterService;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.servlet.mvc.method.annotation.SseEmitter;

import java.util.List;
import java.util.UUID;

@RestController
@RequestMapping("/api")
@CrossOrigin(origins = "*")
public class NotificationLogController {

    private static final int MAX_LOG_LIMIT = 100;

    private final NotificationLogRepository notificationLogRepository;
    private final SseEmitterService sseEmitterService;
    private final List<Notifier> notifiers;
    private final MutationGuard mutationGuard;

    public NotificationLogController(NotificationLogRepository notificationLogRepository,
                                     SseEmitterService sseEmitterService,
                                     List<Notifier> notifiers,
                                     MutationGuard mutationGuard) {
        this.notificationLogRepository = notificationLogRepository;
        this.sseEmitterService = sseEmitterService;
        this.notifiers = notifiers;
        this.mutationGuard = mutationGuard;
    }

    @GetMapping("/logs")
    public List<NotificationLogEntry> listLogs(@RequestParam(defaultValue = "50") int limit) {
        int cappedLimit = Math.min(limit, MAX_LOG_LIMIT);
        return notificationLogRepository.findRecent(cappedLimit);
    }

    @GetMapping(value = "/logs/stream", produces = "text/event-stream")
    public SseEmitter streamLogs() {
        SseEmitter emitter = new SseEmitter(Long.MAX_VALUE);
        sseEmitterService.addEmitter(emitter);
        return emitter;
    }

    @PostMapping("/logs/{logId}/retry")
    public NotificationLogEntry retryLog(
            @PathVariable String logId,
            @RequestHeader(value = "X-Admin-Key", required = false) String adminKey) {
        mutationGuard.requireMutationAccess(adminKey);
        UUID id = UUID.fromString(logId);
        NotificationLogEntry entry = notificationLogRepository.findById(id);

        Notifier targetNotifier = notifiers.stream()
                .filter(n -> n.name().equalsIgnoreCase(entry.channel()))
                .findFirst()
                .orElseThrow(() -> new IllegalArgumentException("Unsupported channel: " + entry.channel()));

        boolean success = targetNotifier.sendChapterAlert(
                entry.seriesTitle(),
                entry.chapterNum().toString(),
                entry.chapterTitle(),
                entry.chapterUrl()
        );

        String status = success ? "SENT" : "FAILED";
        String error = success ? null : "Webhook returned error on retry";

        notificationLogRepository.updateStatus(id, status, error);

        NotificationLogEntry updated = notificationLogRepository.findById(id);
        sseEmitterService.publishLog(updated);

        return updated;
    }
}
