-- Prompt do especialista cresceu para 50k chars. TEXT (65.535 bytes) apertava
-- em UTF-8 (chars PT-BR ocupam 1-2 bytes). MEDIUMTEXT (16 MB) tira o teto do
-- banco; o limite passa a ser só o de domínio (MaxPromptLength).

ALTER TABLE specialists
    MODIFY COLUMN prompt MEDIUMTEXT NOT NULL;
