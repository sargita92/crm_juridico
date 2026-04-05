CREATE TABLE IF NOT EXISTS specialist_tenants (
    specialist_id CHAR(36) NOT NULL,
    tenant_id CHAR(36) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (specialist_id, tenant_id),
    CONSTRAINT fk_specialist_tenants_specialist FOREIGN KEY (specialist_id)
        REFERENCES specialists(id) ON DELETE CASCADE,
    CONSTRAINT fk_specialist_tenants_tenant FOREIGN KEY (tenant_id)
        REFERENCES tenants(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
