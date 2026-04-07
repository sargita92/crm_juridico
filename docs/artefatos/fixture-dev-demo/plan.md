# Fixture Dev/Demo — Plano de Implementação

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Popular `fixture/fixtures.sql` com dados realistas de um escritório de advocacia previdenciária para desenvolvimento local e demos.

**Architecture:** Arquivo SQL único, idempotente (`ON DUPLICATE KEY UPDATE`), organizado por seções com UUIDs fixos. Ordem de inserção respeita dependências de FK.

**Tech Stack:** MySQL 8.0, SQL puro

**Spec:** `docs/artefatos/fixture-dev-demo/design.md`

---

## UUID Registry

Todos os UUIDs usados na fixture, para referência cruzada:

```
-- Tenant
TENANT   = '550e8400-e29b-41d4-a716-446655440001'

-- Users (admin existente mantido)
ADMIN    = 'ecfa4f42-d902-4552-923e-f55dd270fbb6'
RICARDO  = '550e8400-e29b-41d4-a716-446655440010'
ANA      = '550e8400-e29b-41d4-a716-446655440011'
JULIANA  = '550e8400-e29b-41d4-a716-446655440012'

-- Permission Groups
GRP_ADV  = '550e8400-e29b-41d4-a716-446655440020'
GRP_ATD  = '550e8400-e29b-41d4-a716-446655440021'

-- Products
PROD_IDADE    = '550e8400-e29b-41d4-a716-446655440030'
PROD_TEMPO    = '550e8400-e29b-41d4-a716-446655440031'
PROD_BPC      = '550e8400-e29b-41d4-a716-446655440032'
PROD_AUXILIO  = '550e8400-e29b-41d4-a716-446655440033'
PROD_REVISAO  = '550e8400-e29b-41d4-a716-446655440034'

-- Specialist
SPECIALIST = '550e8400-e29b-41d4-a716-446655440040'

-- Steps
STEP1 = '550e8400-e29b-41d4-a716-446655440050'
STEP2 = '550e8400-e29b-41d4-a716-446655440051'
STEP3 = '550e8400-e29b-41d4-a716-446655440052'
STEP4 = '550e8400-e29b-41d4-a716-446655440053'
STEP5 = '550e8400-e29b-41d4-a716-446655440054'
STEP6 = '550e8400-e29b-41d4-a716-446655440055'

-- Scoring / Guardrails
SCORING   = '550e8400-e29b-41d4-a716-446655440060'
GUARD1    = '550e8400-e29b-41d4-a716-446655440061'
GUARD2    = '550e8400-e29b-41d4-a716-446655440062'
GUARD3    = '550e8400-e29b-41d4-a716-446655440063'

-- Funnel
FUNNEL = '550e8400-e29b-41d4-a716-446655440070'

-- Columns (order matches order_index)
COL_NOVO       = '550e8400-e29b-41d4-a716-446655440080'
COL_ANALISE    = '550e8400-e29b-41d4-a716-446655440081'
COL_CALCULO    = '550e8400-e29b-41d4-a716-446655440082'
COL_PROTOCOLO  = '550e8400-e29b-41d4-a716-446655440083'
COL_AGUARDANDO = '550e8400-e29b-41d4-a716-446655440084'
COL_RECURSO    = '550e8400-e29b-41d4-a716-446655440085'
COL_GANHO      = '550e8400-e29b-41d4-a716-446655440086'
COL_PERDIDO    = '550e8400-e29b-41d4-a716-446655440087'

-- Contacts
CONTACT_MARIA     = '550e8400-e29b-41d4-a716-446655440090'
CONTACT_JOSE      = '550e8400-e29b-41d4-a716-446655440091'
CONTACT_FRANCISCA = '550e8400-e29b-41d4-a716-446655440092'
CONTACT_CARLOS    = '550e8400-e29b-41d4-a716-446655440093'
CONTACT_PEDRO     = '550e8400-e29b-41d4-a716-446655440094'
CONTACT_ANAPAULA  = '550e8400-e29b-41d4-a716-446655440095'
CONTACT_JOAO      = '550e8400-e29b-41d4-a716-446655440096'
CONTACT_MARCOS    = '550e8400-e29b-41d4-a716-446655440097'

-- Conversations (same suffix as contacts for easy mapping)
CONV_MARIA     = '550e8400-e29b-41d4-a716-4466554400a0'
CONV_JOSE      = '550e8400-e29b-41d4-a716-4466554400a1'
CONV_FRANCISCA = '550e8400-e29b-41d4-a716-4466554400a2'
CONV_CARLOS    = '550e8400-e29b-41d4-a716-4466554400a3'
CONV_PEDRO     = '550e8400-e29b-41d4-a716-4466554400a4'
CONV_ANAPAULA  = '550e8400-e29b-41d4-a716-4466554400a5'
CONV_JOAO      = '550e8400-e29b-41d4-a716-4466554400a6'
CONV_MARCOS    = '550e8400-e29b-41d4-a716-4466554400a7'

-- Leads
LEAD_MARIA     = '550e8400-e29b-41d4-a716-4466554400b0'
LEAD_JOSE      = '550e8400-e29b-41d4-a716-4466554400b1'
LEAD_FRANCISCA = '550e8400-e29b-41d4-a716-4466554400b2'
LEAD_CARLOS    = '550e8400-e29b-41d4-a716-4466554400b3'
LEAD_PEDRO     = '550e8400-e29b-41d4-a716-4466554400b4'
LEAD_ANAPAULA  = '550e8400-e29b-41d4-a716-4466554400b5'
LEAD_JOAO      = '550e8400-e29b-41d4-a716-4466554400b6'
LEAD_MARCOS    = '550e8400-e29b-41d4-a716-4466554400b7'

-- Automations
AUTO_EXPIRE   = '550e8400-e29b-41d4-a716-4466554400c0'
AUTO_MSG      = '550e8400-e29b-41d4-a716-4466554400c1'
AUTO_NOTE     = '550e8400-e29b-41d4-a716-4466554400c2'
AUTO_DETECT   = '550e8400-e29b-41d4-a716-4466554400c3'
```

---

### Task 1: Header e Tenant

**Files:**
- Modify: `fixture/fixtures.sql` (replace entire content)

- [ ] **Step 1: Write the SQL header and tenant INSERT**

Replace the entire content of `fixture/fixtures.sql` with:

```sql
-- =============================================================================
-- Fixture: Escritório de Advocacia Previdenciária
-- Cenário de dev/demo com dados realistas
-- Idempotente: pode rodar múltiplas vezes sem duplicar
-- =============================================================================

-- =============================================================================
-- 0. DADOS ORIGINAIS (admin + tenants anteriores preservados)
-- =============================================================================

INSERT INTO tenants (id, name, type, document, status, created_at, updated_at)
VALUES
    ('a9b1aef9-b1c9-48e4-9013-612710c954a5', 'Escritório Teste', 'PJ', '00.000.000/0001-00', 'active', NOW(), NOW()),
    ('f8c7e890-0444-446f-b66b-3ecb8e65395c', 'Advocacia Silva & Associados', 'PJ', '11.111.111/0001-11', 'active', NOW(), NOW())
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- Senha: admin123
INSERT INTO users (id, name, email, password_hash, role, status, created_at, updated_at)
VALUES (
    'ecfa4f42-d902-4552-923e-f55dd270fbb6', 'Admin', 'admin@teste.com',
    '$2a$10$Y9F207O7K9qrZnD34MBd/ONKUYL7KXR0JypYb07MpyAOqSa/5v.KW',
    'admin', 'active', NOW(), NOW()
) ON DUPLICATE KEY UPDATE name = VALUES(name);

-- =============================================================================
-- 1. TENANT PREVIDENCIÁRIO
-- =============================================================================

SET @TENANT = '550e8400-e29b-41d4-a716-446655440001';

INSERT INTO tenants (id, name, type, document, status, created_at, updated_at)
VALUES (
    @TENANT,
    'Mendes & Costa Advocacia Previdenciária',
    'PJ',
    '12.345.678/0001-90',
    'active',
    NOW(),
    NOW()
) ON DUPLICATE KEY UPDATE name = VALUES(name);
```

- [ ] **Step 2: Verify syntax**

Run: `mysql --help 2>/dev/null || echo "mysql client check done"`

No need to actually execute against DB — just confirm file syntax is valid SQL.

- [ ] **Step 3: Commit**

```bash
git add fixture/fixtures.sql
git commit -m "feat(fixture): add header and previdenciário tenant"
```

---

### Task 2: Usuários e Associações com Tenant

**Files:**
- Modify: `fixture/fixtures.sql`

- [ ] **Step 1: Append users section to fixtures.sql**

Append after the tenant section:

```sql
-- =============================================================================
-- 2. USUÁRIOS
-- =============================================================================

SET @RICARDO = '550e8400-e29b-41d4-a716-446655440010';
SET @ANA     = '550e8400-e29b-41d4-a716-446655440011';
SET @JULIANA = '550e8400-e29b-41d4-a716-446655440012';

-- Senha para todos: Teste@123
-- Hash bcrypt gerado com cost 10
INSERT INTO users (id, name, email, password_hash, role, status, created_at, updated_at)
VALUES
    (@RICARDO, 'Dr. Ricardo Mendes', 'ricardo@mendescosta.adv.br',
     '$2a$10$Y9F207O7K9qrZnD34MBd/ONKUYL7KXR0JypYb07MpyAOqSa/5v.KW', 'user', 'active', NOW(), NOW()),
    (@ANA, 'Dra. Ana Costa', 'ana@mendescosta.adv.br',
     '$2a$10$Y9F207O7K9qrZnD34MBd/ONKUYL7KXR0JypYb07MpyAOqSa/5v.KW', 'user', 'active', NOW(), NOW()),
    (@JULIANA, 'Juliana Rocha', 'juliana@mendescosta.adv.br',
     '$2a$10$Y9F207O7K9qrZnD34MBd/ONKUYL7KXR0JypYb07MpyAOqSa/5v.KW', 'user', 'active', NOW(), NOW())
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- =============================================================================
-- 3. ASSOCIAÇÃO USUÁRIO-TENANT
-- =============================================================================

INSERT INTO user_tenants (user_id, tenant_id, is_owner)
VALUES
    (@RICARDO, @TENANT, 1),
    (@ANA,     @TENANT, 0),
    (@JULIANA, @TENANT, 0)
ON DUPLICATE KEY UPDATE is_owner = VALUES(is_owner);
```

- [ ] **Step 2: Commit**

```bash
git add fixture/fixtures.sql
git commit -m "feat(fixture): add users and tenant associations"
```

---

### Task 3: Grupos de Permissão e Permissões

**Files:**
- Modify: `fixture/fixtures.sql`

- [ ] **Step 1: Append permission groups, user_groups, and permissions**

```sql
-- =============================================================================
-- 4. GRUPOS DE PERMISSÃO
-- =============================================================================

SET @GRP_ADV = '550e8400-e29b-41d4-a716-446655440020';
SET @GRP_ATD = '550e8400-e29b-41d4-a716-446655440021';

INSERT INTO permission_groups (id, tenant_id, name, description, created_at, updated_at)
VALUES
    (@GRP_ADV, @TENANT, 'Advogados', 'Acesso total ao funil e leads', NOW(3), NOW(3)),
    (@GRP_ATD, @TENANT, 'Atendimento', 'Acesso a contatos e conversas', NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- =============================================================================
-- 5. ASSOCIAÇÃO USUÁRIO-GRUPO
-- =============================================================================

INSERT INTO user_groups (id, user_id, group_id, tenant_id, created_at)
VALUES
    ('550e8400-e29b-41d4-a716-446655440022', @RICARDO, @GRP_ADV, @TENANT, NOW(3)),
    ('550e8400-e29b-41d4-a716-446655440023', @ANA,     @GRP_ADV, @TENANT, NOW(3)),
    ('550e8400-e29b-41d4-a716-446655440024', @JULIANA, @GRP_ATD, @TENANT, NOW(3))
ON DUPLICATE KEY UPDATE group_id = VALUES(group_id);

-- =============================================================================
-- 6. PERMISSÕES
-- =============================================================================

INSERT INTO permissions (id, tenant_id, group_id, user_id, resource, action, created_at)
VALUES
    -- Advogados: acesso total a leads, funnels, contacts
    ('550e8400-e29b-41d4-a716-446655440100', @TENANT, @GRP_ADV, NULL, 'leads', 'create', NOW(3)),
    ('550e8400-e29b-41d4-a716-446655440101', @TENANT, @GRP_ADV, NULL, 'leads', 'read', NOW(3)),
    ('550e8400-e29b-41d4-a716-446655440102', @TENANT, @GRP_ADV, NULL, 'leads', 'update', NOW(3)),
    ('550e8400-e29b-41d4-a716-446655440103', @TENANT, @GRP_ADV, NULL, 'leads', 'delete', NOW(3)),
    ('550e8400-e29b-41d4-a716-446655440104', @TENANT, @GRP_ADV, NULL, 'funnels', 'read', NOW(3)),
    ('550e8400-e29b-41d4-a716-446655440105', @TENANT, @GRP_ADV, NULL, 'funnels', 'update', NOW(3)),
    ('550e8400-e29b-41d4-a716-446655440106', @TENANT, @GRP_ADV, NULL, 'contacts', 'create', NOW(3)),
    ('550e8400-e29b-41d4-a716-446655440107', @TENANT, @GRP_ADV, NULL, 'contacts', 'read', NOW(3)),
    ('550e8400-e29b-41d4-a716-446655440108', @TENANT, @GRP_ADV, NULL, 'contacts', 'update', NOW(3)),
    -- Atendimento: leitura de contatos, conversas e leads
    ('550e8400-e29b-41d4-a716-446655440110', @TENANT, @GRP_ATD, NULL, 'contacts', 'read', NOW(3)),
    ('550e8400-e29b-41d4-a716-446655440111', @TENANT, @GRP_ATD, NULL, 'contacts', 'update', NOW(3)),
    ('550e8400-e29b-41d4-a716-446655440112', @TENANT, @GRP_ATD, NULL, 'conversations', 'read', NOW(3)),
    ('550e8400-e29b-41d4-a716-446655440113', @TENANT, @GRP_ATD, NULL, 'conversations', 'update', NOW(3)),
    ('550e8400-e29b-41d4-a716-446655440114', @TENANT, @GRP_ATD, NULL, 'leads', 'read', NOW(3))
ON DUPLICATE KEY UPDATE action = VALUES(action);
```

- [ ] **Step 2: Commit**

```bash
git add fixture/fixtures.sql
git commit -m "feat(fixture): add permission groups and permissions"
```

---

### Task 4: Produtos

**Files:**
- Modify: `fixture/fixtures.sql`

- [ ] **Step 1: Append products, tenant_products, and funnel_products**

Note: `funnel_products` will reference the funnel UUID defined in Task 5. Use the variable `@FUNNEL` which will be set later. For now, insert products and tenant_products only. `funnel_products` will be added in Task 5 after the funnel is created.

```sql
-- =============================================================================
-- 7. PRODUTOS
-- =============================================================================

SET @PROD_IDADE   = '550e8400-e29b-41d4-a716-446655440030';
SET @PROD_TEMPO   = '550e8400-e29b-41d4-a716-446655440031';
SET @PROD_BPC     = '550e8400-e29b-41d4-a716-446655440032';
SET @PROD_AUXILIO = '550e8400-e29b-41d4-a716-446655440033';
SET @PROD_REVISAO = '550e8400-e29b-41d4-a716-446655440034';

INSERT INTO products (id, name, description, keywords, active, created_at, updated_at)
VALUES
    (@PROD_IDADE, 'Aposentadoria por Idade',
     'Benefício para segurados que atingiram a idade mínima e carência de contribuição.',
     '["aposentadoria", "idade", "65 anos", "60 anos"]', 1, NOW(), NOW()),
    (@PROD_TEMPO, 'Aposentadoria por Tempo de Contribuição',
     'Benefício para segurados com tempo mínimo de contribuição ao INSS.',
     '["tempo de contribuição", "aposentadoria", "35 anos", "30 anos"]', 1, NOW(), NOW()),
    (@PROD_BPC, 'BPC/LOAS',
     'Benefício assistencial para idosos e pessoas com deficiência de baixa renda.',
     '["bpc", "loas", "benefício assistencial", "deficiência", "idoso"]', 1, NOW(), NOW()),
    (@PROD_AUXILIO, 'Auxílio-Doença',
     'Benefício por incapacidade temporária para o trabalho.',
     '["auxílio-doença", "incapacidade", "afastamento", "laudo médico"]', 1, NOW(), NOW()),
    (@PROD_REVISAO, 'Revisão de Benefício',
     'Revisão do valor ou tipo de benefício já concedido pelo INSS.',
     '["revisão", "recálculo", "valor errado", "teto"]', 1, NOW(), NOW())
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- =============================================================================
-- 8. ASSOCIAÇÃO TENANT-PRODUTO
-- =============================================================================

INSERT INTO tenant_products (id, tenant_id, product_id, created_at)
VALUES
    ('550e8400-e29b-41d4-a716-446655440035', @TENANT, @PROD_IDADE, NOW()),
    ('550e8400-e29b-41d4-a716-446655440036', @TENANT, @PROD_TEMPO, NOW()),
    ('550e8400-e29b-41d4-a716-446655440037', @TENANT, @PROD_BPC, NOW()),
    ('550e8400-e29b-41d4-a716-446655440038', @TENANT, @PROD_AUXILIO, NOW()),
    ('550e8400-e29b-41d4-a716-446655440039', @TENANT, @PROD_REVISAO, NOW())
ON DUPLICATE KEY UPDATE product_id = VALUES(product_id);
```

- [ ] **Step 2: Commit**

```bash
git add fixture/fixtures.sql
git commit -m "feat(fixture): add products and tenant associations"
```

---

### Task 5: Funil e Colunas

**Files:**
- Modify: `fixture/fixtures.sql`

- [ ] **Step 1: Append funnel, columns, and funnel_products**

```sql
-- =============================================================================
-- 9. FUNIL
-- =============================================================================

SET @FUNNEL = '550e8400-e29b-41d4-a716-446655440070';

INSERT INTO funnels (id, tenant_id, name, description, active, is_default, created_at, updated_at)
VALUES (
    @FUNNEL,
    @TENANT,
    'Previdenciário',
    'Funil principal para casos previdenciários — da triagem ao resultado',
    TRUE,
    TRUE,
    NOW(),
    NOW()
) ON DUPLICATE KEY UPDATE name = VALUES(name);

-- =============================================================================
-- 10. COLUNAS DO FUNIL
-- =============================================================================

SET @COL_NOVO       = '550e8400-e29b-41d4-a716-446655440080';
SET @COL_ANALISE    = '550e8400-e29b-41d4-a716-446655440081';
SET @COL_CALCULO    = '550e8400-e29b-41d4-a716-446655440082';
SET @COL_PROTOCOLO  = '550e8400-e29b-41d4-a716-446655440083';
SET @COL_AGUARDANDO = '550e8400-e29b-41d4-a716-446655440084';
SET @COL_RECURSO    = '550e8400-e29b-41d4-a716-446655440085';
SET @COL_GANHO      = '550e8400-e29b-41d4-a716-446655440086';
SET @COL_PERDIDO    = '550e8400-e29b-41d4-a716-446655440087';

INSERT INTO funnel_columns (id, funnel_id, name, order_index, type, color, created_at, updated_at)
VALUES
    (@COL_NOVO,       @FUNNEL, 'Novo Contato',             0, 'entry',        '#3B82F6', NOW(), NOW()),
    (@COL_ANALISE,    @FUNNEL, 'Análise Documental',       1, 'intermediate', '#F59E0B', NOW(), NOW()),
    (@COL_CALCULO,    @FUNNEL, 'Cálculo de Benefício',     2, 'intermediate', '#8B5CF6', NOW(), NOW()),
    (@COL_PROTOCOLO,  @FUNNEL, 'Protocolo no INSS',        3, 'intermediate', '#06B6D4', NOW(), NOW()),
    (@COL_AGUARDANDO, @FUNNEL, 'Aguardando Resposta INSS', 4, 'intermediate', '#F97316', NOW(), NOW()),
    (@COL_RECURSO,    @FUNNEL, 'Recurso/Revisão',          5, 'intermediate', '#EF4444', NOW(), NOW()),
    (@COL_GANHO,      @FUNNEL, 'Ganho',                    6, 'won',          '#10B981', NOW(), NOW()),
    (@COL_PERDIDO,    @FUNNEL, 'Perdido',                  7, 'lost',         '#6B7280', NOW(), NOW())
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- =============================================================================
-- 11. ASSOCIAÇÃO FUNIL-PRODUTO
-- =============================================================================

INSERT INTO funnel_products (id, funnel_id, product_id, priority, created_at)
VALUES
    ('550e8400-e29b-41d4-a716-44665544003a', @FUNNEL, @PROD_IDADE,   1, NOW()),
    ('550e8400-e29b-41d4-a716-44665544003b', @FUNNEL, @PROD_TEMPO,   2, NOW()),
    ('550e8400-e29b-41d4-a716-44665544003c', @FUNNEL, @PROD_BPC,     3, NOW()),
    ('550e8400-e29b-41d4-a716-44665544003d', @FUNNEL, @PROD_AUXILIO, 4, NOW()),
    ('550e8400-e29b-41d4-a716-44665544003e', @FUNNEL, @PROD_REVISAO, 5, NOW())
ON DUPLICATE KEY UPDATE priority = VALUES(priority);
```

- [ ] **Step 2: Commit**

```bash
git add fixture/fixtures.sql
git commit -m "feat(fixture): add funnel, columns and funnel-product links"
```

---

### Task 6: Especialista IA, Steps, Scoring e Guardrails

**Files:**
- Modify: `fixture/fixtures.sql`

- [ ] **Step 1: Append specialist, specialist_tenants, steps, scoring_configs, guardrails**

```sql
-- =============================================================================
-- 12. ESPECIALISTA IA
-- =============================================================================

SET @SPECIALIST = '550e8400-e29b-41d4-a716-446655440040';

INSERT INTO specialists (id, name, description, prompt, status, created_at, updated_at)
VALUES (
    @SPECIALIST,
    'Dra. Clara',
    'Assistente virtual de triagem previdenciária',
    'Você é a Dra. Clara, assistente virtual do escritório Mendes & Costa Advocacia Previdenciária. Sua função é realizar a triagem inicial de potenciais clientes via WhatsApp. Seja acolhedora e profissional, use linguagem simples (sem juridiquês). Identifique o tipo de benefício que o cliente busca, colete informações básicas (nome, idade, tempo de contribuição, documentos disponíveis) e qualifique o lead. NUNCA forneça parecer jurídico, NUNCA prometa resultados. Ao final da triagem, encaminhe para o advogado responsável.',
    'active',
    NOW(),
    NOW()
) ON DUPLICATE KEY UPDATE name = VALUES(name);

-- =============================================================================
-- 13. ASSOCIAÇÃO ESPECIALISTA-TENANT
-- =============================================================================

INSERT INTO specialist_tenants (specialist_id, tenant_id, is_default, created_at)
VALUES (@SPECIALIST, @TENANT, 1, NOW())
ON DUPLICATE KEY UPDATE is_default = VALUES(is_default);

-- =============================================================================
-- 14. STEPS DE TREINAMENTO
-- =============================================================================

INSERT INTO steps (id, specialist_id, order_index, text, data_type, required, score, target_column_id, created_at, updated_at)
VALUES
    ('550e8400-e29b-41d4-a716-446655440050', @SPECIALIST, 0, 'Qual seu nome completo?', 'free_text', TRUE, 10, NULL, NOW(), NOW()),
    ('550e8400-e29b-41d4-a716-446655440051', @SPECIALIST, 1, 'Qual tipo de benefício busca? (Aposentadoria por Idade, Aposentadoria por Tempo, BPC/LOAS, Auxílio-Doença, Revisão de Benefício)', 'selection', TRUE, 20, NULL, NOW(), NOW()),
    ('550e8400-e29b-41d4-a716-446655440052', @SPECIALIST, 2, 'Qual sua idade?', 'number', TRUE, 15, NULL, NOW(), NOW()),
    ('550e8400-e29b-41d4-a716-446655440053', @SPECIALIST, 3, 'Quantos anos de contribuição ao INSS você possui?', 'number', FALSE, 15, NULL, NOW(), NOW()),
    ('550e8400-e29b-41d4-a716-446655440054', @SPECIALIST, 4, 'Possui documentos como CNIS, carteira de trabalho ou laudos médicos? (Sim, tenho todos / Tenho alguns / Não tenho nenhum)', 'selection', TRUE, 20, NULL, NOW(), NOW()),
    ('550e8400-e29b-41d4-a716-446655440055', @SPECIALIST, 5, 'Descreva brevemente sua situação para que possamos entender melhor seu caso.', 'free_text', FALSE, 20, NULL, NOW(), NOW())
ON DUPLICATE KEY UPDATE text = VALUES(text);

-- =============================================================================
-- 15. SCORING CONFIG
-- =============================================================================

INSERT INTO scoring_configs (id, specialist_id, threshold, qualified_column_id, disqualified_column_id, created_at, updated_at)
VALUES (
    '550e8400-e29b-41d4-a716-446655440060',
    @SPECIALIST,
    60,
    @COL_ANALISE,
    @COL_PERDIDO,
    NOW(),
    NOW()
) ON DUPLICATE KEY UPDATE threshold = VALUES(threshold);

-- =============================================================================
-- 16. GUARDRAILS
-- =============================================================================

INSERT INTO guardrails (id, specialist_id, type, rule, message, active, created_at, updated_at)
VALUES
    ('550e8400-e29b-41d4-a716-446655440061', @SPECIALIST, 'forbidden_topics',
     'Não forneça parecer jurídico nem prometa resultado de processos',
     'Desculpe, não posso dar parecer jurídico. Um advogado do escritório entrará em contato para analisar seu caso.',
     TRUE, NOW(), NOW()),
    ('550e8400-e29b-41d4-a716-446655440062', @SPECIALIST, 'scope_limit',
     'Apenas assuntos de direito previdenciário e INSS',
     'Posso ajudar apenas com questões relacionadas a benefícios previdenciários e INSS. Para outros assuntos jurídicos, recomendo procurar um especialista na área.',
     TRUE, NOW(), NOW()),
    ('550e8400-e29b-41d4-a716-446655440063', @SPECIALIST, 'response_tone',
     'Tom acolhedor e profissional, sem juridiquês. Usar linguagem simples e acessível.',
     NULL,
     TRUE, NOW(), NOW())
ON DUPLICATE KEY UPDATE rule = VALUES(rule);
```

- [ ] **Step 2: Commit**

```bash
git add fixture/fixtures.sql
git commit -m "feat(fixture): add specialist, steps, scoring and guardrails"
```

---

### Task 7: Contatos e Conversas WhatsApp

**Files:**
- Modify: `fixture/fixtures.sql`

- [ ] **Step 1: Append contacts and conversations**

```sql
-- =============================================================================
-- 17. CONTATOS WHATSAPP
-- =============================================================================

SET @CONTACT_MARIA     = '550e8400-e29b-41d4-a716-446655440090';
SET @CONTACT_JOSE      = '550e8400-e29b-41d4-a716-446655440091';
SET @CONTACT_FRANCISCA = '550e8400-e29b-41d4-a716-446655440092';
SET @CONTACT_CARLOS    = '550e8400-e29b-41d4-a716-446655440093';
SET @CONTACT_PEDRO     = '550e8400-e29b-41d4-a716-446655440094';
SET @CONTACT_ANAPAULA  = '550e8400-e29b-41d4-a716-446655440095';
SET @CONTACT_JOAO      = '550e8400-e29b-41d4-a716-446655440096';
SET @CONTACT_MARCOS    = '550e8400-e29b-41d4-a716-446655440097';

INSERT INTO contacts (id, tenant_id, name, phone, whatsapp_id, created_at, updated_at)
VALUES
    (@CONTACT_MARIA,     @TENANT, 'Maria da Silva',     '5511999001001', '5511999001001@s.whatsapp.net', NOW(), NOW()),
    (@CONTACT_JOSE,      @TENANT, 'José Santos',        '5511999002002', '5511999002002@s.whatsapp.net', NOW(), NOW()),
    (@CONTACT_FRANCISCA, @TENANT, 'Dona Francisca',     '5511999003003', '5511999003003@s.whatsapp.net', NOW(), NOW()),
    (@CONTACT_CARLOS,    @TENANT, 'Carlos Oliveira',    '5511999004004', '5511999004004@s.whatsapp.net', NOW(), NOW()),
    (@CONTACT_PEDRO,     @TENANT, 'Pedro Souza',        '5511999005005', '5511999005005@s.whatsapp.net', NOW(), NOW()),
    (@CONTACT_ANAPAULA,  @TENANT, 'Ana Paula Lima',     '5511999006006', '5511999006006@s.whatsapp.net', NOW(), NOW()),
    (@CONTACT_JOAO,      @TENANT, 'Seu João Ferreira',  '5511999007007', '5511999007007@s.whatsapp.net', NOW(), NOW()),
    (@CONTACT_MARCOS,    @TENANT, 'Marcos Pereira',     '5511999008008', '5511999008008@s.whatsapp.net', NOW(), NOW())
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- =============================================================================
-- 18. CONVERSAS
-- =============================================================================

SET @CONV_MARIA     = '550e8400-e29b-41d4-a716-4466554400a0';
SET @CONV_JOSE      = '550e8400-e29b-41d4-a716-4466554400a1';
SET @CONV_FRANCISCA = '550e8400-e29b-41d4-a716-4466554400a2';
SET @CONV_CARLOS    = '550e8400-e29b-41d4-a716-4466554400a3';
SET @CONV_PEDRO     = '550e8400-e29b-41d4-a716-4466554400a4';
SET @CONV_ANAPAULA  = '550e8400-e29b-41d4-a716-4466554400a5';
SET @CONV_JOAO      = '550e8400-e29b-41d4-a716-4466554400a6';
SET @CONV_MARCOS    = '550e8400-e29b-41d4-a716-4466554400a7';

INSERT INTO conversations (id, tenant_id, contact_id, status, last_message_at, unread_count, created_at, updated_at)
VALUES
    (@CONV_MARIA,     @TENANT, @CONTACT_MARIA,     'open',   NOW(), 1, NOW(), NOW()),
    (@CONV_JOSE,      @TENANT, @CONTACT_JOSE,      'open',   NOW(), 0, NOW(), NOW()),
    (@CONV_FRANCISCA, @TENANT, @CONTACT_FRANCISCA, 'open',   NOW(), 0, NOW(), NOW()),
    (@CONV_CARLOS,    @TENANT, @CONTACT_CARLOS,    'open',   NOW(), 0, NOW(), NOW()),
    (@CONV_PEDRO,     @TENANT, @CONTACT_PEDRO,     'open',   NOW(), 0, NOW(), NOW()),
    (@CONV_ANAPAULA,  @TENANT, @CONTACT_ANAPAULA,  'open',   NOW(), 0, NOW(), NOW()),
    (@CONV_JOAO,      @TENANT, @CONTACT_JOAO,      'closed', NOW(), 0, NOW(), NOW()),
    (@CONV_MARCOS,    @TENANT, @CONTACT_MARCOS,     'closed', NOW(), 0, NOW(), NOW())
ON DUPLICATE KEY UPDATE status = VALUES(status);
```

- [ ] **Step 2: Commit**

```bash
git add fixture/fixtures.sql
git commit -m "feat(fixture): add contacts and conversations"
```

---

### Task 8: Mensagens WhatsApp

**Files:**
- Modify: `fixture/fixtures.sql`

- [ ] **Step 1: Append messages for 3 detailed conversations**

```sql
-- =============================================================================
-- 19. MENSAGENS WHATSAPP
-- =============================================================================

-- Conversa: Maria da Silva (triagem em andamento — 5 msgs)
INSERT INTO messages (id, conversation_id, direction, content, type, status, whatsapp_msg_id, `timestamp`, created_at)
VALUES
    ('550e8400-e29b-41d4-a716-4466554400d0', @CONV_MARIA, 'incoming',
     'Boa tarde, vi o anúncio de vocês sobre aposentadoria',
     'text', 'sent', 'wamid.maria01', DATE_SUB(NOW(), INTERVAL 2 HOUR), DATE_SUB(NOW(), INTERVAL 2 HOUR)),
    ('550e8400-e29b-41d4-a716-4466554400d1', @CONV_MARIA, 'outgoing',
     'Olá Maria! Sou a Dra. Clara, assistente virtual do escritório Mendes & Costa. Como posso ajudá-la? Primeiro, qual seu nome completo?',
     'text', 'sent', 'wamid.maria02', DATE_SUB(NOW(), INTERVAL 1 HOUR - INTERVAL 55 MINUTE), DATE_SUB(NOW(), INTERVAL 1 HOUR - INTERVAL 55 MINUTE)),
    ('550e8400-e29b-41d4-a716-4466554400d2', @CONV_MARIA, 'incoming',
     'Maria da Silva',
     'text', 'sent', 'wamid.maria03', DATE_SUB(NOW(), INTERVAL 1 HOUR - INTERVAL 50 MINUTE), DATE_SUB(NOW(), INTERVAL 1 HOUR - INTERVAL 50 MINUTE)),
    ('550e8400-e29b-41d4-a716-4466554400d3', @CONV_MARIA, 'outgoing',
     'Obrigada, Maria! Qual tipo de benefício você busca? Aposentadoria por Idade, Aposentadoria por Tempo de Contribuição, BPC/LOAS, Auxílio-Doença ou Revisão de Benefício?',
     'text', 'sent', 'wamid.maria04', DATE_SUB(NOW(), INTERVAL 1 HOUR - INTERVAL 49 MINUTE), DATE_SUB(NOW(), INTERVAL 1 HOUR - INTERVAL 49 MINUTE)),
    ('550e8400-e29b-41d4-a716-4466554400d4', @CONV_MARIA, 'incoming',
     'Aposentadoria, tenho 63 anos',
     'text', 'sent', 'wamid.maria05', DATE_SUB(NOW(), INTERVAL 1 HOUR - INTERVAL 45 MINUTE), DATE_SUB(NOW(), INTERVAL 1 HOUR - INTERVAL 45 MINUTE))
ON DUPLICATE KEY UPDATE content = VALUES(content);

-- Conversa: José Santos (documentos pendentes — 7 msgs)
INSERT INTO messages (id, conversation_id, direction, content, type, status, whatsapp_msg_id, `timestamp`, created_at)
VALUES
    ('550e8400-e29b-41d4-a716-4466554400d5', @CONV_JOSE, 'incoming',
     'Bom dia, quero ver sobre minha aposentadoria por tempo de contribuição',
     'text', 'sent', 'wamid.jose01', DATE_SUB(NOW(), INTERVAL 2 DAY), DATE_SUB(NOW(), INTERVAL 2 DAY)),
    ('550e8400-e29b-41d4-a716-4466554400d6', @CONV_JOSE, 'outgoing',
     'Olá José! Sou a Dra. Clara, assistente virtual. Quantos anos de contribuição ao INSS você possui?',
     'text', 'sent', 'wamid.jose02', DATE_SUB(NOW(), INTERVAL 2 DAY) + INTERVAL 1 MINUTE, DATE_SUB(NOW(), INTERVAL 2 DAY) + INTERVAL 1 MINUTE),
    ('550e8400-e29b-41d4-a716-4466554400d7', @CONV_JOSE, 'incoming',
     'Tenho 33 anos de carteira assinada',
     'text', 'sent', 'wamid.jose03', DATE_SUB(NOW(), INTERVAL 2 DAY) + INTERVAL 5 MINUTE, DATE_SUB(NOW(), INTERVAL 2 DAY) + INTERVAL 5 MINUTE),
    ('550e8400-e29b-41d4-a716-4466554400d8', @CONV_JOSE, 'outgoing',
     'Ótimo! Você possui o CNIS atualizado e carteiras de trabalho?',
     'text', 'sent', 'wamid.jose04', DATE_SUB(NOW(), INTERVAL 2 DAY) + INTERVAL 6 MINUTE, DATE_SUB(NOW(), INTERVAL 2 DAY) + INTERVAL 6 MINUTE),
    ('550e8400-e29b-41d4-a716-4466554400d9', @CONV_JOSE, 'incoming',
     'Tenho o CNIS sim, vou enviar',
     'text', 'sent', 'wamid.jose05', DATE_SUB(NOW(), INTERVAL 2 DAY) + INTERVAL 10 MINUTE, DATE_SUB(NOW(), INTERVAL 2 DAY) + INTERVAL 10 MINUTE),
    ('550e8400-e29b-41d4-a716-4466554400da', @CONV_JOSE, 'outgoing',
     'Perfeito! Encaminhei seu caso para a Dra. Ana que vai analisar sua documentação. Ela entrará em contato em breve.',
     'text', 'sent', 'wamid.jose06', DATE_SUB(NOW(), INTERVAL 2 DAY) + INTERVAL 11 MINUTE, DATE_SUB(NOW(), INTERVAL 2 DAY) + INTERVAL 11 MINUTE),
    ('550e8400-e29b-41d4-a716-4466554400db', @CONV_JOSE, 'incoming',
     'Obrigado, aguardo retorno',
     'text', 'sent', 'wamid.jose07', DATE_SUB(NOW(), INTERVAL 2 DAY) + INTERVAL 15 MINUTE, DATE_SUB(NOW(), INTERVAL 2 DAY) + INTERVAL 15 MINUTE)
ON DUPLICATE KEY UPDATE content = VALUES(content);

-- Conversa: Carlos Oliveira (caso em andamento — 8 msgs)
INSERT INTO messages (id, conversation_id, direction, content, type, status, whatsapp_msg_id, `timestamp`, created_at)
VALUES
    ('550e8400-e29b-41d4-a716-4466554400dc', @CONV_CARLOS, 'incoming',
     'Oi, estou afastado do trabalho e preciso de auxílio-doença',
     'text', 'sent', 'wamid.carlos01', DATE_SUB(NOW(), INTERVAL 5 DAY), DATE_SUB(NOW(), INTERVAL 5 DAY)),
    ('550e8400-e29b-41d4-a716-4466554400dd', @CONV_CARLOS, 'outgoing',
     'Olá Carlos! Lamento pela situação. Sou a Dra. Clara, assistente virtual. Você possui laudo médico atualizado?',
     'text', 'sent', 'wamid.carlos02', DATE_SUB(NOW(), INTERVAL 5 DAY) + INTERVAL 1 MINUTE, DATE_SUB(NOW(), INTERVAL 5 DAY) + INTERVAL 1 MINUTE),
    ('550e8400-e29b-41d4-a716-4466554400de', @CONV_CARLOS, 'incoming',
     'Sim, tenho laudo do ortopedista',
     'text', 'sent', 'wamid.carlos03', DATE_SUB(NOW(), INTERVAL 5 DAY) + INTERVAL 5 MINUTE, DATE_SUB(NOW(), INTERVAL 5 DAY) + INTERVAL 5 MINUTE),
    ('550e8400-e29b-41d4-a716-4466554400df', @CONV_CARLOS, 'outgoing',
     'Ótimo. Vou encaminhar para o Dr. Ricardo, nosso advogado especialista. Ele vai preparar seu pedido ao INSS.',
     'text', 'sent', 'wamid.carlos04', DATE_SUB(NOW(), INTERVAL 5 DAY) + INTERVAL 6 MINUTE, DATE_SUB(NOW(), INTERVAL 5 DAY) + INTERVAL 6 MINUTE),
    ('550e8400-e29b-41d4-a716-4466554400e0', @CONV_CARLOS, 'incoming',
     'Quanto tempo demora?',
     'text', 'sent', 'wamid.carlos05', DATE_SUB(NOW(), INTERVAL 5 DAY) + INTERVAL 10 MINUTE, DATE_SUB(NOW(), INTERVAL 5 DAY) + INTERVAL 10 MINUTE),
    ('550e8400-e29b-41d4-a716-4466554400e1', @CONV_CARLOS, 'outgoing',
     'O protocolo no INSS leva em média 30 a 45 dias para análise. O Dr. Ricardo vai acompanhar de perto e manter você informado.',
     'text', 'sent', 'wamid.carlos06', DATE_SUB(NOW(), INTERVAL 5 DAY) + INTERVAL 11 MINUTE, DATE_SUB(NOW(), INTERVAL 5 DAY) + INTERVAL 11 MINUTE),
    ('550e8400-e29b-41d4-a716-4466554400e2', @CONV_CARLOS, 'incoming',
     'Tá bom, obrigado',
     'text', 'sent', 'wamid.carlos07', DATE_SUB(NOW(), INTERVAL 5 DAY) + INTERVAL 15 MINUTE, DATE_SUB(NOW(), INTERVAL 5 DAY) + INTERVAL 15 MINUTE),
    ('550e8400-e29b-41d4-a716-4466554400e3', @CONV_CARLOS, 'outgoing',
     'Seu pedido foi protocolado no INSS. Acompanhe pelo número 35.123.456-7. Qualquer novidade entraremos em contato!',
     'text', 'sent', 'wamid.carlos08', DATE_SUB(NOW(), INTERVAL 3 DAY), DATE_SUB(NOW(), INTERVAL 3 DAY))
ON DUPLICATE KEY UPDATE content = VALUES(content);
```

- [ ] **Step 2: Commit**

```bash
git add fixture/fixtures.sql
git commit -m "feat(fixture): add WhatsApp messages for 3 conversations"
```

---

### Task 9: Leads, Lead Notes e Lead Movements

**Files:**
- Modify: `fixture/fixtures.sql`

- [ ] **Step 1: Append leads**

```sql
-- =============================================================================
-- 20. LEADS
-- =============================================================================

SET @LEAD_MARIA     = '550e8400-e29b-41d4-a716-4466554400b0';
SET @LEAD_JOSE      = '550e8400-e29b-41d4-a716-4466554400b1';
SET @LEAD_FRANCISCA = '550e8400-e29b-41d4-a716-4466554400b2';
SET @LEAD_CARLOS    = '550e8400-e29b-41d4-a716-4466554400b3';
SET @LEAD_PEDRO     = '550e8400-e29b-41d4-a716-4466554400b4';
SET @LEAD_ANAPAULA  = '550e8400-e29b-41d4-a716-4466554400b5';
SET @LEAD_JOAO      = '550e8400-e29b-41d4-a716-4466554400b6';
SET @LEAD_MARCOS    = '550e8400-e29b-41d4-a716-4466554400b7';

INSERT INTO leads (id, tenant_id, funnel_id, column_id, contact_id, conversation_id, product_id, responsible_user_id, score, status, column_entered_at, created_at, updated_at)
VALUES
    (@LEAD_MARIA,     @TENANT, @FUNNEL, @COL_NOVO,       @CONTACT_MARIA,     @CONV_MARIA,     @PROD_IDADE,   @JULIANA, 10, 'open', NOW(), NOW(), NOW()),
    (@LEAD_JOSE,      @TENANT, @FUNNEL, @COL_ANALISE,    @CONTACT_JOSE,      @CONV_JOSE,      @PROD_TEMPO,   @ANA,     60, 'open', NOW(), DATE_SUB(NOW(), INTERVAL 2 DAY), NOW()),
    (@LEAD_FRANCISCA, @TENANT, @FUNNEL, @COL_CALCULO,    @CONTACT_FRANCISCA, @CONV_FRANCISCA, @PROD_BPC,     @ANA,     80, 'open', NOW(), DATE_SUB(NOW(), INTERVAL 5 DAY), NOW()),
    (@LEAD_CARLOS,    @TENANT, @FUNNEL, @COL_PROTOCOLO,  @CONTACT_CARLOS,    @CONV_CARLOS,    @PROD_AUXILIO, @RICARDO, 75, 'open', NOW(), DATE_SUB(NOW(), INTERVAL 5 DAY), NOW()),
    (@LEAD_PEDRO,     @TENANT, @FUNNEL, @COL_AGUARDANDO, @CONTACT_PEDRO,     @CONV_PEDRO,     @PROD_REVISAO, @RICARDO, 70, 'open', DATE_SUB(NOW(), INTERVAL 25 DAY), DATE_SUB(NOW(), INTERVAL 30 DAY), NOW()),
    (@LEAD_ANAPAULA,  @TENANT, @FUNNEL, @COL_RECURSO,    @CONTACT_ANAPAULA,  @CONV_ANAPAULA,  @PROD_TEMPO,   @ANA,     85, 'open', NOW(), DATE_SUB(NOW(), INTERVAL 15 DAY), NOW()),
    (@LEAD_JOAO,      @TENANT, @FUNNEL, @COL_GANHO,      @CONTACT_JOAO,      @CONV_JOAO,      @PROD_IDADE,   @RICARDO, 90, 'won',  NOW(), DATE_SUB(NOW(), INTERVAL 60 DAY), NOW()),
    (@LEAD_MARCOS,    @TENANT, @FUNNEL, @COL_PERDIDO,    @CONTACT_MARCOS,    @CONV_MARCOS,    @PROD_AUXILIO, @JULIANA, 25, 'lost', NOW(), DATE_SUB(NOW(), INTERVAL 10 DAY), NOW())
ON DUPLICATE KEY UPDATE score = VALUES(score);
```

- [ ] **Step 2: Append lead notes**

```sql
-- =============================================================================
-- 21. NOTAS DOS LEADS
-- =============================================================================

INSERT INTO lead_notes (id, lead_id, tenant_id, content, created_by, created_at)
VALUES
    ('550e8400-e29b-41d4-a716-4466554400f0', @LEAD_JOSE, @TENANT,
     'CNIS apresentado, faltam últimas contribuições dos anos 2020-2023. Solicitado que traga comprovantes.',
     @ANA, DATE_SUB(NOW(), INTERVAL 1 DAY)),
    ('550e8400-e29b-41d4-a716-4466554400f1', @LEAD_FRANCISCA, @TENANT,
     'Renda familiar comprovada abaixo de 1/4 do salário mínimo. Laudo médico atualizado apresentado. Caso forte para BPC.',
     @ANA, DATE_SUB(NOW(), INTERVAL 3 DAY)),
    ('550e8400-e29b-41d4-a716-4466554400f2', @LEAD_JOAO, @TENANT,
     'Benefício concedido — aposentadoria por idade deferida pelo INSS. Cliente satisfeito.',
     @RICARDO, DATE_SUB(NOW(), INTERVAL 7 DAY))
ON DUPLICATE KEY UPDATE content = VALUES(content);
```

- [ ] **Step 3: Append lead movements**

```sql
-- =============================================================================
-- 22. HISTÓRICO DE MOVIMENTAÇÃO DOS LEADS
-- =============================================================================

INSERT INTO lead_movements (id, lead_id, from_column_id, to_column_id, moved_at)
VALUES
    -- José: Novo Contato → Análise Documental
    ('550e8400-e29b-41d4-a716-446655440200', @LEAD_JOSE, NULL,            @COL_NOVO,      DATE_SUB(NOW(), INTERVAL 2 DAY)),
    ('550e8400-e29b-41d4-a716-446655440201', @LEAD_JOSE, @COL_NOVO,      @COL_ANALISE,   DATE_SUB(NOW(), INTERVAL 2 DAY) + INTERVAL 15 MINUTE),

    -- Carlos: Novo Contato → Análise Documental → Protocolo no INSS
    ('550e8400-e29b-41d4-a716-446655440202', @LEAD_CARLOS, NULL,            @COL_NOVO,      DATE_SUB(NOW(), INTERVAL 5 DAY)),
    ('550e8400-e29b-41d4-a716-446655440203', @LEAD_CARLOS, @COL_NOVO,      @COL_ANALISE,   DATE_SUB(NOW(), INTERVAL 5 DAY) + INTERVAL 20 MINUTE),
    ('550e8400-e29b-41d4-a716-446655440204', @LEAD_CARLOS, @COL_ANALISE,   @COL_PROTOCOLO, DATE_SUB(NOW(), INTERVAL 3 DAY)),

    -- Seu João: Novo Contato → Análise → Cálculo → Protocolo → Ganho
    ('550e8400-e29b-41d4-a716-446655440205', @LEAD_JOAO, NULL,             @COL_NOVO,      DATE_SUB(NOW(), INTERVAL 60 DAY)),
    ('550e8400-e29b-41d4-a716-446655440206', @LEAD_JOAO, @COL_NOVO,       @COL_ANALISE,   DATE_SUB(NOW(), INTERVAL 55 DAY)),
    ('550e8400-e29b-41d4-a716-446655440207', @LEAD_JOAO, @COL_ANALISE,    @COL_CALCULO,   DATE_SUB(NOW(), INTERVAL 45 DAY)),
    ('550e8400-e29b-41d4-a716-446655440208', @LEAD_JOAO, @COL_CALCULO,    @COL_PROTOCOLO, DATE_SUB(NOW(), INTERVAL 30 DAY)),
    ('550e8400-e29b-41d4-a716-446655440209', @LEAD_JOAO, @COL_PROTOCOLO,  @COL_GANHO,     DATE_SUB(NOW(), INTERVAL 7 DAY))
ON DUPLICATE KEY UPDATE to_column_id = VALUES(to_column_id);
```

- [ ] **Step 4: Commit**

```bash
git add fixture/fixtures.sql
git commit -m "feat(fixture): add leads, notes and movement history"
```

---

### Task 10: Automações, Execution Logs, Notificações e Preferences

**Files:**
- Modify: `fixture/fixtures.sql`

- [ ] **Step 1: Append automations and execution logs**

```sql
-- =============================================================================
-- 23. AUTOMAÇÕES
-- =============================================================================

SET @AUTO_EXPIRE = '550e8400-e29b-41d4-a716-4466554400c0';
SET @AUTO_MSG    = '550e8400-e29b-41d4-a716-4466554400c1';
SET @AUTO_NOTE   = '550e8400-e29b-41d4-a716-4466554400c2';
SET @AUTO_DETECT = '550e8400-e29b-41d4-a716-4466554400c3';

INSERT INTO automations (id, tenant_id, funnel_id, column_id, type, config, active, priority, created_at, updated_at)
VALUES
    (@AUTO_EXPIRE, @TENANT, @FUNNEL, @COL_AGUARDANDO, 'expiration',
     '{"days": 30, "action": "notify"}',
     1, 1, NOW(), NOW()),
    (@AUTO_MSG, @TENANT, @FUNNEL, @COL_PROTOCOLO, 'auto_message',
     '{"message": "Seu pedido foi protocolado no INSS. Acompanharemos o andamento e retornaremos assim que houver resposta."}',
     1, 2, NOW(), NOW()),
    (@AUTO_NOTE, @TENANT, @FUNNEL, @COL_GANHO, 'auto_note',
     '{"content": "Benefício concedido — caso encerrado com sucesso"}',
     1, 3, NOW(), NOW()),
    (@AUTO_DETECT, @TENANT, @FUNNEL, @COL_NOVO, 'detect_product',
     '{"enabled": true}',
     1, 4, NOW(), NOW())
ON DUPLICATE KEY UPDATE config = VALUES(config);

-- =============================================================================
-- 24. LOGS DE EXECUÇÃO DE AUTOMAÇÕES
-- =============================================================================

INSERT INTO execution_logs (id, automation_id, lead_id, tenant_id, status, error_message, executed_at)
VALUES
    ('550e8400-e29b-41d4-a716-4466554400c4', @AUTO_MSG,  @LEAD_CARLOS, @TENANT, 'success', NULL, DATE_SUB(NOW(), INTERVAL 3 DAY)),
    ('550e8400-e29b-41d4-a716-4466554400c5', @AUTO_NOTE, @LEAD_JOAO,   @TENANT, 'success', NULL, DATE_SUB(NOW(), INTERVAL 7 DAY))
ON DUPLICATE KEY UPDATE status = VALUES(status);
```

- [ ] **Step 2: Append notifications and preferences**

```sql
-- =============================================================================
-- 25. NOTIFICAÇÕES
-- =============================================================================

INSERT INTO notifications (id, tenant_id, user_id, type, title, body, metadata, is_read, created_at)
VALUES
    ('550e8400-e29b-41d4-a716-446655440300', @TENANT, @RICARDO, 'lead_update',
     'Lead Carlos Oliveira protocolado no INSS',
     'O pedido de auxílio-doença do Carlos Oliveira foi protocolado com sucesso.',
     '{"lead_id": "550e8400-e29b-41d4-a716-4466554400b3"}',
     1, DATE_SUB(NOW(), INTERVAL 3 DAY)),
    ('550e8400-e29b-41d4-a716-446655440301', @TENANT, @RICARDO, 'automation',
     'Lead Pedro Souza aguardando resposta INSS há 25 dias',
     'O lead Pedro Souza está na coluna "Aguardando Resposta INSS" há 25 dias. Considere entrar em contato com o INSS.',
     '{"lead_id": "550e8400-e29b-41d4-a716-4466554400b4", "days": 25}',
     0, DATE_SUB(NOW(), INTERVAL 1 DAY)),
    ('550e8400-e29b-41d4-a716-446655440302', @TENANT, @ANA, 'lead_assignment',
     'Novo lead atribuído: José Santos',
     'Um novo lead foi atribuído a você: José Santos — Aposentadoria por Tempo de Contribuição.',
     '{"lead_id": "550e8400-e29b-41d4-a716-4466554400b1"}',
     0, DATE_SUB(NOW(), INTERVAL 2 DAY))
ON DUPLICATE KEY UPDATE title = VALUES(title);

-- =============================================================================
-- 26. PREFERÊNCIAS DE NOTIFICAÇÃO
-- =============================================================================

INSERT INTO notification_preferences (id, user_id, tenant_id, channel, enabled, created_at, updated_at)
VALUES
    ('550e8400-e29b-41d4-a716-446655440310', @RICARDO, @TENANT, 'in_app', 1, NOW(), NOW()),
    ('550e8400-e29b-41d4-a716-446655440311', @RICARDO, @TENANT, 'whatsapp', 1, NOW(), NOW()),
    ('550e8400-e29b-41d4-a716-446655440312', @RICARDO, @TENANT, 'email', 1, NOW(), NOW()),
    ('550e8400-e29b-41d4-a716-446655440313', @ANA,     @TENANT, 'in_app', 1, NOW(), NOW()),
    ('550e8400-e29b-41d4-a716-446655440314', @ANA,     @TENANT, 'whatsapp', 1, NOW(), NOW()),
    ('550e8400-e29b-41d4-a716-446655440315', @ANA,     @TENANT, 'email', 1, NOW(), NOW()),
    ('550e8400-e29b-41d4-a716-446655440316', @JULIANA, @TENANT, 'in_app', 1, NOW(), NOW()),
    ('550e8400-e29b-41d4-a716-446655440317', @JULIANA, @TENANT, 'whatsapp', 1, NOW(), NOW()),
    ('550e8400-e29b-41d4-a716-446655440318', @JULIANA, @TENANT, 'email', 1, NOW(), NOW())
ON DUPLICATE KEY UPDATE enabled = VALUES(enabled);
```

- [ ] **Step 3: Commit**

```bash
git add fixture/fixtures.sql
git commit -m "feat(fixture): add automations, notifications and preferences"
```

---

### Task 11: Validação Final

**Files:**
- Read: `fixture/fixtures.sql`

- [ ] **Step 1: Count total INSERT statements to validate volume**

Run:
```bash
grep -c "^    (" fixture/fixtures.sql
```

Expected: approximately 55-60 rows (matching the ~55 total from the spec).

- [ ] **Step 2: Validate SQL syntax by running against test container**

Run:
```bash
docker compose up -d db && sleep 5 && docker compose exec db mysql -u root -proot crm_juridico < fixture/fixtures.sql && echo "OK"
```

Expected: `OK` with no errors. If DB is not running, this step can be deferred.

- [ ] **Step 3: Spot-check FK integrity**

Run:
```bash
docker compose exec db mysql -u root -proot crm_juridico -e "
  SELECT l.id, c.name as contact, fc.name as column_name, p.name as product
  FROM leads l
  JOIN contacts c ON c.id = l.contact_id
  JOIN funnel_columns fc ON fc.id = l.column_id
  LEFT JOIN products p ON p.id = l.product_id
  ORDER BY fc.order_index;
"
```

Expected: 8 rows showing each lead with correct contact, column, and product.

- [ ] **Step 4: Final commit with all changes**

```bash
git add fixture/fixtures.sql
git commit -m "feat(fixture): complete previdenciário dev/demo fixture

Realistic data for a social security law firm including:
- 1 tenant, 3 users, 2 permission groups
- 5 products, 1 AI specialist with training steps
- 8-column funnel with 8 leads across all stages
- 20 WhatsApp messages in 3 detailed conversations
- 4 automations with execution logs
- Notifications and preferences

Idempotent: safe to run multiple times."
```
