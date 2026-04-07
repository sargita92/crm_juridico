CREATE TABLE user_groups (
    id CHAR(36) PRIMARY KEY,
    user_id CHAR(36) NOT NULL,
    group_id CHAR(36) NOT NULL,
    tenant_id CHAR(36) NOT NULL,
    created_at DATETIME(3) NOT NULL,
    CONSTRAINT fk_user_groups_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_user_groups_group FOREIGN KEY (group_id) REFERENCES permission_groups(id) ON DELETE CASCADE,
    CONSTRAINT fk_user_groups_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    UNIQUE INDEX idx_user_groups_unique (user_id, group_id),
    INDEX idx_user_groups_group (group_id),
    INDEX idx_user_groups_tenant_user (tenant_id, user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
