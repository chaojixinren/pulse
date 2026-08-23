-- Pulse 初始化迁移：创建核心表

CREATE TABLE IF NOT EXISTS users (
    id            CHAR(36)     NOT NULL,
    email         VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name          VARCHAR(100) NOT NULL,
    avatar_url    TEXT         NULL,
    settings      JSON         NOT NULL,
    created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at    DATETIME     NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         CHAR(36)     NOT NULL,
    user_id    CHAR(36)     NOT NULL,
    token_hash VARCHAR(255) NOT NULL,
    expires_at DATETIME     NOT NULL,
    created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at DATETIME     NULL,
    PRIMARY KEY (id),
    KEY idx_refresh_tokens_user_id (user_id),
    KEY idx_refresh_tokens_token_hash (token_hash),
    CONSTRAINT fk_refresh_tokens_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS identities (
    id              CHAR(36)    NOT NULL,
    user_id         CHAR(36)    NOT NULL,
    name            VARCHAR(50) NOT NULL,
    description     TEXT        NULL,
    color           VARCHAR(7)  NOT NULL DEFAULT '#000000',
    icon            VARCHAR(50) NOT NULL DEFAULT 'person',
    is_default      BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at      DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at      DATETIME    NULL,
    default_user_id CHAR(36) GENERATED ALWAYS AS (
        CASE WHEN is_default = 1 AND deleted_at IS NULL THEN user_id ELSE NULL END
    ) STORED,
    PRIMARY KEY (id),
    KEY idx_identities_user_id (user_id),
    UNIQUE KEY uq_identities_user_default (default_user_id),
    CONSTRAINT fk_identities_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS audio_sessions (
    id             CHAR(36)     NOT NULL,
    user_id        CHAR(36)     NOT NULL,
    identity_id    CHAR(36)     NULL,
    device_id      VARCHAR(100) NULL,
    audio_data     LONGBLOB     NOT NULL,
    audio_mime     VARCHAR(100) NULL,
    transcript     TEXT         NULL,
    duration       INTEGER      NOT NULL DEFAULT 0,
    file_size      BIGINT       NULL,
    status         VARCHAR(20)  NOT NULL DEFAULT 'pending',
    error_message  TEXT         NULL,
    extracted_data JSON         NOT NULL,
    ai_confidence  DECIMAL(3,2) NULL,
    recorded_at    DATETIME     NOT NULL,
    processed_at   DATETIME     NULL,
    created_at     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_audio_sessions_user_recorded (user_id, recorded_at),
    KEY idx_audio_sessions_status (status),
    CONSTRAINT fk_audio_sessions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_audio_sessions_identity FOREIGN KEY (identity_id) REFERENCES identities(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
