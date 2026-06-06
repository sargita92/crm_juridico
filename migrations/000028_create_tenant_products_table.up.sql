CREATE TABLE tenant_products (
    id CHAR(36) PRIMARY KEY,
    tenant_id CHAR(36) NOT NULL,
    product_id CHAR(36) NOT NULL,
    created_at DATETIME NOT NULL,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    UNIQUE INDEX idx_tenant_products_unique (tenant_id, product_id),
    INDEX idx_tenant_products_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Migrate existing data: create tenant_products entries from products.tenant_id
INSERT INTO tenant_products (id, tenant_id, product_id, created_at)
SELECT UUID(), tenant_id, id, NOW() FROM products WHERE tenant_id IS NOT NULL AND tenant_id != '';

-- Remove tenant_id from products
ALTER TABLE products DROP FOREIGN KEY products_ibfk_1;
ALTER TABLE products DROP INDEX idx_products_tenant_id;
ALTER TABLE products DROP INDEX idx_products_tenant_active;
ALTER TABLE products DROP COLUMN tenant_id;
ALTER TABLE products ADD INDEX idx_products_active (active);
