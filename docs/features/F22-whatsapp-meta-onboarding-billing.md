# F22 - WhatsApp Meta — Onboarding e Billing Avançado

## Objetivo
Reduzir fricção de onboarding e ganhar visibilidade financeira sobre o uso do WhatsApp Cloud API. Habilita escala (dezenas/centenas de tenants) sem suporte manual a cada novo cliente.

## Pré-requisitos
- F20 (provider Meta funcional em produção)
- F11 (pagamentos — para integração opcional de billing repassado)

## Status: backlog

## Steps

### Step 1: Aprovação como Tech Provider Meta
- [ ] cadastrar app no Meta Business Manager
- [ ] solicitar permissões necessárias (`whatsapp_business_management`, `whatsapp_business_messaging`, `business_management`)
- [ ] passar revisão de App Review da Meta
- [ ] documentar processo em `docs/produto/meta-tech-provider.md`

> **Nota**: este step não envolve código — é processo administrativo/comercial com a Meta. Pode levar semanas. Bloqueia steps 2 e 3.

### Step 2: Embedded Signup
- [ ] integrar SDK JavaScript da Meta na UI de onboarding do tenant
- [ ] fluxo OAuth-like: tenant clica "Conectar WhatsApp" → popup Meta → autoriza → CRM recebe tokens automaticamente
- [ ] callback que persiste credenciais (reusando entidade `MetaCredentials` da F20)
- [ ] tela de seleção de phone number (lista os números da Business Account conectada)
- [ ] tratamento de erros de autorização (cancelamento, escopo insuficiente, conta inválida)
- [ ] testes E2E com conta Meta de homologação

### Step 3: Gestão de Business Account via API
- [ ] caso de uso: listar phone numbers da BA conectada
- [ ] caso de uso: solicitar verificação de número (OTP via SMS/voz)
- [ ] caso de uso: criar template HSM direto pelo CRM (sem ir ao Business Manager)
- [ ] webhook de callback de aprovação/rejeição de template
- [ ] UI: tela de gestão de templates (criar, listar, ver status, deletar)
- [ ] testes

### Step 4: Tracking de custo Meta
- [ ] entidade `WhatsAppConversationCost` (id, tenant_id, conversation_id, categoria, pais, custo_unitario, moeda, mensagem_id, created_at)
- [ ] categorias Meta: `utility`, `marketing`, `authentication`, `service`
- [ ] webhook adicional para eventos de billing (`pricing` no payload de status)
- [ ] tabela de preços por país/categoria (atualizada periodicamente da documentação Meta)
- [ ] testes

### Step 5: Dashboard de custos por tenant
- [ ] dashboard admin: custo total por tenant, por mês, por categoria
- [ ] dashboard tenant: próprio consumo no mês
- [ ] alerta de uso anômalo (custo do dia > N x média móvel 7 dias)
- [ ] integração com F19 (dashboards)

### Step 6: Repasse de custo (opcional, integrado ao F11)
- [ ] flag por plano: incluir custo Meta na fatura ou cobrar à parte
- [ ] geração de linha de fatura mensal com consumo Meta
- [ ] testes

### Step 7: Migração assistida whatsmeow → Meta
- [ ] caso de uso: migrar tenant de whatsmeow para Meta sem perder conversas/contatos
- [ ] estratégia: provider em modo "shadow" por janela de N dias, comparando entrega
- [ ] mapeamento de `whatsapp_msg_id` antigo↔novo (necessário se Meta atribuir IDs diferentes em conversas espelhadas)
- [ ] guia operacional em `docs/produto/migracao-whatsmeow-meta.md`
- [ ] testes

### Step 8: Documentação
- [ ] guia de onboarding via Embedded Signup (substitui guia manual da F20)
- [ ] FAQ de billing Meta (categorias, preços, janela 24h)
- [ ] entrada no changelog
- [ ] atualizar backlog

## Critérios de aceite
- tenant consegue conectar WhatsApp em < 5 minutos via Embedded Signup
- templates HSM podem ser criados, monitorados e deletados pelo CRM
- custo Meta é rastreado por mensagem, agregado por tenant/mês/categoria
- dashboards de custo (admin e tenant) operacionais
- migração whatsmeow → Meta documentada e testada
- cobertura >= 80%

## Decisões técnicas

### Por que feature separada da F20
- Embedded Signup depende de **aprovação Meta como Tech Provider** (processo externo, semanas). Bloquear F20 nisso atrasa produção.
- Billing avançado e gestão de templates só fazem sentido com **volume** (dezenas+ de tenants). Antes disso, é overengineering.
- Migração whatsmeow → Meta provavelmente é não-problema no MVP — tenants novos já entram direto no Meta após F20.

### Embedded Signup vs OAuth tradicional
- Embedded Signup é o fluxo oficial Meta para onboarding de WhatsApp Business. Não é OAuth genérico — é um SDK específico que abre popup com fluxo guiado.
- Sem isso, tenant precisa: criar BA → verificar negócio → comprar/conectar número → gerar System User → copiar tokens. ≈ 1–2h por tenant.

### Categorias de billing Meta
- Meta cobra por **conversa** (janela de 24h), não por mensagem. Categoria definida pela primeira mensagem do business na conversa.
- Preços variam por país (ex.: Brasil utility ~$0.008, marketing ~$0.0625 por conversa em 2026). Tabela precisa atualização periódica.

### Migração shadow-mode
- Rodar os dois providers em paralelo durante transição evita perda de mensagens. Tenant decide quando fazer cutover definitivo.
- Custo: dobra processamento durante a janela. Aceitável para feature de migração one-shot.
