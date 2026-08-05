-- Reverte a biblioteca compartilhada para o vínculo 1:1. Operação lossy: um
-- guardrail anexado a vários especialistas volta associado a apenas um (o vínculo
-- mais antigo); os demais vínculos são perdidos. Guardrails sem nenhum vínculo
-- não têm especialista de retorno e são removidos para satisfazer o NOT NULL.

ALTER TABLE guardrails ADD COLUMN specialist_id CHAR(36) NULL AFTER id;

-- Re-popula a coluna a partir do vínculo mais antigo de cada guardrail.
UPDATE guardrails g
JOIN (
    SELECT sg.guardrail_id, sg.specialist_id
    FROM specialist_guardrails sg
    JOIN (
        SELECT guardrail_id, MIN(created_at) AS min_created
        FROM specialist_guardrails
        GROUP BY guardrail_id
    ) first ON first.guardrail_id = sg.guardrail_id AND first.min_created = sg.created_at
) pick ON pick.guardrail_id = g.id
SET g.specialist_id = pick.specialist_id;

DELETE FROM guardrails WHERE specialist_id IS NULL;

ALTER TABLE guardrails MODIFY COLUMN specialist_id CHAR(36) NOT NULL;
ALTER TABLE guardrails ADD INDEX idx_guardrails_specialist (specialist_id);
ALTER TABLE guardrails ADD CONSTRAINT fk_guardrails_specialist
    FOREIGN KEY (specialist_id) REFERENCES specialists(id) ON DELETE CASCADE;

DROP TABLE IF EXISTS specialist_guardrails;
