-- Adiciona nome distintivo por guardrail (usado em auditoria/busca) e amplia
-- os tipos aceitos para cobrir LGPD, escalonamento humano e validação de saída.

ALTER TABLE guardrails
    ADD COLUMN name VARCHAR(120) NOT NULL DEFAULT '' AFTER specialist_id;

-- Backfill: guardrails legados ficam com nome derivado do tipo para satisfazer
-- a exigência de nome distintivo. Truncar em 120 chars é a largura do campo.
UPDATE guardrails
SET name = CONCAT('Guardrail — ', REPLACE(REPLACE(REPLACE(type,
    'forbidden_topics', 'Tópicos proibidos'),
    'scope_limit',      'Limite de escopo'),
    'response_tone',    'Tom de resposta'))
WHERE name = '';

-- MySQL não permite ALTER + drop de DEFAULT em uma passada; separar mantém a
-- coluna NOT NULL sem default agora que os legados foram preenchidos.
ALTER TABLE guardrails
    MODIFY COLUMN name VARCHAR(120) NOT NULL;

-- Amplia o ENUM. Ordem existente preservada para não invalidar rows atuais.
ALTER TABLE guardrails
    MODIFY COLUMN type ENUM(
        'forbidden_topics',
        'scope_limit',
        'response_tone',
        'security_lgpd',
        'human_escalation',
        'output_validation'
    ) NOT NULL;
