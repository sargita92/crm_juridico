CREATE TABLE execution_logs (
    id CHAR(36) NOT NULL PRIMARY KEY,
    automation_id CHAR(36) NOT NULL,
    lead_id CHAR(36) NOT NULL,
    tenant_id CHAR(36) NOT NULL,
    status VARCHAR(20) NOT NULL,
    error_message TEXT NULL,
    executed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (automation_id) REFERENCES automations(id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    INDEX idx_exec_logs_automation (automation_id),
    INDEX idx_exec_logs_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
