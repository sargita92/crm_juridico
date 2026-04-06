CREATE TABLE IF NOT EXISTS guardrails (
    id CHAR(36) PRIMARY KEY,
    specialist_id CHAR(36) NOT NULL,
    type ENUM('forbidden_topics', 'scope_limit', 'response_tone') NOT NULL,
    rule TEXT NOT NULL,
    message TEXT,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_guardrails_specialist (specialist_id),
    CONSTRAINT fk_guardrails_specialist FOREIGN KEY (specialist_id)
        REFERENCES specialists(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
