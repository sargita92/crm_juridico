CREATE TABLE IF NOT EXISTS whatsapp_sessions (
    id CHAR(36) PRIMARY KEY,
    tenant_id CHAR(36) NOT NULL,
    jid VARCHAR(100) DEFAULT NULL,
    session_data LONGBLOB DEFAULT NULL,
    connected BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE INDEX idx_whatsapp_sessions_tenant (tenant_id),
    CONSTRAINT fk_whatsapp_sessions_tenant FOREIGN KEY (tenant_id)
        REFERENCES tenants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
