CREATE DATABASE IF NOT EXISTS cdp_db;

USE cdp_db;

CREATE TABLE IF NOT EXISTS mapping_tab
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
    `tag_value` Nullable(Int64),
    `mapping_id` UInt32,
    `create_time` DateTime DEFAULT now()
)
ENGINE = ReplacingMergeTree
PARTITION BY tag_id % 10
ORDER BY (tag_id)
SETTINGS index_granularity = 8192;

CREATE TABLE IF NOT EXISTS cdp_str_tab
(
    `tag_id` UInt64,
    `tag_value` Nullable(String),
    `mapping_id` UInt32,
    `create_time` DateTime DEFAULT now()
)
ENGINE = ReplacingMergeTree
PARTITION BY tag_id % 10
ORDER BY (tag_id)
SETTINGS index_granularity = 8192;

CREATE TABLE IF NOT EXISTS cdp_float_tab
(
    `tag_id` UInt64,
    `tag_value` Nullable(Float64),
    `mapping_id` UInt32,
    `create_time` DateTime DEFAULT now()
)
ENGINE = ReplacingMergeTree
PARTITION BY tag_id % 10
ORDER BY (tag_id)
SETTINGS index_granularity = 8192;


-- SELECT mapping_id
-- FROM cdp_int_tab
--          FINAL
-- WHERE tag_id = 10 AND tag_value = 7

