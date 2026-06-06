CREATE TABLE cross_sell_rules (
    id                 VARCHAR(36) PRIMARY KEY,
    specialist_id      VARCHAR(36) NOT NULL,
    ordem              INT NOT NULL DEFAULT 0,
    trigger_type       VARCHAR(20) NOT NULL,
    trigger_config     JSON NOT NULL,
    target_product_id  VARCHAR(36) NOT NULL,
    ativo              TINYINT(1) NOT NULL DEFAULT 1,
    created_at         DATETIME(3) NOT NULL,
    updated_at         DATETIME(3) NOT NULL,
    KEY idx_csr_spec (specialist_id, ordem),
    KEY idx_csr_target (target_product_id),
    CONSTRAINT fk_csr_spec FOREIGN KEY (specialist_id) REFERENCES specialists(id) ON DELETE CASCADE,
    CONSTRAINT fk_csr_product FOREIGN KEY (target_product_id) REFERENCES products(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
