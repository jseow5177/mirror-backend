DROP DATABASE IF EXISTS mirrorcdp_db;

CREATE DATABASE IF NOT EXISTS mirrorcdp_db;

USE mirrorcdp_db;

CREATE TABLE IF NOT EXISTS tag_tab (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `tenant_id` BIGINT UNSIGNED NOT NULL,
    `name` VARCHAR(64) NOT NULL,
    `tag_desc` VARCHAR(256) NOT NULL,
    `enum` TEXT NOT NULL,
    `value_type` TINYINT UNSIGNED NOT NULL,
    `status` TINYINT UNSIGNED NOT NULL DEFAULT '1',
    `ext_info` TEXT NOT NULL,
    `creator_id` BIGINT UNSIGNED NOT NULL,
    `create_time` BIGINT UNSIGNED NOT NULL,
    `update_time` BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_tenant_id_name_status` (`tenant_id`, `name`, `status`),
    KEY `idx_tag_desc` (`tag_desc`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS segment_tab (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `tenant_id` BIGINT UNSIGNED NOT NULL,
    `name` VARCHAR(64) NOT NULL,
    `segment_desc` VARCHAR(256) NOT NULL,
    `criteria` TEXT NOT NULL,
    `status` TINYINT UNSIGNED NOT NULL DEFAULT '1',
    `creator_id` BIGINT UNSIGNED NOT NULL,
    `create_time` BIGINT UNSIGNED NOT NULL,
    `update_time` BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_tenant_id_name_status` (`tenant_id`, `name`, `status`),
    KEY `idx_segment_desc` (`segment_desc`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS email_tab (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `tenant_id` BIGINT UNSIGNED NOT NULL,
    `name` VARCHAR(64) NOT NULL,
    `email_desc` VARCHAR(256) NOT NULL,
    `json` TEXT NOT NULL,
    `html` TEXT NOT NULL,
    `status` TINYINT UNSIGNED NOT NULL DEFAULT '1',
    `creator_id` BIGINT UNSIGNED NOT NULL,
    `create_time` BIGINT UNSIGNED NOT NULL,
    `update_time` BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (`id`),
    KEY `idx_tenant_id_name_email_desc_status` (`tenant_id`, `name`, `email_desc`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS campaign_tab (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `tenant_id` BIGINT UNSIGNED NOT NULL,
    `name` VARCHAR(64) NOT NULL,
    `campaign_desc` VARCHAR(256) NOT NULL,
    `segment_id` BIGINT UNSIGNED NOT NULL,
    `sender_id` BIGINT UNSIGNED NOT NULL,
    `segment_size` BIGINT UNSIGNED NOT NULL,
    `progress` TINYINT UNSIGNED NOT NULL,
    `schedule` BIGINT UNSIGNED NOT NULL,
    `status` TINYINT UNSIGNED NOT NULL,
    `creator_id` BIGINT UNSIGNED NOT NULL,
    `create_time` BIGINT UNSIGNED NOT NULL,
    `update_time` BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (`id`),
    KEY `idx_tenant_id_name_campaign_desc_status` (`tenant_id`, `name`, `campaign_desc`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS campaign_email_tab (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `campaign_id` BIGINT UNSIGNED NOT NULL,
    `email_id` BIGINT UNSIGNED NOT NULL,
    `subject` VARCHAR(256) NOT NULL,
    `ratio` TINYINT UNSIGNED NOT NULL,
    PRIMARY KEY (`id`),
    KEY `idx_campaign_id` (`campaign_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS campaign_log_tab (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `campaign_email_id` BIGINT UNSIGNED NOT NULL,
    `event` TINYINT UNSIGNED NOT NULL,
    `link` VARCHAR(2048) NOT NULL DEFAULT '',
    `email` VARCHAR(320) NOT NULL,
    `event_time` BIGINT UNSIGNED NOT NULL,
    `create_time` BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (`id`),
    KEY `idx_campaign_id` (`campaign_email_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS tenant_tab (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(64) NOT NULL,
    `status` TINYINT UNSIGNED NOT NULL,
    `ext_info` TEXT NOT NULL,
    `create_time` BIGINT UNSIGNED NOT NULL,
    `update_time` BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_name_status` (`name`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS user_tab (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `tenant_id` BIGINT UNSIGNED NOT NULL,
    `email` VARCHAR(320) NOT NULL,
    `username` VARCHAR(256) NOT NULL,
    `password` TEXT NOT NULL,
    `display_name` VARCHAR(256) NOT NULL DEFAULT '',
    `status` TINYINT UNSIGNED NOT NULL,
    `create_time` BIGINT UNSIGNED NOT NULL,
    `update_time` BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_tenant_id_email_status` (`tenant_id`, `email`, `status`),
    UNIQUE KEY `idx_tenant_id_username_status` (`tenant_id`, `username`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS activation_tab (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `token_hash` VARCHAR(128) NOT NULL,
    `token_type` TINYINT UNSIGNED NOT NULL,
    `target_id` BIGINT UNSIGNED NOT NULL,
    `expire_time` BIGINT UNSIGNED NOT NULL,
    `create_time` BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_token_hash_token_type` (`token_hash`, `token_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS session_tab (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `user_id` BIGINT UNSIGNED NOT NULL,
    `token_hash` VARCHAR(128) NOT NULL,
    `expire_time` BIGINT UNSIGNED NOT NULL,
    `create_time` BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_token_hash_expire_time` (`token_hash`, `expire_time`),
    KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS task_tab (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `tenant_id` BIGINT UNSIGNED NOT NULL,
    `resource_id` BIGINT UNSIGNED NOT NULL,
    `status` TINYINT UNSIGNED NOT NULL,
    `resource_type` TINYINT UNSIGNED NOT NULL,
    `task_type` TINYINT UNSIGNED NOT NULL,
    `ext_info` TEXT NOT NULL,
    `creator_id` BIGINT UNSIGNED NOT NULL,
    `create_time` BIGINT UNSIGNED NOT NULL,
    `update_time` BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (`id`),
    KEY `idx_resource_id` (`resource_id`),
    KEY `idx_resource_type_task_type_status` (`resource_type`, `task_type`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS role_tab (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `tenant_id` BIGINT UNSIGNED NOT NULL,
    `name` VARCHAR(64) NOT NULL,
    `role_desc` VARCHAR(256) NOT NULL,
    `actions` TEXT NOT NULL,
    `status` TINYINT UNSIGNED NOT NULL,
    `create_time` BIGINT UNSIGNED NOT NULL,
    `update_time` BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_tenant_id_name` (`tenant_id`, `name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS user_role_tab (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `tenant_id` BIGINT UNSIGNED NOT NULL,
    `user_id` BIGINT UNSIGNED NOT NULL,
    `role_id` BIGINT UNSIGNED NOT NULL,
    `create_time` BIGINT UNSIGNED NOT NULL,
    `update_time` BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_tenant_id_user_id_role_id` (`tenant_id`, `user_id`, `role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS sender_tab (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `tenant_id` BIGINT UNSIGNED NOT NULL,
    `name` VARCHAR(64) NOT NULL,
    `local_part` VARCHAR(64) NOT NULL,
    `create_time` BIGINT UNSIGNED NOT NULL,
    `update_time` BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_tenant_id_name_local_part` (`tenant_id`, `name`, `local_part`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
