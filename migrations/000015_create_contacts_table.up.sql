CREATE TABLE IF NOT EXISTS contacts (
    id CHAR(36) PRIMARY KEY,
    tenant_id CHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL DEFAULT '',
    phone VARCHAR(20) NOT NULL,
    whatsapp_id VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_contacts_tenant_id (tenant_id),
    UNIQUE INDEX idx_contacts_tenant_whatsapp (tenant_id, whatsapp_id),
    CONSTRAINT fk_contacts_tenant FOREIGN KEY (tenant_id)
        REFERENCES tenants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
