package com.mangacdc.security;

import org.springframework.stereotype.Component;

import java.time.Instant;
import java.util.ArrayDeque;
import java.util.Deque;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

@Component
public class InMemoryRateLimiter {

    private final Map<String, Deque<Long>> windows = new ConcurrentHashMap<>();

    public boolean allow(String key, int maxRequests, int windowSeconds) {
        long now = Instant.now().getEpochSecond();
        Deque<Long> bucket = windows.computeIfAbsent(key, ignored -> new ArrayDeque<>());
        synchronized (bucket) {
            while (!bucket.isEmpty() && bucket.peekFirst() <= now - windowSeconds) {
                bucket.removeFirst();
            }
            if (bucket.size() >= maxRequests) {
                return false;
            }
            bucket.addLast(now);
            return true;
        }
    }
}
