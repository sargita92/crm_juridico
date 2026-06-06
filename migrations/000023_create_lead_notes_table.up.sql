CREATE TABLE lead_notes (
    id CHAR(36) PRIMARY KEY,
    lead_id CHAR(36) NOT NULL,
    tenant_id CHAR(36) NOT NULL,
    content TEXT NOT NULL,
    created_by CHAR(36) NOT NULL,
    created_at DATETIME NOT NULL,
    FOREIGN KEY (lead_id) REFERENCES leads(id) ON DELETE CASCADE,
    INDEX idx_lead_notes_lead_id (lead_id),
    INDEX idx_lead_notes_tenant_id (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
