ALTER TABLE specialists
    ADD COLUMN cross_sell_enabled                TINYINT(1) NOT NULL DEFAULT 0,
    ADD COLUMN cross_sell_mode                   VARCHAR(16) NOT NULL DEFAULT 'announce',
    ADD COLUMN cross_sell_announcement_template TEXT NULL,
    ADD COLUMN allow_ai_cross_sell_suggestion   TINYINT(1) NOT NULL DEFAULT 0;
