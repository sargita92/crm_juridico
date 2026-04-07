ALTER TABLE scoring_configs ADD COLUMN qualified_column_id CHAR(36) NULL;
ALTER TABLE scoring_configs ADD COLUMN disqualified_column_id CHAR(36) NULL;
