package com.mangacdc.config;

import com.zaxxer.hikari.HikariDataSource;
import org.springframework.boot.autoconfigure.jdbc.DataSourceProperties;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.Lazy;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.util.StringUtils;

import javax.sql.DataSource;

@Configuration
public class DatabaseConfig {

    @Bean
    @ConfigurationProperties("spring.datasource.read")
    public DataSourceProperties readDataSourceProperties() {
        return new DataSourceProperties();
    }

    @Bean
    public JdbcTemplate readJdbcTemplate(
            DataSourceProperties readDataSourceProperties,
            @Lazy DataSource dataSource) {
        if (!StringUtils.hasText(readDataSourceProperties.getUrl())) {
            return new JdbcTemplate(dataSource);
        }

        HikariDataSource readPool = readDataSourceProperties.initializeDataSourceBuilder()
                .type(HikariDataSource.class)
                .build();
        if (!StringUtils.hasText(readDataSourceProperties.getUsername())
                && dataSource instanceof HikariDataSource primary) {
            readPool.setUsername(primary.getUsername());
        }
        if (!StringUtils.hasText(readDataSourceProperties.getPassword())
                && dataSource instanceof HikariDataSource primary) {
            readPool.setPassword(primary.getPassword());
        }
        return new JdbcTemplate(readPool);
    }
}
