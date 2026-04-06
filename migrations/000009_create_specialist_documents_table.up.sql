CREATE TABLE IF NOT EXISTS specialist_documents (
    specialist_id CHAR(36) NOT NULL,
    document_id CHAR(36) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (specialist_id, document_id),
    CONSTRAINT fk_spec_docs_specialist FOREIGN KEY (specialist_id)
        REFERENCES specialists(id) ON DELETE CASCADE,
    CONSTRAINT fk_spec_docs_document FOREIGN KEY (document_id)
        REFERENCES documents(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
