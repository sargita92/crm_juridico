ALTER TABLE messages
    MODIFY COLUMN type ENUM('text') NOT NULL DEFAULT 'text';
