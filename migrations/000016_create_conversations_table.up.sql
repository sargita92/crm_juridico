CREATE TABLE IF NOT EXISTS conversations (
    id CHAR(36) PRIMARY KEY,
    tenant_id CHAR(36) NOT NULL,
    contact_id CHAR(36) NOT NULL,
    status ENUM('open', 'closed') NOT NULL DEFAULT 'open',
    last_message_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    unread_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_conversations_tenant_id (tenant_id),
    INDEX idx_conversations_contact_id (contact_id),
    INDEX idx_conversations_last_message (tenant_id, last_message_at DESC),
    CONSTRAINT fk_conversations_tenant FOREIGN KEY (tenant_id)
        REFERENCES tenants(id),
    CONSTRAINT fk_conversations_contact FOREIGN KEY (contact_id)
        REFERENCES contacts(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
