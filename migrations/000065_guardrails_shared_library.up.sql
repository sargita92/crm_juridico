-- Guardrails passam a ser itens de biblioteca reutilizáveis por vários
-- especialistas. A associação 1:1 (guardrails.specialist_id) vira N:N através da
-- tabela de junção specialist_guardrails.

CREATE TABLE IF NOT EXISTS specialist_guardrails (
    specialist_id CHAR(36) NOT NULL,
    guardrail_id  CHAR(36) NOT NULL,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (specialist_id, guardrail_id),
    INDEX idx_sg_guardrail (guardrail_id),
    CONSTRAINT fk_sg_specialist FOREIGN KEY (specialist_id)
        REFERENCES specialists(id) ON DELETE CASCADE,
    CONSTRAINT fk_sg_guardrail FOREIGN KEY (guardrail_id)
        REFERENCES guardrails(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Preserva os vínculos atuais: cada guardrail existente fica anexado ao seu
-- especialista de origem.
INSERT INTO specialist_guardrails (specialist_id, guardrail_id, created_at)
SELECT specialist_id, id, created_at FROM guardrails;

-- Remove o vínculo 1:1, agora migrado para a tabela de junção.
ALTER TABLE guardrails DROP FOREIGN KEY fk_guardrails_specialist;
ALTER TABLE guardrails DROP INDEX idx_guardrails_specialist;
ALTER TABLE guardrails DROP COLUMN specialist_id;
