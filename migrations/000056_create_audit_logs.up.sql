CREATE TABLE IF NOT EXISTS audit_logs (
    id           CHAR(36)     NOT NULL,
    tenant_id    CHAR(36)     NULL,
    user_id      CHAR(36)     NULL,
    actor_email  VARCHAR(255) NOT NULL,
    action       VARCHAR(64)  NOT NULL,
    entity       VARCHAR(64)  NOT NULL DEFAULT '',
    entity_id    CHAR(36)     NULL,
    ip           VARCHAR(45)  NOT NULL,
    user_agent   VARCHAR(255) NOT NULL DEFAULT '',
    metadata     JSON         NULL,
    created_at   DATETIME(3)  NOT NULL,

    PRIMARY KEY (id),
    INDEX idx_audit_created_at        (created_at),
    INDEX idx_audit_tenant_created    (tenant_id, created_at),
    INDEX idx_audit_user_created      (user_id, created_at),
    INDEX idx_audit_action_created    (action, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
