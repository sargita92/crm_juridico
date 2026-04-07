CREATE TABLE group_funnels (
    id CHAR(36) PRIMARY KEY,
    group_id CHAR(36) NOT NULL,
    funnel_id CHAR(36) NOT NULL,
    column_ids JSON NOT NULL,
    created_at DATETIME(3) NOT NULL,
    CONSTRAINT fk_group_funnels_group FOREIGN KEY (group_id) REFERENCES permission_groups(id) ON DELETE CASCADE,
    CONSTRAINT fk_group_funnels_funnel FOREIGN KEY (funnel_id) REFERENCES funnels(id) ON DELETE CASCADE,
    UNIQUE INDEX idx_group_funnels_unique (group_id, funnel_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
