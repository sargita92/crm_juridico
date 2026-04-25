# F20 - WhatsApp Business API (Meta) — Provider de Produção

## Objetivo
Implementar o provider WhatsApp Cloud API (Meta) na interface `WhatsAppProvider` já definida em F06, viabilizando uso em produção. whatsmeow permanece como provider de desenvolvimento (sem custo, login via QR).

## Pré-requisitos
- F06 (integração WhatsApp via whatsmeow — define a interface `WhatsAppProvider`)
- F02 (multitenancy — credenciais Meta são por tenant)

## Status: backlog

## Steps

### Step 1: Domínio e credenciais Meta por tenant
- [ ] entidade `MetaCredentials` (id, tenant_id, phone_number_id, business_account_id, access_token criptografado, app_secret criptografado, verify_token, ativo, created_at, updated_at)
- [ ] migration com índice único em (tenant_id) e (phone_number_id)
- [ ] criptografia de access_token e app_secret em repouso (chave via env, AES-GCM)
- [ ] testes unitários do domínio

### Step 2: Provider Meta Cloud API
- [ ] implementação `MetaProvider` da interface `WhatsAppProvider`
- [ ] envio de mensagem de texto via `POST /v20.0/{phone_number_id}/messages`
- [ ] tratamento de erros da API (rate limit, token expirado, número inválido)
- [ ] retry com backoff exponencial em erros transitórios
- [ ] testes unitários com mocks HTTP

### Step 3: Seleção de provider por env e por tenant
- [ ] env `WHATSAPP_PROVIDER_DEFAULT` (whatsmeow | meta)
- [ ] override por tenant (se tenant tem `MetaCredentials` ativo, usa Meta; senão, usa default)
- [ ] factory que resolve o provider correto no boot e por requisição
- [ ] testes

### Step 4: Webhook de recebimento
- [ ] endpoint público `GET /webhooks/whatsapp/meta` (verificação inicial com `hub.verify_token`)
- [ ] endpoint público `POST /webhooks/whatsapp/meta` (recebe eventos)
- [ ] validação de assinatura `X-Hub-Signature-256` (HMAC SHA-256 com app_secret)
- [ ] roteamento por `phone_number_id` para o tenant correto
- [ ] parser de payloads: mensagem recebida, status (sent/delivered/read/failed), erros
- [ ] deduplicação via `whatsapp_msg_id` (mesma constraint do F06)
- [ ] testes de integração com payloads reais da Meta (fixtures)

### Step 5: Templates HSM (mensagens fora da janela 24h)
- [ ] caso de uso: listar templates aprovados do tenant (cache local com refresh por TTL)
- [ ] caso de uso: enviar mensagem template (com parâmetros)
- [ ] domínio: entidade `MessageTemplate` (nome, idioma, categoria, status, parâmetros esperados)
- [ ] UI: dropdown de templates ao tentar enviar fora da janela 24h
- [ ] testes

### Step 6: Mídia (imagem, áudio, documento)
- [ ] envio: upload prévio via `POST /v20.0/{phone_number_id}/media` → recebe `media_id` → envia mensagem referenciando
- [ ] recebimento: ao receber `media_id` no webhook, GET autenticado em `/v20.0/{media_id}` para baixar
- [ ] integração com F14 (arquivos por lead) para persistir mídia recebida/enviada
- [ ] testes

### Step 7: UI de configuração no painel admin do tenant
- [ ] tela de configuração WhatsApp Meta (cadastrar phone_number_id, business_account_id, access_token, app_secret)
- [ ] teste de conexão (chama endpoint `/v20.0/{phone_number_id}` para validar credenciais)
- [ ] indicador visual de status (ativo, erro, não configurado)
- [ ] documentação inline com link para guia de setup Meta Business

### Step 8: Observabilidade
- [ ] métricas: mensagens enviadas/recebidas por provider, latência de envio Meta, erros 4xx/5xx por tipo, taxa de templates HSM
- [ ] logs estruturados com `tenant_id`, `phone_number_id`, `provider=meta`
- [ ] traces end-to-end (webhook → handler → repositório)
- [ ] alerta para falhas de assinatura HMAC (possível tentativa de injeção)

### Step 9: Testes OWASP do webhook
- [ ] requisição sem assinatura → 401
- [ ] requisição com assinatura inválida → 401
- [ ] requisição com `phone_number_id` desconhecido → 404 (não vazar lista de tenants)
- [ ] payload malformado → 400 sem stack trace
- [ ] rate limit por IP no endpoint público

### Step 10: Documentação e arquivos .http
- [ ] `rest/whatsapp-meta.http` com exemplos de webhook, envio de texto, envio de template, envio de mídia
- [ ] guia de onboarding manual em `docs/produto/onboarding-whatsapp-meta.md` (passo a passo do Meta Business Manager)
- [ ] entrada no changelog
- [ ] atualizar backlog

## Critérios de aceite
- tenant consegue cadastrar credenciais Meta e enviar/receber mensagens via Cloud API
- whatsmeow continua funcionando para tenants não migrados (compatibilidade)
- webhooks validam assinatura corretamente
- templates HSM permitem reabrir conversa fora da janela 24h
- mídia (imagem, áudio, documento) flui em ambas direções
- métricas, logs e traces cobrem o novo provider
- testes OWASP passam
- cobertura >= 80%

## Decisões técnicas

### Por que escopo "produção mínima viável"
- Embedded Signup, gestão de Business Account via API e billing Meta integrado ficam em **F22** (feature separada). Razão: Embedded Signup exige aprovação Meta como Tech Provider (semanas), e billing/gestão avançada só agregam valor com volume de tenants.

### Onboarding manual nesta feature
- Cliente cria Meta Business Account, verifica número, gera System User token e cola no CRM. Aceitável para os primeiros 5–20 tenants. Automação fica em F22.

### Compatibilidade com whatsmeow
- A interface `WhatsAppProvider` (F06) abstrai a troca. Nenhum código de domínio ou aplicação muda — só infraestrutura.
- whatsmeow permanece em `internal/whatsapp/infrastructure/whatsmeow_provider.go`. Meta entra em `meta_provider.go` no mesmo pacote.

### Criptografia de credenciais
- access_token e app_secret são segredos de longo prazo. Persistir em texto puro viola compliance básico. AES-GCM com chave em env (`META_CREDENTIALS_KEY`).

### Roteamento de webhook multitenant
- Meta envia todos os eventos para a **mesma URL** do app. O `phone_number_id` no payload identifica o tenant. Sem roteamento correto, mensagens vazariam entre tenants.
