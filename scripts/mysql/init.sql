CREATE DATABASE IF NOT EXISTS metadata_db;

USE metadata_db;

CREATE TABLE IF NOT EXISTS tag_tab (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(64) NOT NULL,
    `desc` VARCHAR(255) NOT NULL,
    `enum` TEXT NOT NULL,
    `value_type` TINYINT UNSIGNED NOT NULL,
    `status` TINYINT UNSIGNED NOT NULL DEFAULT '1',
    `ext_info` TEXT NOT NULL,
    `create_time` BIGINT UNSIGNED NOT NULL,
    `update_time` BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_name` (`name`),
    KEY `idx_name_desc_status` (`name`, `desc`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS segment_tab (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(64) NOT NULL,
    `desc` VARCHAR(255) NOT NULL,
    `criteria` TEXT NOT NULL,
    `status` TINYINT UNSIGNED NOT NULL DEFAULT '1',
    `create_time` BIGINT UNSIGNED NOT NULL,
    `update_time` BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_name` (`name`),
    KEY `idx_name_desc_status` (`name`, `desc`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS email_tab (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(64) NOT NULL,
    `email_desc` VARCHAR(255) NOT NULL,
    `json` TEXT NOT NULL,
    `html` TEXT NOT NULL,
    `status` TINYINT UNSIGNED NOT NULL DEFAULT '1',
    `create_time` BIGINT UNSIGNED NOT NULL,
    `update_time` BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (`id`),
    KEY `idx_name_email_desc_status` (`name`, `email_desc`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS task_tab (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `tag_id` BIGINT UNSIGNED NOT NULL,
    `tag_value` VARCHAR(255) NOT NULL DEFAULT '',
    `file_name` VARCHAR(64) NOT NULL,
    `file_key` VARCHAR(64) NOT NULL,
    `url` VARCHAR(255) NOT NULL,
    `status` TINYINT UNSIGNED NOT NULL,
    `create_time` BIGINT UNSIGNED NOT NULL,
    `update_time` BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_key` (`file_key`),
    KEY `idx_tag_id_status_action` (`tag_id`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS campaign_tab (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(64) NOT NULL,
    `campaign_desc` VARCHAR(255) NOT NULL,
    `segment_id` BIGINT UNSIGNED NOT NULL,
    `segment_size` BIGINT UNSIGNED NOT NULL,
    `progress` TINYINT UNSIGNED NOT NULL,
    `schedule` BIGINT UNSIGNED NOT NULL,
    `status` TINYINT UNSIGNED NOT NULL,
    `create_time` BIGINT UNSIGNED NOT NULL,
    `update_time` BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS campaign_email_tab (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `campaign_id` BIGINT UNSIGNED NOT NULL,
    `email_id` BIGINT UNSIGNED NOT NULL,
    `subject` VARCHAR(255) NOT NULL,
    `html` TEXT NOT NULL,
    `ratio` TINYINT UNSIGNED NOT NULL,
    `open_count` INT UNSIGNED NOT NULL,
    `click_counts` TEXT NOT NULL,
    PRIMARY KEY (`id`),
    KEY `idx_campaign_id` (`campaign_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE DATABASE IF NOT EXISTS mapping_id_db;

USE mapping_id_db;

CREATE TABLE IF NOT EXISTS mapping_id_tab (
    id bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    ud_id varchar(64) NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE INDEX idx_ud_id (`ud_id`)
);
