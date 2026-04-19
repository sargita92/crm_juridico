ALTER TABLE messages
    MODIFY COLUMN type ENUM('text','image','document','audio','video','sticker','other') NOT NULL DEFAULT 'text';
