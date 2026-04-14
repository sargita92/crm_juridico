# F17 — Fluxo de teste manual e comando /reset

## Objetivo

Permitir validação ponta-a-ponta do fluxo "lead manda mensagem → especialista IA atende → percorre os steps do produto → lead qualificado" sem depender de WhatsApp real conectado. Entrega três capacidades:

1. **Playground dev** em `/tenant/ai/playground` — injeta mensagens no pipeline inbound real como se viessem do lead.
2. **Comando `/reset`** — reinicia a conversa do zero (state, score e lead).
3. **Keywords com frases em aspas** — permite cadastrar frases wa.me completas (com vírgula) como keyword de produto.

Além disso, fecha o gap de fixture seedando `ai_configs` pro especialista previdenciário e estabelece teste de integração end-to-end com `FakeProvider`.

## Pré-requisitos

- F06 (WhatsApp recebendo/enviando — concluído)
- F07 (funis/kanban — concluído)
- F10 (produtos e detecção por keyword — concluído)
- F16 (motor de IA dos especialistas — concluído)

## Status: concluído

## Steps

### Step 1: Infraestrutura de config e métricas

- [x] Flags `AI_PLAYGROUND_ENABLED` e `AI_RESET_COMMAND_ENABLED` em `config.Config` com defaults seguros (playground off, reset on)
- [x] Explicit `viper.BindEnv` para garantir mapeamento env → viper
- [x] Warning no startup se `Env=production && PlaygroundEnabled`
- [x] Métricas `ai_reset_commands_total{tenant_id,specialist_id,source}` e `ai_playground_messages_total{tenant_id}`

### Step 2: Comando /reset

- [x] Método `ConversationState.Reset()` (zera step, score, handoff, collected_data)
- [x] Função pura `IsResetCommand([]string) bool` (case-sensitive, trimmed)
- [x] `ResetConversationUseCase` com interfaces `EntryColumnFinder` + `LeadResetter`
- [x] Adapters `FunnelEntryAdapter` + `LeadResetterAdapter` em `infrastructure/`
- [x] Intercepção no início de `ConversationEngine.HandleMessages` (gated por flag)
- [x] Fallback do lead pra entry column do funil default + score zerado
- [x] Mensagem de confirmação "Conversa reiniciada. Pode começar de novo quando quiser."
- [x] Testes unit cobrindo: state existente, state ausente, lead ausente, flag off

### Step 3: FakeProvider determinístico

- [x] `FakeProvider` em `infrastructure/fake_provider.go` implementando `domain.AIProvider`
- [x] Registrado no `ProviderRegistry` sempre (alongside OpenAI)
- [x] Usado pela fixture em dev/test via `ai_configs.provider='fake'`
- [x] Step evaluator continua rule-based (qualidade da reply do fake é irrelevante pro fluxo)

### Step 4: Keywords CSV-aware

- [x] `parseKeywords` reescrita usando `encoding/csv` com `LazyQuotes` + pré-normalização para espaços pós-aspas
- [x] Fallback pra split simples se o reader falhar (compat com inputs antigos)
- [x] Hint do form de produto atualizado com exemplo `aposentadoria, "preciso revisar, é urgente"`
- [x] Matcher `Product.MatchesText` inalterado (substring case-insensitive)
- [x] Testes cobrindo frases com/sem vírgula, múltiplas citadas, espaços

### Step 5: Fixture completa

- [x] Seed de `ai_configs` pro especialista Dra. Clara com `provider='fake'`, `debounce=2s`
- [x] Contato e conversa limpos `Teste Playground` / `550e8400-...4400fe` (sem mensagens, sem lead)
- [x] Frases wa.me-friendly adicionadas como keyword extra nos 5 produtos previdenciários
- [x] `ON DUPLICATE KEY UPDATE` atualizado para refrescar `keywords` em re-runs do fixture

### Step 6: Playground dev HTMX

- [x] Handler novo em `internal/ai/interfaces/http/playground/` com 4 rotas
- [x] Interfaces `ContactLister` + `MessageLister` isolando o handler dos repos
- [x] Adapters `PlaygroundContactAdapter` + `PlaygroundMessageAdapter` delegando a `ConversationRepository.FindByTenantID` e `MessageRepository.FindByConversationID`
- [x] Reuso do `ReceiveMessageUseCase` existente — playground e WhatsApp real exercitam mesmo código
- [x] Templates `web/templates/ai/playground.html` + `playground_messages.html`
- [x] Polling HTMX de 2s pra puxar novas mensagens automaticamente
- [x] Botão "Reset" com `hx-confirm`
- [x] Montagem condicional no módulo AI — rotas só existem quando flag on

### Step 7: OWASP do playground

- [x] Teste sem tenant context → 404 (filtro por tenant retorna lista vazia)
- [x] Teste cross-tenant em GET/POST send/POST reset → 404
- [x] Controle positivo: mesma tenant só vê seus próprios contatos
- [x] Fake `tenantScopedContacts` pra detectar vazamento real (diferente do fake do handler_test.go)

### Step 8: Teste de integração end-to-end

- [x] Build tag `integration` (não roda no `go test ./...` default)
- [x] Helper `setupTestEnv` com testcontainers-go reusando pattern de `internal/shared/testhelper/`
- [x] `TestE2E_PrevidenciaFlow` — percorre steps do especialista previdenciário via fixture, asserta score ≥ 60
- [x] `TestE2E_ResetCommand_ReturnsStateToZero` — /reset zera state e devolve lead
- [x] `TestE2E_ResetCommand_DisabledFallsThrough` — com flag off, /reset vira mensagem normal

## Bugs corrigidos incidentalmente

- **`ConversationEngine.HandleMessages` criava `ConversationState` com PK vazio** (`NewConversationState("", ...)`). O primeiro INSERT passava, mas qualquer `Save` subsequente batia em `Error 1062 (23000): Duplicate entry '' for key 'conversation_states.PRIMARY'` — ou seja, nenhum avanço de step persistia na segunda mensagem em diante. Fix: `uuid.New().String()` no momento da criação. Descoberto enquanto escrevíamos o teste de integração.

## Gaps conhecidos (fora do escopo da F17)

- **Steps `free_text` não avançam sem metadata LLM.** O rule-based evaluator só casa `selection` e `number`. Para o e2e test usamos um helper `SetStepIndex` pra pular os dois steps de texto livre. Follow-up: permitir fallback pra aceitação simples (qualquer não-vazio) quando o provider for fake ou não retornar metadata.
- **`scoring_configs.qualified_column_id` não é aplicado.** O engine registra a qualificação via métrica mas não move o lead pra coluna qualificada quando o threshold é atingido. Follow-up: honrar `qualified_column_id` + `disqualified_column_id` no `ConversationEngine` após o último step.
- **Sidebar do playground** só aparece na própria página do playground (limitação arquitetural: os templates tenant são standalone, não há middleware comum injetando flags). Funcional, mas inconsistente. Follow-up: layout-context middleware pra injetar `AIPlaygroundEnabled` em todas as páginas tenant.
- **`internal/shared/config` não tem testes.** Pré-existente. Follow-up: adicionar `config_test.go` cobrindo defaults + env overrides.

## Cobertura pós-entrega

| Pacote | Cobertura |
|---|---|
| `internal/ai/application` | 89.6% |
| `internal/ai/domain` | 100.0% |
| `internal/ai/infrastructure` | 17.3% (pré-existente, adapters cobertos via integração) |
| `internal/ai/interfaces/http/playground` | 55.7% (handler + OWASP; templates não cobertos em unit) |
| `internal/product/interfaces/http` | 32.7% (pré-existente) |

## Referências

- Spec: `docs/artefatos/F17-fluxo-teste-manual/arquiteto-design/v1.md`
- Plano de implementação: `docs/artefatos/F17-fluxo-teste-manual/plano-implementacao/v1.md`
- Testes manuais: `rest/14-ai-playground.http`
- Observabilidade: métricas `ai_reset_commands_total` e `ai_playground_messages_total` em `docs/engenharia/observabilidade.md`
