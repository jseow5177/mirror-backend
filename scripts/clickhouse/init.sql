CREATE DATABASE IF NOT EXISTS cdp_db;

USE cdp_db;

CREATE TABLE IF NOT EXISTS cdp_mapping_tab
(
    `mapping_id` UInt32,
    `ud_id` String,
    `create_time` DateTime DEFAULT now()
)
ENGINE = ReplacingMergeTree
PARTITION BY mapping_id % 100
ORDER BY (mapping_id)
SETTINGS index_granularity = 8192;

CREATE TABLE IF NOT EXISTS cdp_int_tab
(
    `tag_id` UInt64,
    `mapping_id` UInt32,
    `ud_id` String,
    `tag_value` Nullable(Int64),
    `create_time` DateTime DEFAULT now()
)
ENGINE = ReplacingMergeTree
PARTITION BY tag_id % 10
ORDER BY (tag_id, mapping_id)
SETTINGS index_granularity = 8192;

CREATE TABLE IF NOT EXISTS cdp_str_tab
(
    `tag_id` UInt64,
    `mapping_id` UInt32,
    `ud_id` String,
    `tag_value` Nullable(String),
    `create_time` DateTime DEFAULT now()
)
ENGINE = ReplacingMergeTree
PARTITION BY tag_id % 10
ORDER BY (tag_id, mapping_id)
SETTINGS index_granularity = 8192;


-- SELECT mapping_id FROM cdp_str WHERE tag_id = 10 GROUP BY tag_id, mapping_id HAVING argMax(tag_value, create_time) = 'World';


