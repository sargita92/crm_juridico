CREATE TABLE ai_configs (
    id CHAR(36) PRIMARY KEY,
    specialist_id CHAR(36) NULL,
    provider VARCHAR(50) NOT NULL DEFAULT 'openai',
    model VARCHAR(100) NOT NULL DEFAULT 'gpt-5.4-nano',
    temperature DECIMAL(3,2) NOT NULL DEFAULT 0.70,
    max_tokens INT NOT NULL DEFAULT 1024,
    debounce_seconds INT NOT NULL DEFAULT 8,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    UNIQUE INDEX idx_ai_configs_specialist (specialist_id),
    CONSTRAINT fk_ai_configs_specialist FOREIGN KEY (specialist_id) REFERENCES specialists(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
