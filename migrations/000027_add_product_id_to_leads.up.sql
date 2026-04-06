ALTER TABLE leads ADD COLUMN product_id CHAR(36) NULL;
ALTER TABLE leads ADD INDEX idx_leads_product_id (product_id);
ALTER TABLE leads ADD CONSTRAINT fk_leads_product FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE SET NULL;
