CREATE TABLE IF NOT EXISTS tenants (
    id CHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type ENUM('PF', 'PJ') NOT NULL,
    document VARCHAR(20) NOT NULL,
    status ENUM('active', 'blocked', 'inactive') NOT NULL DEFAULT 'active',
    block_reason VARCHAR(500),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY idx_tenants_document (document)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
