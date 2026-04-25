-- =============================================================================
-- Fixture: Escritório Teste — simulação de escritório previdenciário completo
-- Idempotente (ON DUPLICATE KEY UPDATE) | Rodar DEPOIS de fixtures.sql
-- =============================================================================

SET @TENANT = 'a9b1aef9-b1c9-48e4-9013-612710c954a5';
SET @RICARDO = '550e8400-e29b-41d4-a716-446655440010';
SET @ANA     = '550e8400-e29b-41d4-a716-446655440011';
SET @JULIANA = '550e8400-e29b-41d4-a716-446655440012';

-- Produtos globais (criados em fixtures.sql — aqui só referenciamos)
SET @PROD_IDADE   = '550e8400-e29b-41d4-a716-446655440030';
SET @PROD_TEMPO   = '550e8400-e29b-41d4-a716-446655440031';
SET @PROD_BPC     = '550e8400-e29b-41d4-a716-446655440032';
SET @PROD_AUXILIO = '550e8400-e29b-41d4-a716-446655440033';
SET @PROD_REVISAO = '550e8400-e29b-41d4-a716-446655440034';

-- Limpeza de versão anterior (simples) do Escritório Teste
DELETE FROM scoring_configs  WHERE specialist_id IN ('550e8400-e29b-41d4-a716-446655440050');
DELETE FROM steps            WHERE specialist_id IN ('550e8400-e29b-41d4-a716-446655440050');
DELETE FROM guardrails       WHERE specialist_id IN ('550e8400-e29b-41d4-a716-446655440050');
DELETE FROM funnel_products  WHERE funnel_id = 'e57e0001-0003-0000-0000-000000000001';
DELETE FROM funnel_columns   WHERE funnel_id = 'e57e0001-0003-0000-0000-000000000001';
DELETE FROM funnels          WHERE id = 'e57e0001-0003-0000-0000-000000000001';

-- =============================================================================
-- 1. USUÁRIOS NO TENANT + GRUPOS + PERMISSÕES
-- =============================================================================
INSERT INTO user_tenants (user_id, tenant_id, is_owner) VALUES
    (@RICARDO, @TENANT, 1),
    (@ANA,     @TENANT, 0),
    (@JULIANA, @TENANT, 0)
ON DUPLICATE KEY UPDATE is_owner = VALUES(is_owner);

SET @GRP_ADV = 'e57e0001-0000-0000-0001-000000000001';
SET @GRP_ATD = 'e57e0001-0000-0000-0001-000000000002';

INSERT INTO permission_groups (id, tenant_id, name, description, created_at, updated_at) VALUES
    (@GRP_ADV, @TENANT, 'Advogados',   'Acesso total ao funil e leads',      NOW(3), NOW(3)),
    (@GRP_ATD, @TENANT, 'Atendimento', 'Acesso a contatos e conversas',      NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE name = VALUES(name);

INSERT INTO user_groups (id, user_id, group_id, tenant_id, created_at) VALUES
    ('e57e0001-0001-0000-0000-000000000001', @RICARDO, @GRP_ADV, @TENANT, NOW(3)),
    ('e57e0001-0001-0000-0000-000000000002', @ANA,     @GRP_ADV, @TENANT, NOW(3)),
    ('e57e0001-0001-0000-0000-000000000003', @JULIANA, @GRP_ATD, @TENANT, NOW(3))
ON DUPLICATE KEY UPDATE group_id = VALUES(group_id);

INSERT INTO permissions (id, tenant_id, group_id, user_id, resource, action, created_at) VALUES
    ('e57e0001-0002-0000-0000-000000000001', @TENANT, @GRP_ADV, NULL, 'leads',         'create', NOW(3)),
    ('e57e0001-0002-0000-0000-000000000002', @TENANT, @GRP_ADV, NULL, 'leads',         'read',   NOW(3)),
    ('e57e0001-0002-0000-0000-000000000003', @TENANT, @GRP_ADV, NULL, 'leads',         'update', NOW(3)),
    ('e57e0001-0002-0000-0000-000000000004', @TENANT, @GRP_ADV, NULL, 'leads',         'delete', NOW(3)),
    ('e57e0001-0002-0000-0000-000000000005', @TENANT, @GRP_ADV, NULL, 'funnels',       'read',   NOW(3)),
    ('e57e0001-0002-0000-0000-000000000006', @TENANT, @GRP_ADV, NULL, 'funnels',       'update', NOW(3)),
    ('e57e0001-0002-0000-0000-000000000007', @TENANT, @GRP_ADV, NULL, 'contacts',      'create', NOW(3)),
    ('e57e0001-0002-0000-0000-000000000008', @TENANT, @GRP_ADV, NULL, 'contacts',      'read',   NOW(3)),
    ('e57e0001-0002-0000-0000-000000000009', @TENANT, @GRP_ADV, NULL, 'contacts',      'update', NOW(3)),
    ('e57e0001-0002-0000-0000-00000000000a', @TENANT, @GRP_ATD, NULL, 'contacts',      'read',   NOW(3)),
    ('e57e0001-0002-0000-0000-00000000000b', @TENANT, @GRP_ATD, NULL, 'contacts',      'update', NOW(3)),
    ('e57e0001-0002-0000-0000-00000000000c', @TENANT, @GRP_ATD, NULL, 'conversations', 'read',   NOW(3)),
    ('e57e0001-0002-0000-0000-00000000000d', @TENANT, @GRP_ATD, NULL, 'conversations', 'update', NOW(3)),
    ('e57e0001-0002-0000-0000-00000000000e', @TENANT, @GRP_ATD, NULL, 'leads',         'read',   NOW(3))
ON DUPLICATE KEY UPDATE action = VALUES(action);

-- =============================================================================
-- 2. PRODUTOS DO TENANT (reutiliza os 5 produtos previdenciários globais)
-- =============================================================================
INSERT INTO tenant_products (id, tenant_id, product_id, created_at) VALUES
    ('e57e0001-00a0-0000-0000-000000000001', @TENANT, @PROD_IDADE,   NOW()),
    ('e57e0001-00a0-0000-0000-000000000002', @TENANT, @PROD_TEMPO,   NOW()),
    ('e57e0001-00a0-0000-0000-000000000003', @TENANT, @PROD_BPC,     NOW()),
    ('e57e0001-00a0-0000-0000-000000000004', @TENANT, @PROD_AUXILIO, NOW()),
    ('e57e0001-00a0-0000-0000-000000000005', @TENANT, @PROD_REVISAO, NOW())
ON DUPLICATE KEY UPDATE product_id = VALUES(product_id);

-- =============================================================================
-- 3. FUNIL "PREVIDENCIÁRIO" — 8 COLUNAS
-- =============================================================================
SET @FUNNEL       = 'e57e0001-00b0-0000-0000-000000000001';
SET @COL_NOVO     = 'e57e0001-00b1-0000-0000-000000000001';
SET @COL_TRIAGEM  = 'e57e0001-00b1-0000-0000-000000000002';
SET @COL_ANALISE  = 'e57e0001-00b1-0000-0000-000000000003';
SET @COL_CALCULO  = 'e57e0001-00b1-0000-0000-000000000004';
SET @COL_PROTO    = 'e57e0001-00b1-0000-0000-000000000005';
SET @COL_AGUARDA  = 'e57e0001-00b1-0000-0000-000000000006';
SET @COL_GANHO    = 'e57e0001-00b1-0000-0000-000000000007';
SET @COL_PERDIDO  = 'e57e0001-00b1-0000-0000-000000000008';

INSERT INTO funnels (id, tenant_id, name, description, active, is_default, created_at, updated_at)
VALUES (@FUNNEL, @TENANT, 'Previdenciário', 'Funil de triagem e acompanhamento de casos previdenciários', TRUE, TRUE, NOW(), NOW())
ON DUPLICATE KEY UPDATE name = VALUES(name), description = VALUES(description);

INSERT INTO funnel_columns (id, funnel_id, name, order_index, type, color, created_at, updated_at) VALUES
    (@COL_NOVO,     @FUNNEL, 'Novo Contato',              0, 'entry',        '#3B82F6', NOW(), NOW()),
    (@COL_TRIAGEM,  @FUNNEL, 'Triagem Inicial',           1, 'intermediate', '#8B5CF6', NOW(), NOW()),
    (@COL_ANALISE,  @FUNNEL, 'Análise Documental',        2, 'intermediate', '#F59E0B', NOW(), NOW()),
    (@COL_CALCULO,  @FUNNEL, 'Cálculo de Benefício',      3, 'intermediate', '#06B6D4', NOW(), NOW()),
    (@COL_PROTO,    @FUNNEL, 'Protocolo no INSS',         4, 'intermediate', '#0EA5E9', NOW(), NOW()),
    (@COL_AGUARDA,  @FUNNEL, 'Aguardando Resposta INSS',  5, 'intermediate', '#F97316', NOW(), NOW()),
    (@COL_GANHO,    @FUNNEL, 'Ganho',                     6, 'won',          '#10B981', NOW(), NOW()),
    (@COL_PERDIDO,  @FUNNEL, 'Perdido',                   7, 'lost',         '#6B7280', NOW(), NOW())
ON DUPLICATE KEY UPDATE name = VALUES(name);

INSERT INTO funnel_products (id, funnel_id, product_id, priority, created_at) VALUES
    ('e57e0001-00b2-0000-0000-000000000001', @FUNNEL, @PROD_IDADE,   1, NOW()),
    ('e57e0001-00b2-0000-0000-000000000002', @FUNNEL, @PROD_TEMPO,   2, NOW()),
    ('e57e0001-00b2-0000-0000-000000000003', @FUNNEL, @PROD_BPC,     3, NOW()),
    ('e57e0001-00b2-0000-0000-000000000004', @FUNNEL, @PROD_AUXILIO, 4, NOW()),
    ('e57e0001-00b2-0000-0000-000000000005', @FUNNEL, @PROD_REVISAO, 5, NOW())
ON DUPLICATE KEY UPDATE priority = VALUES(priority);

-- =============================================================================
-- 4. ESPECIALISTAS (2): Triagem e Cálculo
-- =============================================================================
SET @SPEC_CLARA = '550e8400-e29b-41d4-a716-446655440050'; -- reusa ID da "IA Playground"
SET @SPEC_BRUNO = 'e57e0001-00c0-0000-0000-000000000001';

INSERT INTO specialists (id, name, description, prompt, status, created_at, updated_at) VALUES
    (@SPEC_CLARA, 'Dra. Helena — Triagem Previdenciária',
     'Assistente virtual para triagem inicial de potenciais clientes',
     'Você é a Dra. Helena, assistente virtual do Escritório Teste, especializado em Direito Previdenciário. Sua função é realizar a triagem inicial de potenciais clientes pelo WhatsApp. Seja acolhedora, profissional e use linguagem simples (sem juridiquês). Identifique qual benefício o cliente busca (Aposentadoria por Idade, Aposentadoria por Tempo de Contribuição, BPC/LOAS, Auxílio-Doença, Revisão de Benefício), colete informações básicas (nome, idade, tempo de contribuição, documentos disponíveis) e qualifique o lead. NUNCA dê parecer jurídico. NUNCA prometa resultado. Ao final da triagem, encaminhe para o advogado responsável.\n\n## Critérios de veto (desqualificação imediata)\n\nQuando um destes critérios for confirmado pelo cliente, encerre a triagem com uma mensagem empática e explique o motivo. Também sinalize no metadado STEP_META o campo `disqualified: true` para o sistema mover o lead para a coluna de Perdido:\n\n- **Auxílio-Doença**: cliente nunca contribuiu ao INSS (sem período de carência mínimo), OU afastamento inferior a 15 dias sem indício de condição incapacitante duradoura, OU não possui nem pretende obter laudo/atestado médico.\n- **Aposentadoria por Idade**: cliente com menos de 60 anos (mulher) ou 65 anos (homem), sem indício de que se aproxima do requisito nos próximos 12 meses.\n- **Aposentadoria por Tempo de Contribuição**: cliente declara menos de 15 anos de contribuição sem perspectiva de completar, ou nunca contribuiu.\n- **BPC/LOAS**: renda familiar per capita manifestamente acima de 1/4 do salário mínimo E cliente não é idoso (65+) nem tem deficiência.\n- **Revisão de Benefício**: cliente não recebe benefício atual do INSS, OU o benefício foi concedido há mais de 10 anos (decadência).\n\nAo aplicar um veto, seja breve e humana: explique em 2-3 frases por que não conseguimos seguir neste momento, agradeça o contato e oriente o próximo passo possível (ex: "volte quando tiver completado X meses de contribuição"). Não entre em detalhes jurídicos.',
     'active', NOW(), NOW()),
    (@SPEC_BRUNO, 'Dr. Bruno — Documentação e Cálculo',
     'Assistente especializado em orientação sobre documentos e cálculos de benefício',
     'Você é o Dr. Bruno, assistente virtual do Escritório Teste focado em documentação e cálculos previdenciários. Oriente o cliente sobre quais documentos apresentar (CNIS, CTPS, laudos médicos, comprovantes de contribuição) e explique em linguagem simples como é calculado o benefício que ele busca. Nunca dê valor exato nem prometa resultado — apenas explique a lógica geral. Se o cliente tiver dúvidas fora de documentação e cálculo, encaminhe de volta para a Dra. Helena.',
     'active', NOW(), NOW())
ON DUPLICATE KEY UPDATE name = VALUES(name), description = VALUES(description), prompt = VALUES(prompt);

-- Associação Specialist↔Tenant (Clara é default)
INSERT INTO specialist_tenants (specialist_id, tenant_id, is_default, created_at) VALUES
    (@SPEC_CLARA, @TENANT, 1, NOW()),
    (@SPEC_BRUNO, @TENANT, 0, NOW())
ON DUPLICATE KEY UPDATE is_default = VALUES(is_default);

-- Specialist↔Produtos
INSERT INTO specialist_products (id, specialist_id, product_id, created_at) VALUES
    -- Clara cobre todos (triagem)
    ('e57e0001-00c1-0001-0000-000000000001', @SPEC_CLARA, @PROD_IDADE,   NOW(3)),
    ('e57e0001-00c1-0001-0000-000000000002', @SPEC_CLARA, @PROD_TEMPO,   NOW(3)),
    ('e57e0001-00c1-0001-0000-000000000003', @SPEC_CLARA, @PROD_BPC,     NOW(3)),
    ('e57e0001-00c1-0001-0000-000000000004', @SPEC_CLARA, @PROD_AUXILIO, NOW(3)),
    ('e57e0001-00c1-0001-0000-000000000005', @SPEC_CLARA, @PROD_REVISAO, NOW(3)),
    -- Bruno foca em aposentadorias e revisão (cálculos)
    ('e57e0001-00c1-0002-0000-000000000001', @SPEC_BRUNO, @PROD_IDADE,   NOW(3)),
    ('e57e0001-00c1-0002-0000-000000000002', @SPEC_BRUNO, @PROD_TEMPO,   NOW(3)),
    ('e57e0001-00c1-0002-0000-000000000003', @SPEC_BRUNO, @PROD_REVISAO, NOW(3))
ON DUPLICATE KEY UPDATE created_at = VALUES(created_at);

-- =============================================================================
-- 5. AI CONFIGS — ambos usam OpenAI real
-- =============================================================================
INSERT INTO ai_configs (id, specialist_id, provider, model, temperature, max_tokens, debounce_seconds, created_at, updated_at) VALUES
    ('e57e0001-00d0-0000-0000-000000000001', @SPEC_CLARA, 'openai', 'gpt-5.4-mini', 0.70, 1024, 3, NOW(), NOW()),
    ('e57e0001-00d0-0000-0000-000000000002', @SPEC_BRUNO, 'openai', 'gpt-5.4-nano', 0.60, 1024, 3, NOW(), NOW())
ON DUPLICATE KEY UPDATE
    provider = VALUES(provider), model = VALUES(model),
    temperature = VALUES(temperature), max_tokens = VALUES(max_tokens),
    debounce_seconds = VALUES(debounce_seconds), updated_at = NOW();

-- =============================================================================
-- 6. STEPS DE TREINAMENTO
-- =============================================================================
-- Clara: triagem em 6 passos
INSERT INTO steps (id, specialist_id, order_index, text, data_type, required, score, target_column_id, created_at, updated_at) VALUES
    ('e57e0001-00e1-0001-0000-000000000001', @SPEC_CLARA, 0, 'Qual seu nome completo?',                                               'free_text', TRUE,  10, @COL_TRIAGEM,  NOW(), NOW()),
    ('e57e0001-00e1-0001-0000-000000000002', @SPEC_CLARA, 1, 'Qual benefício você busca? (Aposentadoria por Idade, Tempo, BPC/LOAS, Auxílio-Doença, Revisão)', 'selection', TRUE,  20, NULL, NOW(), NOW()),
    ('e57e0001-00e1-0001-0000-000000000003', @SPEC_CLARA, 2, 'Qual sua idade?',                                                       'number',    TRUE,  15, NULL,          NOW(), NOW()),
    ('e57e0001-00e1-0001-0000-000000000004', @SPEC_CLARA, 3, 'Quantos anos de contribuição ao INSS você tem (aproximadamente)?',     'number',    FALSE, 15, NULL,          NOW(), NOW()),
    ('e57e0001-00e1-0001-0000-000000000005', @SPEC_CLARA, 4, 'Você já tem documentos como CNIS, carteira de trabalho ou laudos? (Todos / Alguns / Nenhum)', 'selection', TRUE,  20, NULL, NOW(), NOW()),
    ('e57e0001-00e1-0001-0000-000000000006', @SPEC_CLARA, 5, 'Descreva brevemente sua situação para entendermos melhor seu caso.',   'free_text', FALSE, 20, NULL,          NOW(), NOW())
ON DUPLICATE KEY UPDATE text = VALUES(text);

-- Bruno: documentação e cálculo em 4 passos
INSERT INTO steps (id, specialist_id, order_index, text, data_type, required, score, target_column_id, created_at, updated_at) VALUES
    ('e57e0001-00e1-0002-0000-000000000001', @SPEC_BRUNO, 0, 'Qual benefício você quer calcular/protocolar?',                        'selection', TRUE,  20, NULL, NOW(), NOW()),
    ('e57e0001-00e1-0002-0000-000000000002', @SPEC_BRUNO, 1, 'Você tem CNIS atualizado? Pode enviar uma cópia?',                     'selection', TRUE,  30, NULL, NOW(), NOW()),
    ('e57e0001-00e1-0002-0000-000000000003', @SPEC_BRUNO, 2, 'Tem cópia da(s) carteira(s) de trabalho?',                              'selection', TRUE,  25, NULL, NOW(), NOW()),
    ('e57e0001-00e1-0002-0000-000000000004', @SPEC_BRUNO, 3, 'Qual seu último salário de contribuição (aproximado)?',                'number',    FALSE, 25, NULL, NOW(), NOW())
ON DUPLICATE KEY UPDATE text = VALUES(text);

-- =============================================================================
-- 7. SCORING CONFIG
-- =============================================================================
INSERT INTO scoring_configs (id, specialist_id, threshold, qualified_column_id, disqualified_column_id, created_at, updated_at) VALUES
    ('e57e0001-00f0-0001-0000-000000000001', @SPEC_CLARA, 60, @COL_ANALISE, @COL_PERDIDO, NOW(), NOW()),
    ('e57e0001-00f0-0002-0000-000000000001', @SPEC_BRUNO, 50, @COL_CALCULO, @COL_PERDIDO, NOW(), NOW())
ON DUPLICATE KEY UPDATE threshold = VALUES(threshold);

-- =============================================================================
-- 8. GUARDRAILS
-- =============================================================================
INSERT INTO guardrails (id, specialist_id, type, rule, message, active, created_at, updated_at) VALUES
    ('e57e0001-0110-0001-0000-000000000001', @SPEC_CLARA, 'forbidden_topics',
     'Não dar parecer jurídico nem prometer resultado de processos',
     'Desculpe, não posso dar parecer jurídico. Um advogado do escritório vai analisar seu caso e entrar em contato.',
     TRUE, NOW(), NOW()),
    ('e57e0001-0110-0001-0000-000000000002', @SPEC_CLARA, 'scope_limit',
     'Apenas assuntos de direito previdenciário e INSS',
     'Posso ajudar apenas com benefícios previdenciários e INSS. Para outros assuntos, recomendo procurar um especialista na área.',
     TRUE, NOW(), NOW()),
    ('e57e0001-0110-0001-0000-000000000003', @SPEC_CLARA, 'response_tone',
     'Tom acolhedor e profissional, sem juridiquês. Linguagem simples e acessível.',
     NULL, TRUE, NOW(), NOW()),
    ('e57e0001-0110-0002-0000-000000000001', @SPEC_BRUNO, 'forbidden_topics',
     'Não dar valor exato de benefício nem prometer resultado',
     'Não consigo calcular o valor exato — apenas explicar a lógica. O advogado fará o cálculo preciso com seus documentos.',
     TRUE, NOW(), NOW()),
    ('e57e0001-0110-0002-0000-000000000002', @SPEC_BRUNO, 'scope_limit',
     'Apenas documentação e cálculo previdenciário',
     'Esse assunto foge da minha especialidade. Vou te transferir para a Dra. Helena, que cuida da triagem.',
     TRUE, NOW(), NOW()),
    ('e57e0001-0110-0002-0000-000000000003', @SPEC_BRUNO, 'response_tone',
     'Didático, passo a passo, sem juridiquês',
     NULL, TRUE, NOW(), NOW())
ON DUPLICATE KEY UPDATE rule = VALUES(rule);

-- =============================================================================
-- 9. CONTATOS E CONVERSAS (playground + 3 leads em estágios)
-- =============================================================================
SET @CT_TESTE  = 'e57e0001-0120-0000-0000-000000000001';
SET @CT_MARIA  = 'e57e0001-0120-0000-0000-000000000002';
SET @CT_JOAO   = 'e57e0001-0120-0000-0000-000000000003';
SET @CT_LUCIA  = 'e57e0001-0120-0000-0000-000000000004';

INSERT INTO contacts (id, tenant_id, name, phone, whatsapp_id, created_at, updated_at) VALUES
    (@CT_TESTE, @TENANT, 'Teste Playground ET',  '5511988880000', '5511988880000@s.whatsapp.net', NOW(), NOW()),
    (@CT_MARIA, @TENANT, 'Maria Aparecida',      '5511988880001', '5511988880001@s.whatsapp.net', NOW(), NOW()),
    (@CT_JOAO,  @TENANT, 'João Carvalho',        '5511988880002', '5511988880002@s.whatsapp.net', NOW(), NOW()),
    (@CT_LUCIA, @TENANT, 'Lúcia Ferreira',       '5511988880003', '5511988880003@s.whatsapp.net', NOW(), NOW())
ON DUPLICATE KEY UPDATE name = VALUES(name);

SET @CV_TESTE = 'e57e0001-0130-0000-0000-000000000001';
SET @CV_MARIA = 'e57e0001-0130-0000-0000-000000000002';
SET @CV_JOAO  = 'e57e0001-0130-0000-0000-000000000003';
SET @CV_LUCIA = 'e57e0001-0130-0000-0000-000000000004';

INSERT INTO conversations (id, tenant_id, contact_id, status, last_message_at, unread_count, created_at, updated_at) VALUES
    (@CV_TESTE, @TENANT, @CT_TESTE, 'open', NOW(), 0, NOW(), NOW()),
    (@CV_MARIA, @TENANT, @CT_MARIA, 'open', NOW(), 0, NOW(), NOW()),
    (@CV_JOAO,  @TENANT, @CT_JOAO,  'open', NOW(), 0, NOW(), NOW()),
    (@CV_LUCIA, @TENANT, @CT_LUCIA, 'open', NOW(), 0, NOW(), NOW())
ON DUPLICATE KEY UPDATE status = VALUES(status);

-- =============================================================================
-- 10. LEADS DE EXEMPLO EM ESTÁGIOS DIFERENTES
-- =============================================================================
INSERT INTO leads (id, tenant_id, funnel_id, column_id, contact_id, conversation_id, product_id, responsible_user_id, score, status, column_entered_at, created_at, updated_at) VALUES
    ('e57e0001-0140-0000-0000-000000000001', @TENANT, @FUNNEL, @COL_NOVO,    @CT_MARIA, @CV_MARIA, @PROD_IDADE,   @JULIANA, 15, 'open', NOW(),                             NOW(),                              NOW()),
    ('e57e0001-0140-0000-0000-000000000002', @TENANT, @FUNNEL, @COL_ANALISE, @CT_JOAO,  @CV_JOAO,  @PROD_TEMPO,   @ANA,     65, 'open', DATE_SUB(NOW(), INTERVAL 2 DAY),    DATE_SUB(NOW(), INTERVAL 2 DAY),    NOW()),
    ('e57e0001-0140-0000-0000-000000000003', @TENANT, @FUNNEL, @COL_PROTO,   @CT_LUCIA, @CV_LUCIA, @PROD_AUXILIO, @RICARDO, 80, 'open', DATE_SUB(NOW(), INTERVAL 6 DAY),    DATE_SUB(NOW(), INTERVAL 10 DAY),   NOW())
ON DUPLICATE KEY UPDATE score = VALUES(score);
