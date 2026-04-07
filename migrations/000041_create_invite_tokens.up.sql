CREATE TABLE invite_tokens (
    id CHAR(36) NOT NULL PRIMARY KEY,
    tenant_id CHAR(36) NOT NULL,
    token VARCHAR(64) NOT NULL,
    created_by CHAR(36) NOT NULL,
    group_ids JSON NOT NULL,
    expires_at DATETIME NOT NULL,
    used_at DATETIME NULL,
    used_by CHAR(36) NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id),
    UNIQUE KEY uk_invite_token (token),
    INDEX idx_invite_tokens_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
