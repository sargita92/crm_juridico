CREATE TABLE funnel_products (
    id CHAR(36) PRIMARY KEY,
    funnel_id CHAR(36) NOT NULL,
    product_id CHAR(36) NOT NULL,
    priority INT NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL,
    FOREIGN KEY (funnel_id) REFERENCES funnels(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    UNIQUE INDEX idx_funnel_products_unique (funnel_id, product_id),
    INDEX idx_funnel_products_product_priority (product_id, priority DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
