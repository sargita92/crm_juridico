-- Restore the UNIQUE constraint. Note: rollback will fail if duplicate
-- (tenant_id, contact_id) rows already exist (e.g. cross-sell leads); those
-- must be reconciled before downgrading.
ALTER TABLE leads DROP INDEX idx_leads_tenant_contact;
CREATE UNIQUE INDEX idx_leads_tenant_contact ON leads (tenant_id, contact_id);
