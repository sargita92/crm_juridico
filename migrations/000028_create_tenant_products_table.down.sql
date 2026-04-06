ALTER TABLE products DROP INDEX idx_products_active;
ALTER TABLE products ADD COLUMN tenant_id CHAR(36) NULL AFTER id;
ALTER TABLE products ADD INDEX idx_products_tenant_id (tenant_id);
ALTER TABLE products ADD INDEX idx_products_tenant_active (tenant_id, active);
DROP TABLE IF EXISTS tenant_products;
