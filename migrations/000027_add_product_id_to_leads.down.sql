ALTER TABLE leads DROP FOREIGN KEY fk_leads_product;
ALTER TABLE leads DROP INDEX idx_leads_product_id;
ALTER TABLE leads DROP COLUMN product_id;
