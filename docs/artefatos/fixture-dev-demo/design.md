# Fixture de Dev/Demo — Escritório Previdenciário

**Data**: 2026-04-07
**Objetivo**: Popular o banco de dados local com dados realistas de um escritório de advocacia previdenciária, para desenvolvimento e demonstração da plataforma.

## Decisões de Design

- **Arquivo único**: `fixture/fixtures.sql` (substituir conteúdo atual)
- **Idempotente**: usar `ON DUPLICATE KEY UPDATE` em todos os INSERTs
- **UUIDs fixos**: para referências cruzadas previsíveis entre tabelas
- **Português BR**: nomes, mensagens e conteúdos 100% em português brasileiro
- **Admin preservado**: o admin existente (`admin@teste.com`) permanece sem alteração
- **Volume enxuto**: ~55 registros no total, suficiente para demo sem poluir

## Cenário

Escritório **"Mendes & Costa Advocacia Previdenciária"** — um tenant ativo que utiliza todas as funcionalidades da plataforma: funil de leads, WhatsApp, especialista IA, automações e permissões.

---

## Seção 1 — Tenant e Usuários

### Tenant

| Campo | Valor |
|-------|-------|
| Nome | Mendes & Costa Advocacia Previdenciária |
| Tipo | PJ |
| Documento | 12.345.678/0001-90 |
| Status | active |

### Usuários (3 novos)

| Nome | Email | Role | is_owner |
|------|-------|------|----------|
| Dr. Ricardo Mendes | ricardo@mendescosta.adv.br | user | true |
| Dra. Ana Costa | ana@mendescosta.adv.br | user | false |
| Juliana Rocha | juliana@mendescosta.adv.br | user | false |

### Grupos de Permissão

| Grupo | Membros | Acesso |
|-------|---------|--------|
| Advogados | Ricardo, Ana | Acesso total ao funil/leads |
| Atendimento | Juliana | Acesso a contatos e conversas, sem editar funil |

---

## Seção 2 — Produtos e Especialista IA

### 5 Produtos

| Produto | Keywords |
|---------|----------|
| Aposentadoria por Idade | `["aposentadoria", "idade", "65 anos", "60 anos"]` |
| Aposentadoria por Tempo de Contribuição | `["tempo de contribuição", "aposentadoria", "35 anos", "30 anos"]` |
| BPC/LOAS | `["bpc", "loas", "benefício assistencial", "deficiência", "idoso"]` |
| Auxílio-Doença | `["auxílio-doença", "incapacidade", "afastamento", "laudo médico"]` |
| Revisão de Benefício | `["revisão", "recálculo", "valor errado", "teto"]` |

### Especialista: "Dra. Clara"

- **Prompt**: Triagem previdenciária, tom acolhedor, sem parecer jurídico
- **Status**: active
- **Vinculação**: is_default = true para o tenant

### 6 Steps de Treinamento

| Ordem | Texto | Tipo | Obrigatório | Score |
|-------|-------|------|-------------|-------|
| 1 | Qual seu nome completo? | free_text | sim | 10 |
| 2 | Qual tipo de benefício busca? | selection | sim | 20 |
| 3 | Qual sua idade? | number | sim | 15 |
| 4 | Quantos anos de contribuição ao INSS? | number | não | 15 |
| 5 | Possui documentos (CNIS, laudos)? | selection | sim | 20 |
| 6 | Descreva brevemente sua situação | free_text | não | 20 |

### Scoring Config

- **Threshold**: 60
- **Qualified** → coluna "Análise Documental"
- **Disqualified** → coluna "Perdido"

### 3 Guardrails

| Tipo | Regra | Mensagem |
|------|-------|----------|
| forbidden_topics | Não forneça parecer jurídico nem prometa resultado | Desculpe, não posso dar parecer jurídico. Um advogado entrará em contato. |
| scope_limit | Apenas assuntos de direito previdenciário e INSS | Posso ajudar apenas com questões previdenciárias. |
| response_tone | Tom acolhedor e profissional, sem juridiquês | (orientação interna) |

---

## Seção 3 — Funil, Colunas e Leads

### Funil: "Previdenciário"

`active = true`, `is_default = true`

### 8 Colunas

| Ordem | Nome | Tipo | Cor |
|-------|------|------|-----|
| 0 | Novo Contato | entry | #3B82F6 (azul) |
| 1 | Análise Documental | intermediate | #F59E0B (amarelo) |
| 2 | Cálculo de Benefício | intermediate | #8B5CF6 (roxo) |
| 3 | Protocolo no INSS | intermediate | #06B6D4 (ciano) |
| 4 | Aguardando Resposta INSS | intermediate | #F97316 (laranja) |
| 5 | Recurso/Revisão | intermediate | #EF4444 (vermelho) |
| 6 | Ganho | won | #10B981 (verde) |
| 7 | Perdido | lost | #6B7280 (cinza) |

### 8 Leads (1 por coluna)

| Lead | Contato | Produto | Coluna | Responsável | Score |
|------|---------|---------|--------|-------------|-------|
| Maria da Silva, 63 anos | 5511999001001 | Aposentadoria por Idade | Novo Contato | Juliana | 10 |
| José Santos, 58 anos | 5511999002002 | Aposentadoria por Tempo | Análise Documental | Ana | 60 |
| Dona Francisca, 67 anos | 5511999003003 | BPC/LOAS | Cálculo de Benefício | Ana | 80 |
| Carlos Oliveira, 45 anos | 5511999004004 | Auxílio-Doença | Protocolo no INSS | Ricardo | 75 |
| Pedro Souza, 70 anos | 5511999005005 | Revisão de Benefício | Aguardando Resposta INSS | Ricardo | 70 |
| Ana Paula Lima, 55 anos | 5511999006006 | Aposentadoria por Tempo | Recurso/Revisão | Ana | 85 |
| Seu João Ferreira, 66 anos | 5511999007007 | Aposentadoria por Idade | Ganho | Ricardo | 90 |
| Marcos Pereira, 40 anos | 5511999008008 | Auxílio-Doença | Perdido | Juliana | 25 |

### 3 Lead Notes

| Lead | Conteúdo | Criado por |
|------|----------|------------|
| José Santos | CNIS apresentado, faltam últimas contribuições | Ana |
| Dona Francisca | Renda familiar comprovada, laudo médico atualizado | Ana |
| Seu João Ferreira | Benefício concedido — aposentadoria por idade deferida | Ricardo |

### 3 Lead Movements (histórico)

| Lead | Caminho |
|------|---------|
| José Santos | Novo Contato → Análise Documental |
| Carlos Oliveira | Novo Contato → Análise Documental → Protocolo no INSS |
| Seu João Ferreira | Novo Contato → Análise Documental → Cálculo → Protocolo → Ganho |

---

## Seção 4 — WhatsApp

### 8 Contatos

| Nome | Telefone | WhatsApp ID |
|------|----------|-------------|
| Maria da Silva | 5511999001001 | 5511999001001@s.whatsapp.net |
| José Santos | 5511999002002 | 5511999002002@s.whatsapp.net |
| Dona Francisca | 5511999003003 | 5511999003003@s.whatsapp.net |
| Carlos Oliveira | 5511999004004 | 5511999004004@s.whatsapp.net |
| Pedro Souza | 5511999005005 | 5511999005005@s.whatsapp.net |
| Ana Paula Lima | 5511999006006 | 5511999006006@s.whatsapp.net |
| Seu João Ferreira | 5511999007007 | 5511999007007@s.whatsapp.net |
| Marcos Pereira | 5511999008008 | 5511999008008@s.whatsapp.net |

### 8 Conversas

- 6 com status `open` (leads ativos no funil)
- 2 com status `closed` (Ganho: Seu João / Perdido: Marcos)

### ~20 Mensagens (3 conversas detalhadas)

**Maria da Silva** (triagem em andamento — 5 msgs):
1. ← "Boa tarde, vi o anúncio de vocês sobre aposentadoria"
2. → "Olá Maria! Sou a Dra. Clara, assistente virtual. Qual seu nome completo?"
3. ← "Maria da Silva"
4. → "Obrigada, Maria! Qual tipo de benefício você busca? Aposentadoria, BPC/LOAS, auxílio-doença ou revisão?"
5. ← "Aposentadoria, tenho 63 anos"

**José Santos** (documentos pendentes — 7 msgs):
1. ← "Bom dia, quero ver sobre minha aposentadoria por tempo de contribuição"
2. → "Olá José! Quantos anos de contribuição ao INSS você possui?"
3. ← "Tenho 33 anos de carteira assinada"
4. → "Ótimo! Você possui o CNIS atualizado e carteiras de trabalho?"
5. ← "Tenho o CNIS sim, vou enviar"
6. → "Perfeito! Encaminhei seu caso para a Dra. Ana que vai analisar sua documentação"
7. ← "Obrigado, aguardo retorno"

**Carlos Oliveira** (caso em andamento — 8 msgs):
1. ← "Oi, estou afastado do trabalho e preciso de auxílio-doença"
2. → "Olá Carlos! Lamento pela situação. Você possui laudo médico atualizado?"
3. ← "Sim, tenho laudo do ortopedista"
4. → "Ótimo. Vou encaminhar para o Dr. Ricardo. Ele vai preparar seu pedido ao INSS"
5. ← "Quanto tempo demora?"
6. → "O protocolo no INSS leva em média 30 a 45 dias para análise. O Dr. Ricardo vai acompanhar de perto"
7. ← "Tá bom, obrigado"
8. → "Seu pedido foi protocolado no INSS. Acompanhe pelo número 35.123.456-7"

Demais 5 conversas: sem mensagens detalhadas (interação principal foi presencial/telefone).

---

## Seção 5 — Automações e Notificações

### 4 Automações

| Tipo | Coluna alvo | Config | Prioridade |
|------|-------------|--------|------------|
| expiration | Aguardando Resposta INSS | `{"days": 30, "action": "notify"}` | 1 |
| auto_message | Protocolo no INSS | `{"message": "Seu pedido foi protocolado no INSS. Acompanharemos o andamento e retornaremos assim que houver resposta."}` | 2 |
| auto_note | Ganho | `{"content": "Benefício concedido — caso encerrado com sucesso"}` | 3 |
| detect_product | Novo Contato | `{"enabled": true}` | 4 |

### 2 Execution Logs

| Automação | Lead | Status |
|-----------|------|--------|
| auto_message | Carlos Oliveira | success |
| auto_note | Seu João Ferreira | success |

### 3 Notificações

| Usuário | Título | Read |
|---------|--------|------|
| Ricardo | Lead Carlos Oliveira protocolado no INSS | true |
| Ricardo | Lead Pedro Souza aguardando resposta INSS há 25 dias | false |
| Ana | Novo lead atribuído: José Santos | false |

### Notification Preferences

Todos os 3 usuários com todos os tipos de notificação habilitados.

---

## Resumo de Volume

| Entidade | Qty |
|----------|-----|
| Tenant | 1 |
| Usuários | 3 |
| User-tenant associations | 3 |
| Grupos de permissão | 2 |
| User-group associations | 3 |
| Permissões | ~6 |
| Produtos | 5 |
| Tenant-products | 5 |
| Funnel-products | 5 |
| Especialista | 1 |
| Specialist-tenant | 1 |
| Steps | 6 |
| Scoring config | 1 |
| Guardrails | 3 |
| Funil | 1 |
| Colunas | 8 |
| Leads | 8 |
| Lead notes | 3 |
| Lead movements | ~10 |
| Contatos | 8 |
| Conversas | 8 |
| Mensagens | 20 |
| Automações | 4 |
| Execution logs | 2 |
| Notificações | 3 |
| Notification preferences | ~9 |
| **Total** | **~55** |
