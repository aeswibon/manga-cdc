package com.mangacdc.config;

import com.zaxxer.hikari.HikariDataSource;
import org.springframework.beans.factory.annotation.Value;
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
            @Lazy DataSource dataSource,
            @Value("${spring.datasource.username:}") String primaryUsername,
            @Value("${spring.datasource.password:}") String primaryPassword) {
        if (!StringUtils.hasText(readProperties.getUrl())) {
            return new JdbcTemplate(dataSource);
        }

        HikariDataSource readPool = new HikariDataSource();
        readPool.setJdbcUrl(readProperties.getUrl());
        String username = StringUtils.hasText(readProperties.getUsername())
                ? readProperties.getUsername()
                : primaryUsername;
        String password = StringUtils.hasText(readProperties.getPassword())
                ? readProperties.getPassword()
                : primaryPassword;
        readPool.setUsername(username);
        readPool.setPassword(password);
        readPool.setMaximumPoolSize(3);
        readPool.setMinimumIdle(0);
        return new JdbcTemplate(readPool);
    }
}
