-- Fixture: dados iniciais para ambiente de desenvolvimento
-- Senha do admin: admin123

-- Tenant padrão
INSERT INTO tenants (id, name, type, document, status, created_at, updated_at)
VALUES (
    'a9b1aef9-b1c9-48e4-9013-612710c954a5',
    'Escritório Teste',
    'PJ',
    '00.000.000/0001-00',
    'active',
    NOW(),
    NOW()
) ON DUPLICATE KEY UPDATE name = VALUES(name);

-- Segundo tenant
INSERT INTO tenants (id, name, type, document, status, created_at, updated_at)
VALUES (
    'f8c7e890-0444-446f-b66b-3ecb8e65395c',
    'Advocacia Silva & Associados',
    'PJ',
    '11.111.111/0001-11',
    'active',
    NOW(),
    NOW()
) ON DUPLICATE KEY UPDATE name = VALUES(name);

-- Usuário admin
-- Senha: admin123
INSERT INTO users (id, name, email, password_hash, role, status, created_at, updated_at)
VALUES (
    'ecfa4f42-d902-4552-923e-f55dd270fbb6',
    'Admin',
    'admin@teste.com',
    '$2a$10$Y9F207O7K9qrZnD34MBd/ONKUYL7KXR0JypYb07MpyAOqSa/5v.KW',
    'admin',
    'active',
    NOW(),
    NOW()
) ON DUPLICATE KEY UPDATE name = VALUES(name);

-- Admin NÃO precisa de associação com tenants (vê todos automaticamente)
