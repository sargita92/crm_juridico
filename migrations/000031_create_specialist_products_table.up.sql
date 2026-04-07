CREATE TABLE specialist_products (
    id CHAR(36) PRIMARY KEY,
    specialist_id CHAR(36) NOT NULL,
    product_id CHAR(36) NOT NULL,
    created_at DATETIME(3) NOT NULL,
    UNIQUE INDEX idx_specialist_products_unique (specialist_id, product_id),
    CONSTRAINT fk_sp_specialist FOREIGN KEY (specialist_id) REFERENCES specialists(id) ON DELETE CASCADE,
    CONSTRAINT fk_sp_product FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
