DROP INDEX idx_leads_outcome ON leads;
ALTER TABLE leads
    DROP FOREIGN KEY fk_leads_cs_origin,
    DROP COLUMN cross_sell_origin_lead_id,
    DROP COLUMN qualification_outcome;
