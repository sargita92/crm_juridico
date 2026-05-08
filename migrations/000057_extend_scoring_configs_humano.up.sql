ALTER TABLE scoring_configs
    ADD COLUMN threshold_humano_min INT NOT NULL DEFAULT 0,
    ADD COLUMN human_column_id      VARCHAR(36) NULL,
    ADD COLUMN cross_sell_column_id VARCHAR(36) NULL;
