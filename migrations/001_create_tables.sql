-- Quick Commerce Schema
-- Run: mysql -u root -p quickcommerce < migrations/001_create_tables.sql

CREATE DATABASE IF NOT EXISTS quickcommerce;
USE quickcommerce;

CREATE TABLE IF NOT EXISTS users (
    id         VARCHAR(36)  PRIMARY KEY,
    contact    VARCHAR(20)  NOT NULL,
    email      VARCHAR(255) NOT NULL,
    status     VARCHAR(20)  NOT NULL DEFAULT 'created' COMMENT 'created | active | inactive',
    created_at BIGINT       NOT NULL COMMENT 'epoch milliseconds',
    updated_at BIGINT       NOT NULL COMMENT 'epoch milliseconds'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS products (
    id          VARCHAR(36)  PRIMARY KEY,
    description VARCHAR(500) NOT NULL,
    brand       VARCHAR(100) NOT NULL DEFAULT '',
    amount      BIGINT       NOT NULL COMMENT 'price in paise (integer, no float)',
    quantity    INT          NOT NULL DEFAULT 0,
    metadata    JSON         DEFAULT NULL COMMENT '{"prev_amount": ...}',
    created_at  BIGINT       NOT NULL COMMENT 'epoch milliseconds',
    updated_at  BIGINT       NOT NULL COMMENT 'epoch milliseconds'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS orders (
    id             VARCHAR(14)  PRIMARY KEY,
    user_id        VARCHAR(36)  NOT NULL,
    product_id     VARCHAR(36)  NOT NULL,
    amount         BIGINT       NOT NULL COMMENT 'computed = product.amount * quantity, in paise',
    quantity       INT          NOT NULL,
    status         VARCHAR(20)  NOT NULL COMMENT 'success | failed',
    metadata       JSON         DEFAULT NULL COMMENT '{"quantity","latitude","longitude","address"}',
    failure_reason VARCHAR(500) DEFAULT NULL,
    request_id     VARCHAR(255) NOT NULL COMMENT 'Idempotency-Key header value, used for idempotency',
    created_at     BIGINT       NOT NULL COMMENT 'epoch milliseconds',
    INDEX idx_orders_user_id (user_id),
    UNIQUE INDEX idx_orders_request_id (request_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
