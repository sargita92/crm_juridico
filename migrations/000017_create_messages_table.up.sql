CREATE TABLE IF NOT EXISTS messages (
    id CHAR(36) PRIMARY KEY,
    conversation_id CHAR(36) NOT NULL,
    direction ENUM('incoming', 'outgoing') NOT NULL,
    content TEXT NOT NULL,
    type ENUM('text') NOT NULL DEFAULT 'text',
    status ENUM('pending', 'sent', 'failed') NOT NULL DEFAULT 'sent',
    whatsapp_msg_id VARCHAR(100) DEFAULT NULL,
    timestamp TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_messages_conversation_id (conversation_id),
    INDEX idx_messages_timestamp (conversation_id, timestamp),
    UNIQUE INDEX idx_messages_whatsapp_msg_id (whatsapp_msg_id),
    CONSTRAINT fk_messages_conversation FOREIGN KEY (conversation_id)
        REFERENCES conversations(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
