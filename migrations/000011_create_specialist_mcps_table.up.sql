CREATE TABLE IF NOT EXISTS specialist_mcps (
    specialist_id CHAR(36) NOT NULL,
    mcp_id CHAR(36) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (specialist_id, mcp_id),
    CONSTRAINT fk_spec_mcps_specialist FOREIGN KEY (specialist_id)
        REFERENCES specialists(id) ON DELETE CASCADE,
    CONSTRAINT fk_spec_mcps_mcp FOREIGN KEY (mcp_id)
        REFERENCES mcp_servers(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
