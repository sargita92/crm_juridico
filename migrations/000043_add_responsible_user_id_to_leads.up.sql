ALTER TABLE leads ADD COLUMN responsible_user_id CHAR(36) NULL;
ALTER TABLE leads ADD INDEX idx_leads_responsible (responsible_user_id);
