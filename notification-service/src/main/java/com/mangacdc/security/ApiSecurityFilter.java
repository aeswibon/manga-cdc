package com.mangacdc.security;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.mangacdc.config.SecurityProperties;
import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.springframework.core.Ordered;
import org.springframework.core.annotation.Order;
import org.springframework.http.HttpMethod;
import org.springframework.http.MediaType;
import org.springframework.stereotype.Component;
import org.springframework.web.filter.OncePerRequestFilter;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicInteger;

@Component
@Order(Ordered.HIGHEST_PRECEDENCE)
public class ApiSecurityFilter extends OncePerRequestFilter {

    private static final String API_KEY_HEADER = "X-Api-Key";
    private static final String WEBHOOK_SECRET_HEADER = "X-Webhook-Secret";

    private final SecurityProperties securityProperties;
    private final QStashSignatureVerifier qstashSignatureVerifier;
    private final InMemoryRateLimiter rateLimiter;
    private final ObjectMapper objectMapper;
    private final Map<String, AtomicInteger> sseConnections = new ConcurrentHashMap<>();

    public ApiSecurityFilter(
            SecurityProperties securityProperties,
            QStashSignatureVerifier qstashSignatureVerifier,
            InMemoryRateLimiter rateLimiter,
            ObjectMapper objectMapper) {
        this.securityProperties = securityProperties;
        this.qstashSignatureVerifier = qstashSignatureVerifier;
        this.rateLimiter = rateLimiter;
        this.objectMapper = objectMapper;
    }

    @Override
    protected void doFilterInternal(
            HttpServletRequest request,
            HttpServletResponse response,
            FilterChain filterChain) throws ServletException, IOException {
        String path = request.getRequestURI();
        String clientIp = SecurityUtils.clientIp(null, request.getRemoteAddr());

        if (path.startsWith("/actuator/health")) {
            filterChain.doFilter(request, response);
            return;
        }

        if (path.startsWith("/actuator/")) {
            if (!authorizeApiKey(request)) {
                deny(response, HttpServletResponse.SC_UNAUTHORIZED, "Unauthorized");
                return;
            }
            filterChain.doFilter(request, response);
            return;
        }

        if ("/api/webhook".equals(path) && HttpMethod.POST.matches(request.getMethod())) {
            if (!rateLimiter.allow("webhook:" + clientIp, securityProperties.getWebhookRateLimitPerMinute(), 60)) {
                deny(response, 429, "Too many webhook requests");
                return;
            }

            byte[] body = request.getInputStream().readAllBytes();
            if (!authorizeWebhook(request, new String(body, StandardCharsets.UTF_8))) {
                deny(response, HttpServletResponse.SC_UNAUTHORIZED, "Unauthorized webhook");
                return;
            }
            filterChain.doFilter(new CachedBodyHttpServletRequest(request, body), response);
            return;
        }

        if (path.startsWith("/api/")) {
            if (!rateLimiter.allow("api:" + clientIp, securityProperties.getReadRateLimitPerMinute(), 60)) {
                deny(response, 429, "Too many requests");
                return;
            }

            if ("/api/logs/stream".equals(path) && !HttpMethod.GET.matches(request.getMethod())) {
                deny(response, HttpServletResponse.SC_METHOD_NOT_ALLOWED, "Method not allowed");
                return;
            }

            if ("/api/logs/stream".equals(path)) {
                AtomicInteger active = sseConnections.computeIfAbsent(clientIp, ignored -> new AtomicInteger());
                if (active.incrementAndGet() > securityProperties.getMaxSseConnectionsPerIp()) {
                    active.decrementAndGet();
                    deny(response, 429, "Too many live streams");
                    return;
                }
                try {
                    if (!authorizeApiKey(request)) {
                        deny(response, HttpServletResponse.SC_UNAUTHORIZED, "Unauthorized");
                        return;
                    }
                    filterChain.doFilter(request, response);
                } finally {
                    active.decrementAndGet();
                }
                return;
            }

            if (!authorizeApiKey(request)) {
                deny(response, HttpServletResponse.SC_UNAUTHORIZED, "Unauthorized");
                return;
            }
        }

        filterChain.doFilter(request, response);
    }

    private boolean authorizeApiKey(HttpServletRequest request) {
        if (!securityProperties.isRequireApiKey()) {
            return true;
        }
        String provided = request.getHeader(API_KEY_HEADER);
        return SecurityUtils.constantTimeEquals(securityProperties.getApiReadKey(), provided);
    }

    private boolean authorizeWebhook(HttpServletRequest request, String body) {
        if (!securityProperties.isRequireWebhookAuth()) {
            return true;
        }

        String signature = request.getHeader("Upstash-Signature");
        if (signature != null && qstashSignatureVerifier.verify(signature, body)) {
            return true;
        }

        if (securityProperties.hasWebhookSecret()) {
            String provided = request.getHeader(WEBHOOK_SECRET_HEADER);
            if (SecurityUtils.constantTimeEquals(securityProperties.getWebhookSecret(), provided)) {
                return true;
            }

            String authorization = request.getHeader("Authorization");
            if (authorization != null && authorization.startsWith("Bearer ")) {
                String bearer = authorization.substring("Bearer ".length()).trim();
                if (SecurityUtils.constantTimeEquals(securityProperties.getWebhookSecret(), bearer)) {
                    return true;
                }
            }
        }

        return false;
    }

    private void deny(HttpServletResponse response, int status, String message) throws IOException {
        response.setStatus(status);
        response.setContentType(MediaType.APPLICATION_JSON_VALUE);
        objectMapper.writeValue(response.getOutputStream(), Map.of("error", message));
    }
}
