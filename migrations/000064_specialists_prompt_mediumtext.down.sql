-- ATENÇÃO: prompts com mais de 65.535 bytes serão truncados por MySQL ao
-- rebaixar de MEDIUMTEXT para TEXT. Verifique antes de rodar em prod.

ALTER TABLE specialists
    MODIFY COLUMN prompt TEXT NOT NULL;
