# F23 — Qualificação Multi-Destino — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Adicionar 2 destinos intermediários (cross-sell por regra explícita e atendimento humano por faixa cinzenta de score) ao motor de qualificação do CRM jurídico, mantendo retrocompatibilidade.

**Architecture:**
- Domain estende `ScoringConfig` (faixa humana + colunas) e `Specialist` (configs de cross-sell); nova entidade `CrossSellRule`; `Lead` ganha `QualificationOutcome` e `CrossSellOriginLeadID`.
- Application: novo serviço `OutcomeCalculator` (3 zonas), `CrossSellRuleEvaluator` (avaliação ordenada por mensagem) e `CrossSellExecutor` (transição announce/silent/confirm).
- `ConversationEngine` é estendido para acionar avaliador antes do LLM e usar `OutcomeCalculator` no fim do script. Faixa humana ativa o handoff existente (`HandoffActive`).
- Templates HTMX estendidos em `web/templates/specialist/` (cross-sell + thresholds) e `web/templates/funnel/` (mapeamentos).

**Tech Stack:** Go 1.26, Gin, Gorm, MySQL, golang-migrate, testify, Zap, Prometheus.

**Plan deviations from spec (v1):**
- `ScoringConfig.Threshold` **não é renomeado**; adiciona `ThresholdHumanoMin` (default 0).
- **Não há `FunnelOutcomeMapping`**: `ScoringConfig` ganha `HumanColumnID` e `CrossSellColumnID` (per specialist, segue padrão atual de `QualifiedColumnID`/`DisqualifiedColumnID`).
- Trigger `intent` fica fora do MVP. Apenas `keyword` e `step_answer`.

---

## Phase A — Faixa cinzenta (atendimento humano)

### Task A1: ScoringConfig domain — adicionar ThresholdHumanoMin e HumanColumnID

**Files:**
- Modify: `internal/specialist/domain/scoring_config.go`
- Modify: `internal/specialist/domain/scoring_config_test.go`
- Modify: `internal/specialist/domain/errors.go` (se já existir; senão criar)

- [ ] **Step 1: Escrever os testes falhando**

```go
// scoring_config_test.go (acrescentar)
func TestScoringConfig_HumanoMinDefaultsToZero(t *testing.T) {
    sc, err := domain.NewScoringConfig("id", "spec-1", 80)
    require.NoError(t, err)
    assert.Equal(t, 0, sc.ThresholdHumanoMin)
}

func TestScoringConfig_UpdateThresholdHumanoMin_ValidRange(t *testing.T) {
    sc, _ := domain.NewScoringConfig("id", "spec-1", 80)
    err := sc.UpdateThresholdHumanoMin(50, 100)
    require.NoError(t, err)
    assert.Equal(t, 50, sc.ThresholdHumanoMin)
}

func TestScoringConfig_UpdateThresholdHumanoMin_RejectsAboveAprovado(t *testing.T) {
    sc, _ := domain.NewScoringConfig("id", "spec-1", 80)
    err := sc.UpdateThresholdHumanoMin(90, 100)
    require.ErrorIs(t, err, domain.ErrHumanoMinAboveAprovado)
}

func TestScoringConfig_UpdateThresholdHumanoMin_RejectsNegative(t *testing.T) {
    sc, _ := domain.NewScoringConfig("id", "spec-1", 80)
    err := sc.UpdateThresholdHumanoMin(-1, 100)
    require.ErrorIs(t, err, domain.ErrHumanoMinNegative)
}
```

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/specialist/domain/... -run TestScoringConfig_Humano -v`
Expected: FAIL com "undefined: ErrHumanoMinAboveAprovado" e "ThresholdHumanoMin".

- [ ] **Step 3: Implementar campo + método + erros**

```go
// scoring_config.go — substituir struct e adicionar método
type ScoringConfig struct {
    ID                   string
    SpecialistID         string
    Threshold            int
    ThresholdHumanoMin   int
    QualifiedColumnID    string
    DisqualifiedColumnID string
    HumanColumnID        string
    CrossSellColumnID    string
    CreatedAt            time.Time
    UpdatedAt            time.Time
}

func (sc *ScoringConfig) UpdateThresholdHumanoMin(value, totalPossible int) error {
    if value < 0 {
        return ErrHumanoMinNegative
    }
    if value > sc.Threshold {
        return ErrHumanoMinAboveAprovado
    }
    if value > totalPossible {
        return ErrScoringThresholdExceedsTotal
    }
    sc.ThresholdHumanoMin = value
    sc.UpdatedAt = time.Now()
    return nil
}
```

```go
// errors.go — acrescentar
var (
    ErrHumanoMinNegative      = errors.New("scoring: threshold humano min cannot be negative")
    ErrHumanoMinAboveAprovado = errors.New("scoring: threshold humano min cannot exceed threshold aprovado")
)
```

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/specialist/domain/... -v`
Expected: PASS em todos os tests do pacote.

- [ ] **Step 5: Commit**

```bash
git add internal/specialist/domain/scoring_config.go internal/specialist/domain/scoring_config_test.go internal/specialist/domain/errors.go
git commit -m "feat(F23): scoring config ganha threshold humano min e colunas humano/cross-sell"
```

---

### Task A2: OutcomeCalculator — calcular destino a partir do score

**Files:**
- Create: `internal/specialist/domain/outcome.go`
- Create: `internal/specialist/domain/outcome_test.go`

- [ ] **Step 1: Escrever os testes falhando**

```go
// outcome_test.go
package domain_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

func TestCalculateOutcome_BinaryWhenHumanoMinZero(t *testing.T) {
    sc := &domain.ScoringConfig{Threshold: 80, ThresholdHumanoMin: 0}
    assert.Equal(t, domain.OutcomeAprovado, domain.CalculateOutcome(sc, 80))
    assert.Equal(t, domain.OutcomeAprovado, domain.CalculateOutcome(sc, 100))
    assert.Equal(t, domain.OutcomeReprovado, domain.CalculateOutcome(sc, 0))
    assert.Equal(t, domain.OutcomeReprovado, domain.CalculateOutcome(sc, 79))
}

func TestCalculateOutcome_ThreeZonesWhenHumanoMinSet(t *testing.T) {
    sc := &domain.ScoringConfig{Threshold: 80, ThresholdHumanoMin: 50}
    assert.Equal(t, domain.OutcomeAprovado, domain.CalculateOutcome(sc, 80))
    assert.Equal(t, domain.OutcomeAprovado, domain.CalculateOutcome(sc, 100))
    assert.Equal(t, domain.OutcomeHumano, domain.CalculateOutcome(sc, 50))
    assert.Equal(t, domain.OutcomeHumano, domain.CalculateOutcome(sc, 79))
    assert.Equal(t, domain.OutcomeReprovado, domain.CalculateOutcome(sc, 49))
    assert.Equal(t, domain.OutcomeReprovado, domain.CalculateOutcome(sc, 0))
}
```

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/specialist/domain/... -run TestCalculateOutcome -v`
Expected: FAIL "undefined: domain.CalculateOutcome".

- [ ] **Step 3: Implementar**

```go
// outcome.go
package domain

type Outcome string

const (
    OutcomeEmAndamento Outcome = "em_andamento"
    OutcomeAprovado    Outcome = "aprovado"
    OutcomeHumano      Outcome = "humano"
    OutcomeCrossSell   Outcome = "cross_sell"
    OutcomeReprovado   Outcome = "reprovado"
)

func CalculateOutcome(sc *ScoringConfig, score int) Outcome {
    if score >= sc.Threshold {
        return OutcomeAprovado
    }
    if sc.ThresholdHumanoMin > 0 && score >= sc.ThresholdHumanoMin {
        return OutcomeHumano
    }
    return OutcomeReprovado
}
```

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/specialist/domain/... -v`

- [ ] **Step 5: Commit**

```bash
git add internal/specialist/domain/outcome.go internal/specialist/domain/outcome_test.go
git commit -m "feat(F23): adiciona OutcomeCalculator com 3 zonas (aprovado/humano/reprovado)"
```

---

### Task A3: Migration — adicionar colunas em scoring_configs

**Files:**
- Create: `migrations/000057_extend_scoring_configs_humano.up.sql`
- Create: `migrations/000057_extend_scoring_configs_humano.down.sql`

- [ ] **Step 1: Criar a migration up**

```sql
-- migrations/000057_extend_scoring_configs_humano.up.sql
ALTER TABLE scoring_configs
    ADD COLUMN threshold_humano_min INT NOT NULL DEFAULT 0,
    ADD COLUMN human_column_id      VARCHAR(36) NULL,
    ADD COLUMN cross_sell_column_id VARCHAR(36) NULL;
```

- [ ] **Step 2: Criar a migration down**

```sql
-- migrations/000057_extend_scoring_configs_humano.down.sql
ALTER TABLE scoring_configs
    DROP COLUMN threshold_humano_min,
    DROP COLUMN human_column_id,
    DROP COLUMN cross_sell_column_id;
```

- [ ] **Step 3: Aplicar e validar**

Run: `make migrate-up` (ou comando equivalente do projeto)
Then: `make migrate-status` ou inspecionar tabela em ambiente de dev.
Expected: migration 000057 aplicada; colunas presentes; defaults aplicados em rows existentes.

- [ ] **Step 4: Commit**

```bash
git add migrations/000057_extend_scoring_configs_humano.up.sql migrations/000057_extend_scoring_configs_humano.down.sql
git commit -m "chore(F23): migration adiciona threshold_humano_min e colunas humano/cross-sell em scoring_configs"
```

---

### Task A4: Repo gorm — mapear novos campos

**Files:**
- Modify: `internal/specialist/infrastructure/gorm_scoring_config_repository.go`
- Modify: `internal/specialist/infrastructure/gorm_scoring_config_repository_test.go` (se existir; senão criar mínimo)

- [ ] **Step 1: Escrever teste de round-trip**

Cria/usa testcontainer MySQL conforme padrão do projeto e verifica:

```go
func TestGormScoringConfig_PersistsHumanoFields(t *testing.T) {
    db := setupTestDB(t) // padrão do projeto, ver outros _test.go da pasta
    repo := infrastructure.NewGormScoringConfigRepository(db)
    sc := &domain.ScoringConfig{
        ID: uuid.NewString(), SpecialistID: "spec-1", Threshold: 80,
        ThresholdHumanoMin: 50, HumanColumnID: "col-h", CrossSellColumnID: "col-cs",
    }
    require.NoError(t, repo.Save(ctx, sc))
    got, err := repo.FindBySpecialistID(ctx, "spec-1")
    require.NoError(t, err)
    assert.Equal(t, 50, got.ThresholdHumanoMin)
    assert.Equal(t, "col-h", got.HumanColumnID)
    assert.Equal(t, "col-cs", got.CrossSellColumnID)
}
```

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/specialist/infrastructure/... -run TestGormScoringConfig_PersistsHumano -v`
Expected: FAIL — campos não mapeados.

- [ ] **Step 3: Estender o struct gorm e métodos**

Adicionar tags em `scoringConfigGorm` (ou nome equivalente no arquivo) e mapear ↔ domain:

```go
type scoringConfigGorm struct {
    ID                   string `gorm:"primaryKey;type:varchar(36)"`
    SpecialistID         string `gorm:"index;type:varchar(36)"`
    Threshold            int
    ThresholdHumanoMin   int
    QualifiedColumnID    string `gorm:"type:varchar(36)"`
    DisqualifiedColumnID string `gorm:"type:varchar(36)"`
    HumanColumnID        string `gorm:"type:varchar(36)"`
    CrossSellColumnID    string `gorm:"type:varchar(36)"`
    CreatedAt            time.Time
    UpdatedAt            time.Time
}
```

E ajustar `toDomain` / `fromDomain` para incluir os novos campos.

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/specialist/infrastructure/... -v`

- [ ] **Step 5: Commit**

```bash
git add internal/specialist/infrastructure/gorm_scoring_config_repository.go internal/specialist/infrastructure/gorm_scoring_config_repository_test.go
git commit -m "feat(F23): repo gorm persiste campos humano e cross-sell do scoring config"
```

---

### Task A5: ConversationEngine — usar OutcomeCalculator e disparar handoff

**Files:**
- Modify: `internal/ai/application/conversation_engine.go`
- Modify: `internal/ai/application/conversation_engine_test.go`

Buscar o trecho que move o lead para QualifiedColumnID/DisqualifiedColumnID (linhas ~74–79 e onde steps são finalizados).

- [ ] **Step 1: Adicionar dependências de handoff**

Adiciona ao `ConversationEngine`:
- `handoffActivator HandoffActivator` (interface com `Activate(ctx, conversationID string) error`).

A interface `HandoffActivator` no engine evita import direto do pacote `application` da AI sobre si mesmo. A implementação concreta é `*handoff.ActivateHandoffUseCase` injetada no wireup.

```go
type HandoffActivator interface {
    Activate(ctx context.Context, conversationID string) error
}
```

Atualizar construtor para receber `handoffActivator HandoffActivator`.

- [ ] **Step 2: Escrever teste de outcome humano**

```go
func TestConversationEngine_ScoringInHumanZone_PausesAIAndMovesLead(t *testing.T) {
    // setup com ScoringConfig{Threshold: 80, ThresholdHumanoMin: 50, HumanColumnID: "col-h"}
    // simula conclusão de steps com score = 60
    // mensagem do cliente que conclui último step
    // ...
    err := engine.HandleMessages(ctx, tenantID, convID, specID, prodID, []string{"sim"})
    require.NoError(t, err)
    assert.True(t, fakeHandoff.WasActivatedFor(convID))
    assert.Equal(t, "col-h", fakeLeadUpdater.LastColumnFor(convID))
}
```

- [ ] **Step 3: Rodar e ver falhar**

Run: `go test ./internal/ai/application/... -run TestConversationEngine_ScoringInHumanZone -v`
Expected: FAIL — handoff não disparado, coluna errada.

- [ ] **Step 4: Substituir lógica binária por OutcomeCalculator**

No ponto onde o engine hoje compara score com Threshold, trocar para:

```go
outcome := specDomain.CalculateOutcome(scoringCfg, score)
switch outcome {
case specDomain.OutcomeAprovado:
    if scoringCfg.QualifiedColumnID != "" {
        _ = e.leadUpdater.MoveLeadToColumn(ctx, conversationID, scoringCfg.QualifiedColumnID)
    }
case specDomain.OutcomeHumano:
    if scoringCfg.HumanColumnID != "" {
        _ = e.leadUpdater.MoveLeadToColumn(ctx, conversationID, scoringCfg.HumanColumnID)
    }
    if e.handoffActivator != nil {
        _ = e.handoffActivator.Activate(ctx, conversationID)
    }
case specDomain.OutcomeReprovado:
    if scoringCfg.DisqualifiedColumnID != "" {
        _ = e.leadUpdater.MoveLeadToColumn(ctx, conversationID, scoringCfg.DisqualifiedColumnID)
    }
}
```

- [ ] **Step 5: Rodar e ver passar**

Run: `go test ./internal/ai/application/... -v`

- [ ] **Step 6: Atualizar o wireup em cmd/api/main.go**

Localizar onde `NewConversationEngine` é chamado e injetar o `ActivateHandoffUseCase` já existente em `internal/ai/application/handoff.go`.

- [ ] **Step 7: Build inteiro**

Run: `go build ./...`
Expected: 0 erros.

- [ ] **Step 8: Commit**

```bash
git add internal/ai/application/conversation_engine.go internal/ai/application/conversation_engine_test.go cmd/api/main.go
git commit -m "feat(F23): engine usa OutcomeCalculator e ativa handoff em outcome humano"
```

---

### Task A6: Persistir QualificationOutcome no Lead

**Files:**
- Modify: `internal/funnel/domain/lead.go`
- Modify: `internal/funnel/domain/lead_test.go`

- [ ] **Step 1: Teste falhando**

```go
func TestLead_DefaultsOutcomeEmAndamento(t *testing.T) {
    lead, err := domain.NewLead("id-1", "tenant-1", "funnel-1", "col-1", "Nome", "")
    require.NoError(t, err)
    assert.Equal(t, domain.QualificationOutcomeEmAndamento, lead.QualificationOutcome)
}

func TestLead_SetOutcome(t *testing.T) {
    lead, _ := domain.NewLead("id-1", "tenant-1", "funnel-1", "col-1", "Nome", "")
    lead.SetQualificationOutcome(domain.QualificationOutcomeHumano)
    assert.Equal(t, domain.QualificationOutcomeHumano, lead.QualificationOutcome)
}
```

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/funnel/domain/... -run TestLead_(Defaults|SetOutcome) -v`

- [ ] **Step 3: Implementar enum + campo + setter**

```go
// lead.go (acrescentar)
type QualificationOutcome string

const (
    QualificationOutcomeEmAndamento QualificationOutcome = "em_andamento"
    QualificationOutcomeAprovado    QualificationOutcome = "aprovado"
    QualificationOutcomeHumano      QualificationOutcome = "humano"
    QualificationOutcomeCrossSell   QualificationOutcome = "cross_sell"
    QualificationOutcomeReprovado   QualificationOutcome = "reprovado"
)

type Lead struct {
    // ... campos existentes
    QualificationOutcome  QualificationOutcome
    CrossSellOriginLeadID *string
}

func (l *Lead) SetQualificationOutcome(o QualificationOutcome) {
    l.QualificationOutcome = o
    l.UpdatedAt = time.Now()
}
```

E inicializar `QualificationOutcome: QualificationOutcomeEmAndamento` no `NewLead`.

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/funnel/domain/... -v`

- [ ] **Step 5: Commit**

```bash
git add internal/funnel/domain/lead.go internal/funnel/domain/lead_test.go
git commit -m "feat(F23): Lead ganha QualificationOutcome e CrossSellOriginLeadID"
```

---

### Task A7: Migration — adicionar colunas em leads

**Files:**
- Create: `migrations/000058_extend_leads_outcome.up.sql`
- Create: `migrations/000058_extend_leads_outcome.down.sql`

- [ ] **Step 1: Migration up**

```sql
ALTER TABLE leads
    ADD COLUMN qualification_outcome    VARCHAR(20) NOT NULL DEFAULT 'em_andamento',
    ADD COLUMN cross_sell_origin_lead_id VARCHAR(36) NULL,
    ADD CONSTRAINT fk_leads_cs_origin FOREIGN KEY (cross_sell_origin_lead_id) REFERENCES leads(id) ON DELETE SET NULL;

CREATE INDEX idx_leads_outcome ON leads(qualification_outcome);
```

- [ ] **Step 2: Migration down**

```sql
DROP INDEX idx_leads_outcome ON leads;
ALTER TABLE leads
    DROP FOREIGN KEY fk_leads_cs_origin,
    DROP COLUMN cross_sell_origin_lead_id,
    DROP COLUMN qualification_outcome;
```

- [ ] **Step 3: Aplicar**

Run: `make migrate-up`
Expected: 000058 aplicada com sucesso.

- [ ] **Step 4: Atualizar repo gorm de Lead**

Em `internal/funnel/infrastructure/gorm_lead_repository.go`, adicionar campos no struct gorm e mapeamento toDomain/fromDomain.

- [ ] **Step 5: Teste de round-trip**

```go
func TestGormLead_PersistsOutcomeFields(t *testing.T) {
    // ... cria lead, define outcome, salva, recarrega, asserta
}
```

Run: `go test ./internal/funnel/infrastructure/... -v`

- [ ] **Step 6: Commit**

```bash
git add migrations/000058_extend_leads_outcome.up.sql migrations/000058_extend_leads_outcome.down.sql internal/funnel/infrastructure/gorm_lead_repository.go internal/funnel/infrastructure/gorm_lead_repository_test.go
git commit -m "chore(F23): persiste qualification_outcome e cross_sell_origin no lead"
```

---

### Task A8: Engine grava QualificationOutcome no Lead

**Files:**
- Modify: `internal/ai/application/conversation_engine.go`
- Modify: `internal/ai/application/conversation_engine_test.go`

- [ ] **Step 1: Estender LeadUpdater**

```go
type LeadUpdater interface {
    UpdateLeadScore(ctx context.Context, conversationID string, score int) error
    MoveLeadToColumn(ctx context.Context, conversationID, columnID string) error
    SetOutcome(ctx context.Context, conversationID string, outcome string) error
}
```

E implementação correspondente em `internal/funnel/application/lead_updater_impl.go` (ou onde estiver) chamando o repo do lead.

- [ ] **Step 2: Teste falhando**

```go
func TestConversationEngine_PersistsOutcomeInLead(t *testing.T) {
    // após decisão de aprovado/humano/reprovado, fakeLeadUpdater.LastOutcomeFor(convID) deve refletir
}
```

- [ ] **Step 3: Implementar e rodar**

No switch do Task A5, antes de mover a coluna chama:

```go
_ = e.leadUpdater.SetOutcome(ctx, conversationID, string(outcome))
```

Run: `go test ./internal/ai/... -v`

- [ ] **Step 4: Commit**

```bash
git add internal/ai/application/conversation_engine.go internal/ai/application/conversation_engine_test.go internal/funnel/application/lead_updater_impl.go
git commit -m "feat(F23): engine grava outcome no lead via LeadUpdater"
```

---

### Task A9: HTMX scoring section — exibir e editar threshold humano min e human column

**Files:**
- Modify: `web/templates/specialist/scoring_section.html`
- Modify: handler HTMX (provavelmente em `internal/specialist/interfaces/http/`)
- Modify: testes do handler

- [ ] **Step 1: Acrescentar campos no template**

```html
<label>Threshold aprovado <input type="number" name="threshold" value="{{ .Threshold }}" min="1"></label>
<label>Threshold humano min <input type="number" name="threshold_humano_min" value="{{ .ThresholdHumanoMin }}" min="0"></label>
<label>Coluna p/ aprovado <select name="qualified_column_id">{{ ... }}</select></label>
<label>Coluna p/ humano <select name="human_column_id">{{ ... }}</select></label>
<label>Coluna p/ desqualificado <select name="disqualified_column_id">{{ ... }}</select></label>
<label>Coluna p/ cross-sell <select name="cross_sell_column_id">{{ ... }}</select></label>
<small id="hint">Score ≥ {{ .Threshold }} aprovado · faixa cinzenta a partir de {{ .ThresholdHumanoMin }} (humano) · abaixo de {{ .ThresholdHumanoMin }} reprovado</small>
```

- [ ] **Step 2: Atualizar handler**

Bind dos campos novos, validar via domain (`UpdateThresholdHumanoMin`), persistir.

- [ ] **Step 3: Teste de handler**

```go
func TestUpdateScoringConfig_AcceptsHumanoMin(t *testing.T) {
    form := url.Values{
        "threshold": []string{"80"},
        "threshold_humano_min": []string{"50"},
        "human_column_id": []string{"col-h"},
        "qualified_column_id": []string{"col-q"},
        "disqualified_column_id": []string{"col-d"},
        "cross_sell_column_id": []string{"col-cs"},
    }
    // POST e asserta 200 + valor persistido
}
```

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/specialist/... -v`

- [ ] **Step 5: Smoke manual**

Subir o app local, abrir tela de treinamento de um especialista, verificar que os campos aparecem e salvam.

- [ ] **Step 6: Commit**

```bash
git add web/templates/specialist/scoring_section.html internal/specialist/interfaces/http/
git commit -m "feat(F23): tela de scoring permite configurar threshold humano e colunas humano/cross-sell"
```

---

## Phase B — Cross-sell por regra explícita

### Task B1: Specialist domain — campos de cross-sell

**Files:**
- Modify: `internal/specialist/domain/specialist.go`
- Modify: `internal/specialist/domain/specialist_test.go`

- [ ] **Step 1: Testes falhando**

```go
func TestSpecialist_CrossSellDefaultsDisabled(t *testing.T) {
    s, _ := domain.NewSpecialist("id", "tenant", "Nome", "...", domain.SpecialistTypeJuridico)
    assert.False(t, s.CrossSellEnabled)
    assert.Equal(t, domain.CrossSellModeAnnounce, s.CrossSellMode)
}

func TestSpecialist_EnableCrossSell_RequiresTemplateInAnnounceMode(t *testing.T) {
    s, _ := domain.NewSpecialist("id", "tenant", "Nome", "...", domain.SpecialistTypeJuridico)
    err := s.EnableCrossSell(domain.CrossSellModeAnnounce, "")
    require.ErrorIs(t, err, domain.ErrCrossSellTemplateRequired)
}

func TestSpecialist_EnableCrossSell_SilentDoesNotRequireTemplate(t *testing.T) {
    s, _ := domain.NewSpecialist("id", "tenant", "Nome", "...", domain.SpecialistTypeJuridico)
    require.NoError(t, s.EnableCrossSell(domain.CrossSellModeSilent, ""))
    assert.True(t, s.CrossSellEnabled)
}
```

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/specialist/domain/... -run TestSpecialist_CrossSell -v`

- [ ] **Step 3: Implementar**

```go
type CrossSellMode string

const (
    CrossSellModeAnnounce CrossSellMode = "announce"
    CrossSellModeSilent   CrossSellMode = "silent"
    CrossSellModeConfirm  CrossSellMode = "confirm"
)

type Specialist struct {
    // ... campos existentes
    CrossSellEnabled                bool
    CrossSellMode                   CrossSellMode
    CrossSellAnnouncementTemplate   string
    AllowAICrossSellSuggestion      bool
}

func (s *Specialist) EnableCrossSell(mode CrossSellMode, template string) error {
    if mode == CrossSellModeAnnounce && strings.TrimSpace(template) == "" {
        return ErrCrossSellTemplateRequired
    }
    s.CrossSellEnabled = true
    s.CrossSellMode = mode
    s.CrossSellAnnouncementTemplate = template
    s.UpdatedAt = time.Now()
    return nil
}

func (s *Specialist) DisableCrossSell() {
    s.CrossSellEnabled = false
    s.UpdatedAt = time.Now()
}
```

E inicializar `CrossSellMode: CrossSellModeAnnounce` no construtor.

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/specialist/domain/... -v`

- [ ] **Step 5: Commit**

```bash
git add internal/specialist/domain/
git commit -m "feat(F23): specialist ganha campos de cross-sell (enabled, mode, template, ai-suggestion)"
```

---

### Task B2: Migration — campos de cross-sell em specialists

**Files:**
- Create: `migrations/000059_extend_specialists_cross_sell.up.sql`
- Create: `migrations/000059_extend_specialists_cross_sell.down.sql`

- [ ] **Step 1: Up**

```sql
ALTER TABLE specialists
    ADD COLUMN cross_sell_enabled                TINYINT(1) NOT NULL DEFAULT 0,
    ADD COLUMN cross_sell_mode                   VARCHAR(16) NOT NULL DEFAULT 'announce',
    ADD COLUMN cross_sell_announcement_template TEXT NULL,
    ADD COLUMN allow_ai_cross_sell_suggestion   TINYINT(1) NOT NULL DEFAULT 0;
```

- [ ] **Step 2: Down**

```sql
ALTER TABLE specialists
    DROP COLUMN cross_sell_enabled,
    DROP COLUMN cross_sell_mode,
    DROP COLUMN cross_sell_announcement_template,
    DROP COLUMN allow_ai_cross_sell_suggestion;
```

- [ ] **Step 3: Aplicar e atualizar repo gorm**

Run: `make migrate-up`

Adicionar campos no struct gorm em `internal/specialist/infrastructure/gorm_specialist_repository.go` e mapeamento.

- [ ] **Step 4: Teste de round-trip**

```go
func TestGormSpecialist_PersistsCrossSellFields(t *testing.T) { ... }
```

Run: `go test ./internal/specialist/infrastructure/... -v`

- [ ] **Step 5: Commit**

```bash
git add migrations/000059_*.sql internal/specialist/infrastructure/
git commit -m "chore(F23): persiste campos de cross-sell em specialists"
```

---

### Task B3: CrossSellRule domain

**Files:**
- Create: `internal/specialist/domain/cross_sell_rule.go`
- Create: `internal/specialist/domain/cross_sell_rule_test.go`

- [ ] **Step 1: Testes falhando**

```go
func TestNewCrossSellRule_ValidKeyword(t *testing.T) {
    rule, err := domain.NewCrossSellRule("id", "spec-1", 1, domain.CrossSellTriggerKeyword,
        domain.KeywordTrigger{Termos: []string{"trabalhista"}}, "prod-2")
    require.NoError(t, err)
    assert.Equal(t, domain.CrossSellTriggerKeyword, rule.TriggerType)
    assert.True(t, rule.Ativo)
}

func TestNewCrossSellRule_RejectsEmptyKeywords(t *testing.T) {
    _, err := domain.NewCrossSellRule("id", "spec-1", 1, domain.CrossSellTriggerKeyword,
        domain.KeywordTrigger{Termos: []string{}}, "prod-2")
    require.ErrorIs(t, err, domain.ErrKeywordTriggerEmpty)
}

func TestNewCrossSellRule_RejectsInvalidStepRegex(t *testing.T) {
    _, err := domain.NewCrossSellRule("id", "spec-1", 1, domain.CrossSellTriggerStepAnswer,
        domain.StepAnswerTrigger{StepID: "step-1", Regex: "(invalid"}, "prod-2")
    require.ErrorIs(t, err, domain.ErrInvalidRegex)
}
```

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/specialist/domain/... -run TestNewCrossSellRule -v`

- [ ] **Step 3: Implementar**

```go
package domain

import (
    "regexp"
    "strings"
    "time"
)

type CrossSellTriggerType string

const (
    CrossSellTriggerKeyword     CrossSellTriggerType = "keyword"
    CrossSellTriggerStepAnswer  CrossSellTriggerType = "step_answer"
)

type KeywordTrigger struct {
    Termos []string `json:"termos"`
}

type StepAnswerTrigger struct {
    StepID string `json:"step_id"`
    Regex  string `json:"regex"`
}

type CrossSellRule struct {
    ID              string
    SpecialistID    string
    Ordem           int
    TriggerType     CrossSellTriggerType
    TriggerConfig   any
    TargetProductID string
    Ativo           bool
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

func NewCrossSellRule(id, specialistID string, ordem int,
    triggerType CrossSellTriggerType, triggerConfig any, targetProductID string,
) (*CrossSellRule, error) {
    if specialistID == "" {
        return nil, ErrSpecialistIDRequired
    }
    if targetProductID == "" {
        return nil, ErrTargetProductRequired
    }
    if err := validateTriggerConfig(triggerType, triggerConfig); err != nil {
        return nil, err
    }
    now := time.Now()
    return &CrossSellRule{
        ID: id, SpecialistID: specialistID, Ordem: ordem,
        TriggerType: triggerType, TriggerConfig: triggerConfig,
        TargetProductID: targetProductID, Ativo: true,
        CreatedAt: now, UpdatedAt: now,
    }, nil
}

func validateTriggerConfig(t CrossSellTriggerType, cfg any) error {
    switch t {
    case CrossSellTriggerKeyword:
        kw, ok := cfg.(KeywordTrigger)
        if !ok || len(kw.Termos) == 0 {
            return ErrKeywordTriggerEmpty
        }
        for _, term := range kw.Termos {
            if strings.TrimSpace(term) == "" {
                return ErrKeywordTriggerEmpty
            }
        }
    case CrossSellTriggerStepAnswer:
        sa, ok := cfg.(StepAnswerTrigger)
        if !ok || sa.StepID == "" {
            return ErrStepAnswerTriggerInvalid
        }
        if _, err := regexp.Compile(sa.Regex); err != nil {
            return ErrInvalidRegex
        }
    default:
        return ErrUnsupportedTrigger
    }
    return nil
}
```

E acrescentar erros em `errors.go`.

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/specialist/domain/... -v`

- [ ] **Step 5: Commit**

```bash
git add internal/specialist/domain/cross_sell_rule.go internal/specialist/domain/cross_sell_rule_test.go internal/specialist/domain/errors.go
git commit -m "feat(F23): domain CrossSellRule com triggers keyword e step_answer"
```

---

### Task B4: Migration — cross_sell_rules

**Files:**
- Create: `migrations/000060_create_cross_sell_rules.up.sql`
- Create: `migrations/000060_create_cross_sell_rules.down.sql`

- [ ] **Step 1: Up**

```sql
CREATE TABLE cross_sell_rules (
    id                 VARCHAR(36) PRIMARY KEY,
    specialist_id      VARCHAR(36) NOT NULL,
    ordem              INT NOT NULL DEFAULT 0,
    trigger_type       VARCHAR(20) NOT NULL,
    trigger_config     JSON NOT NULL,
    target_product_id  VARCHAR(36) NOT NULL,
    ativo              TINYINT(1) NOT NULL DEFAULT 1,
    created_at         DATETIME(3) NOT NULL,
    updated_at         DATETIME(3) NOT NULL,
    KEY idx_csr_spec (specialist_id, ordem),
    KEY idx_csr_target (target_product_id),
    CONSTRAINT fk_csr_spec FOREIGN KEY (specialist_id) REFERENCES specialists(id) ON DELETE CASCADE,
    CONSTRAINT fk_csr_product FOREIGN KEY (target_product_id) REFERENCES products(id) ON DELETE RESTRICT
);
```

- [ ] **Step 2: Down**

```sql
DROP TABLE cross_sell_rules;
```

- [ ] **Step 3: Aplicar**

Run: `make migrate-up`

- [ ] **Step 4: Commit**

```bash
git add migrations/000060_*.sql
git commit -m "chore(F23): cria tabela cross_sell_rules"
```

---

### Task B5: Repo CrossSellRule

**Files:**
- Create: `internal/specialist/domain/cross_sell_rule_repository.go`
- Create: `internal/specialist/infrastructure/gorm_cross_sell_rule_repository.go`
- Create: `internal/specialist/infrastructure/gorm_cross_sell_rule_repository_test.go`

- [ ] **Step 1: Interface**

```go
// cross_sell_rule_repository.go
package domain

import "context"

type CrossSellRuleRepository interface {
    Save(ctx context.Context, rule *CrossSellRule) error
    FindByID(ctx context.Context, id string) (*CrossSellRule, error)
    ListBySpecialistID(ctx context.Context, specialistID string) ([]*CrossSellRule, error)
    ListActiveBySpecialistOrdered(ctx context.Context, specialistID string) ([]*CrossSellRule, error)
    Delete(ctx context.Context, id string) error
}
```

- [ ] **Step 2: Teste de round-trip**

```go
func TestGormCrossSellRule_RoundTripWithJSONTrigger(t *testing.T) {
    db := setupTestDB(t)
    repo := infrastructure.NewGormCrossSellRuleRepository(db)
    rule, _ := domain.NewCrossSellRule(uuid.NewString(), "spec-1", 0, domain.CrossSellTriggerKeyword,
        domain.KeywordTrigger{Termos: []string{"trabalhista"}}, "prod-2")
    require.NoError(t, repo.Save(ctx, rule))
    list, _ := repo.ListActiveBySpecialistOrdered(ctx, "spec-1")
    require.Len(t, list, 1)
    kw := list[0].TriggerConfig.(domain.KeywordTrigger)
    assert.Equal(t, []string{"trabalhista"}, kw.Termos)
}
```

- [ ] **Step 3: Rodar e ver falhar**

Run: `go test ./internal/specialist/infrastructure/... -run TestGormCrossSellRule -v`

- [ ] **Step 4: Implementar gorm repo**

Struct gorm com `TriggerConfigJSON datatypes.JSON`, e na conversão pra domain decodifica conforme `TriggerType`.

- [ ] **Step 5: Rodar e ver passar**

Run: `go test ./internal/specialist/infrastructure/... -v`

- [ ] **Step 6: Commit**

```bash
git add internal/specialist/domain/cross_sell_rule_repository.go internal/specialist/infrastructure/gorm_cross_sell_rule_repository.go internal/specialist/infrastructure/gorm_cross_sell_rule_repository_test.go
git commit -m "feat(F23): repo gorm de CrossSellRule com serializacao JSON do trigger"
```

---

### Task B6: CrossSellRuleEvaluator (application)

**Files:**
- Create: `internal/ai/application/cross_sell_rule_evaluator.go`
- Create: `internal/ai/application/cross_sell_rule_evaluator_test.go`

Avaliador puro: recebe regras + contexto da mensagem (texto, mapa de respostas de step) e retorna a primeira regra ativa que dispara.

- [ ] **Step 1: Testes falhando**

```go
func TestEvaluator_KeywordMatchCaseInsensitiveAccentsNormalized(t *testing.T) {
    rule := mustNewKeywordRule(t, "spec-1", 0, []string{"trabalhista"}, "prod-2")
    eval := application.NewCrossSellRuleEvaluator()
    match := eval.Evaluate([]*specDomain.CrossSellRule{rule}, "Tenho dúvida TRABALHISTA hoje", nil)
    require.NotNil(t, match)
    assert.Equal(t, rule.ID, match.ID)
}

func TestEvaluator_FirstByOrdemWins(t *testing.T) {
    r1 := mustNewKeywordRule(t, "spec-1", 0, []string{"trabalhista"}, "prod-A")
    r2 := mustNewKeywordRule(t, "spec-1", 1, []string{"trabalhista"}, "prod-B")
    eval := application.NewCrossSellRuleEvaluator()
    match := eval.Evaluate([]*specDomain.CrossSellRule{r1, r2}, "trabalhista", nil)
    assert.Equal(t, r1.ID, match.ID)
}

func TestEvaluator_InactiveRuleIsSkipped(t *testing.T) {
    r := mustNewKeywordRule(t, "spec-1", 0, []string{"x"}, "prod-A")
    r.Ativo = false
    eval := application.NewCrossSellRuleEvaluator()
    assert.Nil(t, eval.Evaluate([]*specDomain.CrossSellRule{r}, "x", nil))
}

func TestEvaluator_StepAnswerRegexMatch(t *testing.T) {
    rule, _ := specDomain.NewCrossSellRule("id", "spec-1", 0, specDomain.CrossSellTriggerStepAnswer,
        specDomain.StepAnswerTrigger{StepID: "step-2", Regex: `(?i)^sim$`}, "prod-2")
    eval := application.NewCrossSellRuleEvaluator()
    match := eval.Evaluate([]*specDomain.CrossSellRule{rule}, "irrelevante", map[string]string{"step-2": "Sim"})
    require.NotNil(t, match)
}
```

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/ai/application/... -run TestEvaluator -v`

- [ ] **Step 3: Implementar**

```go
package application

import (
    "regexp"
    "strings"
    "unicode"

    "golang.org/x/text/runes"
    "golang.org/x/text/transform"
    "golang.org/x/text/unicode/norm"

    specDomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type CrossSellRuleEvaluator struct{}

func NewCrossSellRuleEvaluator() *CrossSellRuleEvaluator { return &CrossSellRuleEvaluator{} }

func (e *CrossSellRuleEvaluator) Evaluate(
    rules []*specDomain.CrossSellRule,
    message string,
    stepAnswers map[string]string,
) *specDomain.CrossSellRule {
    norm := normalize(message)
    sortByOrdem(rules)
    for _, r := range rules {
        if !r.Ativo {
            continue
        }
        if matchRule(r, norm, stepAnswers) {
            return r
        }
    }
    return nil
}

func matchRule(r *specDomain.CrossSellRule, normMsg string, answers map[string]string) bool {
    switch r.TriggerType {
    case specDomain.CrossSellTriggerKeyword:
        kw, ok := r.TriggerConfig.(specDomain.KeywordTrigger)
        if !ok { return false }
        for _, t := range kw.Termos {
            if strings.Contains(normMsg, normalize(t)) {
                return true
            }
        }
    case specDomain.CrossSellTriggerStepAnswer:
        sa, ok := r.TriggerConfig.(specDomain.StepAnswerTrigger)
        if !ok { return false }
        ans, found := answers[sa.StepID]
        if !found { return false }
        re, err := regexp.Compile(sa.Regex)
        if err != nil { return false }
        return re.MatchString(ans)
    }
    return false
}

func normalize(s string) string {
    t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
    out, _, _ := transform.String(t, strings.ToLower(s))
    return out
}

func sortByOrdem(rules []*specDomain.CrossSellRule) {
    // sort.SliceStable usando r.Ordem
}
```

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/ai/application/... -v`

- [ ] **Step 5: Commit**

```bash
git add internal/ai/application/cross_sell_rule_evaluator.go internal/ai/application/cross_sell_rule_evaluator_test.go
git commit -m "feat(F23): CrossSellRuleEvaluator com matchers keyword e step_answer"
```

---

### Task B7: CrossSellExecutor (application)

**Files:**
- Create: `internal/ai/application/cross_sell_executor.go`
- Create: `internal/ai/application/cross_sell_executor_test.go`

Responsabilidades:
1. Resolver novo specialist a partir de `target_product_id` (via SpecialistProductRepository).
2. Renderizar template de anúncio se `mode == announce`.
3. Em `mode == confirm`, marcar conversation com `pending_cross_sell_rule_id` (campo novo no ConversationState) e enviar pergunta. (Este lado é só preparação; consumo da resposta vem em Task B8.)
4. Em `silent` ou após confirmação positiva: criar lead novo no funil do destino, vincular `cross_sell_origin_lead_id`, mover lead atual para coluna mapeada, trocar `current_specialist_id` na conversation.
5. Marcar lead atual com `QualificationOutcomeCrossSell`.

- [ ] **Step 1: Definir interfaces de dependência**

```go
type ProductSpecialistResolver interface {
    FindSpecialistByProduct(ctx context.Context, productID string) (specialistID, funnelID, initialColumnID string, err error)
}

type LeadFactory interface {
    CreateForCrossSell(ctx context.Context, originLeadID, tenantID, funnelID, columnID, specialistID string) (newLeadID string, err error)
}

type ConversationMover interface {
    MigrateSpecialist(ctx context.Context, conversationID, newSpecialistID string) error
    SetPendingCrossSell(ctx context.Context, conversationID, ruleID string) error
    ClearPendingCrossSell(ctx context.Context, conversationID string) error
}
```

- [ ] **Step 2: Testes falhando**

Cobrir: announce envia mensagem com `{{produto}}` substituído; silent não envia; confirm envia pergunta e seta pending; ID de origem é gravado no lead novo.

- [ ] **Step 3: Implementar**

```go
type CrossSellExecutor struct {
    productResolver   ProductSpecialistResolver
    leadFactory       LeadFactory
    conversationMover ConversationMover
    leadUpdater       LeadUpdater
    sender            MessageSender
    productNameLookup ProductNameLookup
}

// Execute decide a ação inicial baseada no modo. CompleteTransition deve ser chamada
// pelo caller após confirmação (modo confirm) ou imediatamente (announce/silent).
func (x *CrossSellExecutor) Execute(
    ctx context.Context,
    convID, tenantID, originLeadID, crossSellColumnID string,
    fromSpecialist *specDomain.Specialist,
    rule *specDomain.CrossSellRule,
) error {
    productName, err := x.productNameLookup.Name(ctx, rule.TargetProductID)
    if err != nil {
        return err
    }

    switch fromSpecialist.CrossSellMode {
    case specDomain.CrossSellModeConfirm:
        msg := fmt.Sprintf("Posso te conectar com nosso especialista em %s?", productName)
        if err := x.sender.SendAIResponse(ctx, tenantID, convID, msg); err != nil {
            return err
        }
        return x.conversationMover.SetPendingCrossSell(ctx, convID, rule.ID)
    case specDomain.CrossSellModeAnnounce:
        rendered := strings.ReplaceAll(fromSpecialist.CrossSellAnnouncementTemplate, "{{produto}}", productName)
        if err := x.sender.SendAIResponse(ctx, tenantID, convID, rendered); err != nil {
            return err
        }
    case specDomain.CrossSellModeSilent:
        // nenhuma mensagem
    }
    return x.CompleteTransition(ctx, convID, tenantID, originLeadID, crossSellColumnID, rule)
}

// CompleteTransition executa a transferência efetiva: cria novo lead, marca outcome,
// move lead atual para a coluna cross-sell e migra a conversation pro novo specialist.
// crossSellColumnID vem do ScoringConfig do specialist atual (caller resolve).
func (x *CrossSellExecutor) CompleteTransition(
    ctx context.Context,
    convID, tenantID, originLeadID, crossSellColumnID string,
    rule *specDomain.CrossSellRule,
) error {
    newSpecID, funnelID, colID, err := x.productResolver.FindSpecialistByProduct(ctx, rule.TargetProductID)
    if err != nil {
        return err
    }
    if _, err := x.leadFactory.CreateForCrossSell(ctx, originLeadID, tenantID, funnelID, colID, newSpecID); err != nil {
        return err
    }
    if err := x.leadUpdater.SetOutcome(ctx, convID, string(specDomain.OutcomeCrossSell)); err != nil {
        return err
    }
    if crossSellColumnID != "" {
        _ = x.leadUpdater.MoveLeadToColumn(ctx, convID, crossSellColumnID)
    }
    return x.conversationMover.MigrateSpecialist(ctx, convID, newSpecID)
}
```

> Nota: `crossSellColumnID` vem do `ScoringConfig` do specialist atual e é resolvido pelo engine antes de chamar o executor (evita acoplamento entre executor e ScoringConfigFinder).

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/ai/application/... -v`

- [ ] **Step 5: Commit**

```bash
git add internal/ai/application/cross_sell_executor.go internal/ai/application/cross_sell_executor_test.go
git commit -m "feat(F23): CrossSellExecutor (announce/silent/confirm + transicao)"
```

---

### Task B8: ConversationState — pending cross-sell rule ID

**Files:**
- Modify: `internal/ai/domain/conversation_state.go`
- Modify: `internal/ai/domain/conversation_state_test.go`
- Create: `migrations/000061_extend_conversation_state_pending_cs.up.sql`
- Create: `migrations/000061_extend_conversation_state_pending_cs.down.sql`
- Modify: repo gorm correspondente

- [ ] **Step 1: Domain field**

Adicionar `PendingCrossSellRuleID *string` ao ConversationState com setter/clearer e teste.

- [ ] **Step 2: Migration**

```sql
ALTER TABLE conversation_states
    ADD COLUMN pending_cross_sell_rule_id VARCHAR(36) NULL;
```

- [ ] **Step 3: Repo gorm + testes**

- [ ] **Step 4: Aplicar e rodar testes**

Run: `make migrate-up && go test ./internal/ai/... -v`

- [ ] **Step 5: Commit**

```bash
git add migrations/000061_*.sql internal/ai/domain/ internal/ai/infrastructure/
git commit -m "feat(F23): ConversationState armazena pending cross-sell rule para modo confirm"
```

---

### Task B9: Integrar evaluator + executor no ConversationEngine

**Files:**
- Modify: `internal/ai/application/conversation_engine.go`
- Modify: `internal/ai/application/conversation_engine_test.go`
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Estender struct e construtor com evaluator + executor + crossSellRuleRepo + specialistRepo**

- [ ] **Step 2: Teste falhando — regra dispara antes do LLM**

```go
func TestConversationEngine_KeywordRuleTriggersBeforeLLM(t *testing.T) {
    // setup specialist com cross-sell habilitado, regra keyword "trabalhista" → produto B
    // mensagem "tenho dúvida trabalhista"
    // espera: LLM NÃO foi chamado, executor.Execute foi chamado com a regra
}
```

- [ ] **Step 3: Inserir avaliação no início do HandleMessages (após loading do specialist)**

```go
if specialist.CrossSellEnabled {
    activeRules, _ := e.crossSellRuleRepo.ListActiveBySpecialistOrdered(ctx, specialistID)
    if len(activeRules) > 0 {
        match := e.crossSellEvaluator.Evaluate(activeRules, latestMessageText, currentStepAnswers)
        if match != nil {
            return e.crossSellExecutor.Execute(ctx, conversationID, tenantID, currentLeadID, specialist, match)
        }
    }
}
```

- [ ] **Step 4: Teste falhando — modo confirm grava pending e processa resposta**

Cliente responde "sim" → engine vê `pending_cross_sell_rule_id`, recarrega regra, chama executor com modo silent (já confirmado).

- [ ] **Step 5: Implementar branch de confirm pendente**

No início do HandleMessages, antes de qualquer outra lógica (após carregar state):

```go
if state.PendingCrossSellRuleID != nil {
    if isAffirmative(latestMessageText) {
        rule, err := e.crossSellRuleRepo.FindByID(ctx, *state.PendingCrossSellRuleID)
        if err != nil { return err }
        _ = e.conversationMover.ClearPendingCrossSell(ctx, conversationID)
        scoringCfg, _ := e.scoringFinder.FindBySpecialistID(ctx, specialistID)
        crossSellColID := ""
        if scoringCfg != nil { crossSellColID = scoringCfg.CrossSellColumnID }
        return e.crossSellExecutor.CompleteTransition(ctx, conversationID, tenantID, currentLeadID, crossSellColID, rule)
    }
    // resposta negativa: limpa pending, segue fluxo normal
    _ = e.conversationMover.ClearPendingCrossSell(ctx, conversationID)
}

// isAffirmative usa regex default `(?i)^(sim|s|ok|claro|pode|por favor)\b`
func isAffirmative(s string) bool {
    return regexp.MustCompile(`(?i)^(sim|s|ok|claro|pode|por favor)\b`).MatchString(strings.TrimSpace(s))
}
```

- [ ] **Step 6: Atualizar wireup em cmd/api/main.go**

- [ ] **Step 7: Build e testes**

Run: `go build ./... && go test ./internal/ai/... -v`

- [ ] **Step 8: Commit**

```bash
git add internal/ai/application/ cmd/api/main.go
git commit -m "feat(F23): engine consulta regras de cross-sell antes do LLM e processa confirmacao"
```

---

### Task B10: HTTP handlers — CRUD de CrossSellRule

**Files:**
- Create: `internal/specialist/interfaces/http/cross_sell_rule_handler.go`
- Create: `internal/specialist/interfaces/http/cross_sell_rule_handler_test.go`
- Modify: `internal/specialist/module_handlers.go` (ou onde rotas são registradas)

- [ ] **Step 1: Definir endpoints**

- `GET    /especialistas/:id/cross-sell-rules` → lista
- `POST   /especialistas/:id/cross-sell-rules` → cria
- `PUT    /especialistas/:id/cross-sell-rules/:rule_id` → atualiza
- `DELETE /especialistas/:id/cross-sell-rules/:rule_id` → remove
- `POST   /especialistas/:id/cross-sell-rules/:rule_id/move-up` e `/move-down` → reordenar

- [ ] **Step 2: Testes falhando (handler)**

Testes cobrindo: 401/403, isolamento de tenant (specialist de outro tenant retorna 404), criar regra válida, reordenar.

- [ ] **Step 3: Implementar**

Reusa padrão de outros handlers do módulo specialist (ver `module_handlers.go` existente).

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/specialist/... -v`

- [ ] **Step 5: Commit**

```bash
git add internal/specialist/interfaces/http/cross_sell_rule_handler.go internal/specialist/interfaces/http/cross_sell_rule_handler_test.go internal/specialist/module_handlers.go
git commit -m "feat(F23): handlers HTTP de CRUD de regras de cross-sell"
```

---

### Task B11: HTMX — telas de cross-sell

**Files:**
- Create: `web/templates/specialist/cross_sell_section.html`
- Create: `web/templates/specialist/cross_sell_rule_form.html`
- Create: `web/templates/specialist/cross_sell_rule_row.html`
- Modify: `web/templates/specialist/specialist_detail.html` (incluir cross_sell_section.html)

- [ ] **Step 1: cross_sell_section.html**

Toggle `cross_sell_enabled`, radio group de modo, textarea de template, checkbox de tool call IA. Lista de regras com move-up/move-down/edit/delete e botão "+ adicionar".

- [ ] **Step 2: cross_sell_rule_form.html**

Select de tipo de trigger; campos contextuais (textarea de termos pra keyword, dois inputs pra step_answer); select de produto alvo (filtrado para produtos do tenant que tenham specialist).

- [ ] **Step 3: cross_sell_rule_row.html**

Linha da listagem com botões HTMX (`hx-post`, `hx-delete`, etc.).

- [ ] **Step 4: Smoke manual**

Subir, criar regra com termo "trabalhista", apontar pra produto que tem specialist, salvar, listar.

- [ ] **Step 5: Commit**

```bash
git add web/templates/specialist/cross_sell_section.html web/templates/specialist/cross_sell_rule_form.html web/templates/specialist/cross_sell_rule_row.html web/templates/specialist/specialist_detail.html
git commit -m "feat(F23): templates HTMX de configuracao de cross-sell e regras"
```

---

## Phase C — Hardening, observabilidade e OWASP

### Task C1: Métricas Prometheus

**Files:**
- Modify: `internal/shared/observability/metrics.go` (ou onde métricas são declaradas)
- Modify: `internal/ai/application/conversation_engine.go`

- [ ] **Step 1: Declarar contadores e histograma**

```go
QualificationOutcomeTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
    Name: "qualification_outcome_total",
    Help: "Lead qualification outcomes",
}, []string{"tenant", "specialist", "outcome"})

CrossSellTriggeredTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
    Name: "cross_sell_triggered_total",
}, []string{"tenant", "specialist", "trigger_type"})

HumanHandoffResolutionLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
    Name: "human_handoff_resolution_latency_seconds",
}, []string{"tenant"})
```

- [ ] **Step 2: Incrementar nos pontos certos**

- Após `CalculateOutcome` no engine: `QualificationOutcomeTotal.WithLabelValues(...).Inc()`.
- No executor de cross-sell: `CrossSellTriggeredTotal.WithLabelValues(tenant, specialist, "rule_keyword"|"rule_step_answer").Inc()`.
- Quando lead em humano vira aprovado/reprovado manualmente, calcular delta.

- [ ] **Step 3: Smoke local (`/metrics`)**

- [ ] **Step 4: Commit**

```bash
git add internal/shared/observability/metrics.go internal/ai/application/conversation_engine.go
git commit -m "chore(F23): metricas Prometheus de outcome, cross-sell e latencia humano"
```

---

### Task C2: OWASP — testes de isolamento de tenant

**Files:**
- Create: `internal/specialist/interfaces/http/cross_sell_rule_owasp_test.go`

- [ ] **Step 1: Testes**

```go
func TestCrossSellRule_RejectsTargetProductFromOtherTenant(t *testing.T) { ... }
func TestCrossSellRule_RejectsAccessToRuleOfOtherTenant(t *testing.T) { ... }
func TestCrossSellRule_Requires401WhenUnauthenticated(t *testing.T) { ... }
func TestCrossSellRule_Requires403WhenWrongPermission(t *testing.T) { ... }
```

- [ ] **Step 2: Implementar até passar**

Onde necessário, adicionar checagem `product.TenantID == ctx.TenantID()` nas validações de criação.

- [ ] **Step 3: Run**

Run: `go test ./internal/specialist/... -run OWASP -v`

- [ ] **Step 4: Commit**

```bash
git add internal/specialist/interfaces/http/cross_sell_rule_owasp_test.go
git commit -m "test(F23): OWASP - isolamento de tenant em CrossSellRule"
```

---

### Task C3: Cobertura, lint e build final

- [ ] **Step 1: Rodar coverage**

Run: `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1`
Expected: cobertura total ≥ 80%.

- [ ] **Step 2: Lint**

Run: `golangci-lint run ./...`
Expected: 0 issues.

- [ ] **Step 3: Build**

Run: `go build ./...`
Expected: sucesso.

- [ ] **Step 4: Atualizar `docs/processo/backlog.md` e `docs/artefatos/F23-qualificacao-multi-destino/status.md`**

Marcar F23 como concluído. Adicionar entrada na tabela do épico de IA/Funil.

- [ ] **Step 5: Commit final**

```bash
git add docs/processo/backlog.md docs/artefatos/F23-qualificacao-multi-destino/status.md
git commit -m "docs(F23): conclusao da feature qualificacao multi-destino"
```

- [ ] **Step 6: Atualizar `rest/`**

Adicionar arquivo `.http` em `rest/F23-qualificacao-multi-destino.http` com requests pros endpoints novos (CRUD CrossSellRule).

```bash
git add rest/F23-qualificacao-multi-destino.http
git commit -m "docs(F23): arquivos .http para endpoints de regras de cross-sell"
```

---

## Backlog (fora do MVP F23)

- Trigger `intent` em CrossSellRule (depende de classifier).
- Tool call `suggest_cross_sell` da IA (campo `AllowAICrossSellSuggestion` já existe, mas integração com LLM fica pra próxima feature).
- Triggers extras pra humano (sentimento, pedido explícito, guardrail repetitivo).
- Load balance/round-robin no humano usando responsável de F07.
- Migrar destinos pra `FunnelOutcomeMapping` (per funil) caso especialistas passem a compartilhar funis com mapeamentos diferentes.
- Bloqueio anti-loop de cross-sell ancestral (evitar A→B→A).
