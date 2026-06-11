package com.mangacdc.controller;

import com.mangacdc.model.NotificationLogEntry;
import com.mangacdc.repository.NotificationLogRepository;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;

@RestController
@RequestMapping("/api")
public class NotificationLogController {

    private final NotificationLogRepository notificationLogRepository;

    public NotificationLogController(NotificationLogRepository notificationLogRepository) {
        this.notificationLogRepository = notificationLogRepository;
    }

    @GetMapping("/logs")
    public List<NotificationLogEntry> listLogs(@RequestParam(defaultValue = "50") int limit) {
        return notificationLogRepository.findRecent(limit);
    }
}
