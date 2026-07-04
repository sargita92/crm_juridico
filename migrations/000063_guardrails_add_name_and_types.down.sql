-- Downgrade: remove os tipos novos e a coluna `name`.
-- ATENÇÃO: guardrails cadastrados com os tipos novos serão rejeitados pelo ENUM
-- restaurado; execute apenas se souber que só existem tipos originais em uso.

ALTER TABLE guardrails
    MODIFY COLUMN type ENUM(
        'forbidden_topics',
        'scope_limit',
        'response_tone'
    ) NOT NULL;

ALTER TABLE guardrails
    DROP COLUMN name;
