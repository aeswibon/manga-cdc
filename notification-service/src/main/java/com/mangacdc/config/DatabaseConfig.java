package com.mangacdc.config;

import com.zaxxer.hikari.HikariDataSource;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.boot.autoconfigure.jdbc.DataSourceProperties;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
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
    public DataSource readDataSource(
            DataSourceProperties readDataSourceProperties,
            DataSource dataSource) {
        if (!StringUtils.hasText(readDataSourceProperties.getUrl())) {
            return dataSource;
        }
        HikariDataSource readPool = readDataSourceProperties.initializeDataSourceBuilder()
                .type(HikariDataSource.class)
                .build();
        if (!StringUtils.hasText(readDataSourceProperties.getUsername())) {
            HikariDataSource primary = (HikariDataSource) dataSource;
            readPool.setUsername(primary.getUsername());
        }
        if (!StringUtils.hasText(readDataSourceProperties.getPassword())) {
            HikariDataSource primary = (HikariDataSource) dataSource;
            readPool.setPassword(primary.getPassword());
        }
        return readPool;
    }

    @Bean
    public JdbcTemplate readJdbcTemplate(@Qualifier("readDataSource") DataSource readDataSource) {
        return new JdbcTemplate(readDataSource);
    }
}
