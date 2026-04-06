CREATE TABLE IF NOT EXISTS lead_movements (
    id CHAR(36) PRIMARY KEY,
    lead_id CHAR(36) NOT NULL,
    from_column_id CHAR(36) DEFAULT NULL,
    to_column_id CHAR(36) NOT NULL,
    moved_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_movements_lead_id (lead_id),
    CONSTRAINT fk_movements_lead FOREIGN KEY (lead_id) REFERENCES leads(id) ON DELETE CASCADE,
    CONSTRAINT fk_movements_from FOREIGN KEY (from_column_id) REFERENCES funnel_columns(id),
    CONSTRAINT fk_movements_to FOREIGN KEY (to_column_id) REFERENCES funnel_columns(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
