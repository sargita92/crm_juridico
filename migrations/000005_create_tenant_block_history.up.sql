CREATE TABLE IF NOT EXISTS tenant_block_history (
    id CHAR(36) PRIMARY KEY,
    tenant_id CHAR(36) NOT NULL,
    action ENUM('block', 'unblock') NOT NULL,
    reason VARCHAR(500) NOT NULL,
    performed_by CHAR(36) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_tbh_tenant_id (tenant_id),
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
