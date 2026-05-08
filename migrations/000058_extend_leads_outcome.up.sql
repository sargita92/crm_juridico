ALTER TABLE leads
    ADD COLUMN qualification_outcome    VARCHAR(20) NOT NULL DEFAULT 'em_andamento',
    ADD COLUMN cross_sell_origin_lead_id VARCHAR(36) NULL,
    ADD CONSTRAINT fk_leads_cs_origin FOREIGN KEY (cross_sell_origin_lead_id) REFERENCES leads(id) ON DELETE SET NULL;

CREATE INDEX idx_leads_outcome ON leads(qualification_outcome);
