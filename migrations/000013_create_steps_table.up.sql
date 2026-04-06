CREATE TABLE IF NOT EXISTS steps (
    id CHAR(36) PRIMARY KEY,
    specialist_id CHAR(36) NOT NULL,
    order_index INT NOT NULL,
    text TEXT NOT NULL,
    data_type ENUM('free_text', 'number', 'date', 'document', 'selection') NOT NULL DEFAULT 'free_text',
    required BOOLEAN NOT NULL DEFAULT TRUE,
    score INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_steps_specialist (specialist_id),
    INDEX idx_steps_order (specialist_id, order_index),
    CONSTRAINT fk_steps_specialist FOREIGN KEY (specialist_id)
        REFERENCES specialists(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
