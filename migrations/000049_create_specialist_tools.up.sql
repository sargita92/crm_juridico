CREATE TABLE specialist_tools (
    id            CHAR(36) NOT NULL PRIMARY KEY,
    specialist_id CHAR(36) NOT NULL,
    tool_name     VARCHAR(100) NOT NULL,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_specialist_tools_specialist FOREIGN KEY (specialist_id) REFERENCES specialists(id) ON DELETE CASCADE,
    UNIQUE KEY idx_specialist_tool (specialist_id, tool_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
