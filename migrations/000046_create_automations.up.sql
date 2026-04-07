CREATE TABLE automations (
    id CHAR(36) NOT NULL PRIMARY KEY,
    tenant_id CHAR(36) NOT NULL,
    funnel_id CHAR(36) NOT NULL,
    column_id CHAR(36) NULL,
    type VARCHAR(30) NOT NULL,
    config JSON NOT NULL,
    active TINYINT(1) NOT NULL DEFAULT 1,
    priority INT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    FOREIGN KEY (funnel_id) REFERENCES funnels(id) ON DELETE CASCADE,
    INDEX idx_automations_tenant (tenant_id),
    INDEX idx_automations_column (column_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
