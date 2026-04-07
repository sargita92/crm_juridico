CREATE TABLE conversation_states (
    id CHAR(36) PRIMARY KEY,
    conversation_id CHAR(36) NOT NULL,
    specialist_id CHAR(36) NOT NULL,
    current_step_index INT NOT NULL DEFAULT 0,
    collected_data JSON,
    accumulated_score INT NOT NULL DEFAULT 0,
    handoff_active TINYINT(1) NOT NULL DEFAULT 0,
    handoff_at DATETIME(3) NULL,
    resumed_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    UNIQUE INDEX idx_conv_states_conversation (conversation_id),
    CONSTRAINT fk_conv_states_conversation FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
    CONSTRAINT fk_conv_states_specialist FOREIGN KEY (specialist_id) REFERENCES specialists(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
