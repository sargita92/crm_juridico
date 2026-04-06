CREATE TABLE product_phone_numbers (
    id CHAR(36) PRIMARY KEY,
    product_id CHAR(36) NOT NULL,
    phone_number VARCHAR(20) NOT NULL,
    created_at DATETIME NOT NULL,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    UNIQUE INDEX idx_product_phone_unique (phone_number)
);
