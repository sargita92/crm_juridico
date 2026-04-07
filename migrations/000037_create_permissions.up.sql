CREATE TABLE permissions (
    id CHAR(36) PRIMARY KEY,
    tenant_id CHAR(36) NOT NULL,
    group_id CHAR(36) NULL,
    user_id CHAR(36) NULL,
    resource VARCHAR(50) NOT NULL,
    action VARCHAR(50) NOT NULL,
    created_at DATETIME(3) NOT NULL,
    CONSTRAINT fk_permissions_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT fk_permissions_group FOREIGN KEY (group_id) REFERENCES permission_groups(id) ON DELETE CASCADE,
    CONSTRAINT fk_permissions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT chk_permissions_xor CHECK (
        (group_id IS NOT NULL AND user_id IS NULL) OR
        (group_id IS NULL AND user_id IS NOT NULL)
    ),
    UNIQUE INDEX idx_permissions_group_unique (tenant_id, group_id, resource, action),
    UNIQUE INDEX idx_permissions_user_unique (tenant_id, user_id, resource, action),
    INDEX idx_permissions_group (group_id),
    INDEX idx_permissions_user (user_id),
    INDEX idx_permissions_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
