-- Pulse Phase 2 迁移：设备管理

CREATE TABLE IF NOT EXISTS devices (
    id                CHAR(36)     NOT NULL,
    user_id           CHAR(36)     NOT NULL,
    device_id         VARCHAR(100) NOT NULL,
    name              VARCHAR(100) NOT NULL,
    device_type       VARCHAR(50)  NOT NULL DEFAULT 'wearable',
    firmware_version  VARCHAR(20)  NULL,
    battery_level     INTEGER      NULL,
    last_seen_at      DATETIME     NULL,
    is_active         BOOLEAN      NOT NULL DEFAULT TRUE,
    device_token_hash VARCHAR(255) NULL,
    created_at        DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_devices_device_id (device_id),
    KEY idx_devices_user_id (user_id),
    CONSTRAINT fk_devices_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS device_commands (
    id         CHAR(36)    NOT NULL,
    device_id  CHAR(36)    NOT NULL,
    user_id    CHAR(36)    NOT NULL,
    command    VARCHAR(50) NOT NULL,
    status     VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_device_commands_device (device_id, status),
    CONSTRAINT fk_device_commands_device FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE,
    CONSTRAINT fk_device_commands_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
