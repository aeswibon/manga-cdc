package com.mangacdc.support;

import com.mangacdc.config.SecurityProperties;
import com.mangacdc.config.WebConfig;
import com.mangacdc.security.InMemoryRateLimiter;
import com.mangacdc.security.QStashSignatureVerifier;
import org.springframework.context.annotation.Import;

@Import({SecurityProperties.class, WebConfig.class, QStashSignatureVerifier.class, InMemoryRateLimiter.class})
public class WebMvcSecurityTestSupport {
}
