CREATE TABLE IF NOT EXISTS funnel_columns (
    id CHAR(36) PRIMARY KEY,
    funnel_id CHAR(36) NOT NULL,
    name VARCHAR(100) NOT NULL,
    order_index INT NOT NULL DEFAULT 0,
    type ENUM('entry', 'intermediate', 'won', 'lost') NOT NULL DEFAULT 'intermediate',
    color VARCHAR(7) NOT NULL DEFAULT '#3b82f6',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_columns_funnel_id (funnel_id),
    CONSTRAINT fk_columns_funnel FOREIGN KEY (funnel_id) REFERENCES funnels(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
