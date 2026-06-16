package com.mangacdc.config;

import com.zaxxer.hikari.HikariDataSource;
import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.Lazy;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.util.StringUtils;

import javax.sql.DataSource;

@Configuration
@EnableConfigurationProperties(ReadDataSourceProperties.class)
public class DatabaseConfig {

    @Bean
    public JdbcTemplate readJdbcTemplate(
            ReadDataSourceProperties readProperties,
            @Lazy DataSource dataSource) {
        if (!StringUtils.hasText(readProperties.getUrl())) {
            return new JdbcTemplate(dataSource);
        }

        HikariDataSource readPool = new HikariDataSource();
        readPool.setJdbcUrl(readProperties.getUrl());
        String username = readProperties.getUsername();
        String password = readProperties.getPassword();
        if (!StringUtils.hasText(username) && dataSource instanceof HikariDataSource primary) {
            username = primary.getUsername();
        }
        if (!StringUtils.hasText(password) && dataSource instanceof HikariDataSource primary) {
            password = primary.getPassword();
        }
        readPool.setUsername(username);
        readPool.setPassword(password);
        readPool.setMaximumPoolSize(3);
        readPool.setMinimumIdle(0);
        return new JdbcTemplate(readPool);
    }
}
