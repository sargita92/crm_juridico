-- F23 cross-sell creates a second lead for the same (tenant_id, contact_id) in the
-- target funnel (see internal/ai/infrastructure/cross_sell_adapters.go). The original
-- UNIQUE (tenant_id, contact_id) forbade that, so the cross-sell flow would fail at
-- runtime with a duplicate-entry error. Relax it to a non-unique index: the
-- "one current lead per contact" invariant is still enforced at the application layer
-- (CreateLeadUseCase guards via FindByContactAndTenant), while multiple leads per
-- contact over time (cross-sell, re-engagement) are now allowed and resolved by
-- "most recent" (created_at DESC) lookups.
ALTER TABLE leads DROP INDEX idx_leads_tenant_contact;
CREATE INDEX idx_leads_tenant_contact ON leads (tenant_id, contact_id);
