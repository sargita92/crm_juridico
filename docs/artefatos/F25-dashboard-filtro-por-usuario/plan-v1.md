# F25 — Filtro de Dashboard por Usuário — Plano de Implementação (v1)

> **Para executores:** implementar task a task com TDD (teste falha → mínimo p/
> passar → refatora → commit). Steps com checkbox `- [ ]`. Ver design em
> [design-v1.md](design-v1.md).

**Goal:** Permitir que o owner do escritório selecione um operador no dashboard
tenant e veja só os dados dele (drill-down), com "Consolidado" como padrão.

**Architecture:** Reusa a fiação de filtro por usuário do `GetTenantDashboard`.
Novo campo `ViewUserID *string` no `TenantInput`; novo port `OperatorLister`
para popular o dropdown; handler valida `?user` (OWASP) na borda. Sem migration.

**Tech Stack:** Go, Gin, Gorm (MySQL), html/template, HTMX, testify,
testcontainers-go.

---

## Mapa de arquivos

| Arquivo | Ação | Responsabilidade |
|---------|------|------------------|
| `internal/dashboard/domain/operator.go` | criar | tipo `Operator{ID,Name}` |
| `internal/dashboard/application/get_tenant_dashboard.go` | modificar | `ViewUserID` + novo switch de filtro + `CurrentUserName` do usuário filtrado |
| `internal/dashboard/application/providers.go` | modificar | interface `OperatorLister` |
| `internal/dashboard/application/get_tenant_dashboard_test.go` | modificar | testes de drill-down/consolidado/não-owner |
| `internal/dashboard/infrastructure/gorm_user_lookup.go` | modificar | método `Operators` (JOIN) + check `OperatorLister` |
| `internal/dashboard/infrastructure/gorm_operator_lister_test.go` | criar | teste de integração de `Operators` + `seedUserTenant` |
| `internal/dashboard/interfaces/http/handler.go` | modificar | injeta `OperatorLister`; parse/valida `?user`; view model |
| `internal/dashboard/interfaces/http/tenant_handler_test.go` | modificar | fake `OperatorLister`, fake `FindByUserAndTenant`, testes do seletor |
| `internal/dashboard/interfaces/http/owasp_test.go` | modificar | casos OWASP do `?user` |
| `internal/dashboard/module.go` | modificar | passa `ul` como `OperatorLister` ao handler |
| `web/templates/tenant/dashboard/page.html` | modificar | `<select>` no header + `hx-include` no Atualizar |
| `web/templates/tenant/dashboard/content.html` | modificar | indicador "Vendo: X" dentro do fragmento |
| `rest/dashboard.http` | modificar | `?user=<id>` + casos OWASP |

---

## Task 1: Domínio `Operator` + filtro por `ViewUserID` (unit)

**Files:**
- Create: `internal/dashboard/domain/operator.go`
- Modify: `internal/dashboard/application/get_tenant_dashboard.go`
- Test: `internal/dashboard/application/get_tenant_dashboard_test.go`

- [ ] **Step 1.1: tipo de domínio** — `operator.go`:
```go
package domain

// Operator é um usuário não-owner do tenant, item do seletor do dashboard.
type Operator struct {
	ID   string
	Name string
}
```

- [ ] **Step 1.2: testes que falham** — adicionar em `get_tenant_dashboard_test.go`:
```go
func TestGetTenantDashboard_Owner_DrillsIntoOperator(t *testing.T) {
	fp := newFakes()
	ul := fakeUserLookup{m: map[string]string{"op1": "Bia"}}
	uc := application.NewGetTenantDashboard(fp, ul, fixedClock{t: time.Now()})
	view := "op1"
	out, err := uc.Execute(context.Background(), application.TenantInput{
		TenantID: "t1", UserID: "owner1", IsOwner: true, ViewUserID: &view,
	})
	require.NoError(t, err)
	assert.True(t, out.ScopeIsUser)
	require.NotNil(t, fp.lastUserFilter)
	assert.Equal(t, "op1", *fp.lastUserFilter)        // filtro vai pro provider
	assert.Equal(t, "op1", *fp.lastUserFilterProd)    // fan-out
	assert.Equal(t, "Bia", out.CurrentUserName)       // nome do usuário VISTO, não do requisitante
}

func TestGetTenantDashboard_Owner_NoView_Consolidated(t *testing.T) {
	fp := newFakes()
	uc := application.NewGetTenantDashboard(fp, fakeUserLookup{}, fixedClock{t: time.Now()})
	out, err := uc.Execute(context.Background(), application.TenantInput{
		TenantID: "t1", UserID: "owner1", IsOwner: true, ViewUserID: nil,
	})
	require.NoError(t, err)
	assert.False(t, out.ScopeIsUser)
	assert.Nil(t, fp.lastUserFilter)
}

func TestGetTenantDashboard_NonOwner_IgnoresViewUserID(t *testing.T) {
	fp := newFakes()
	ul := fakeUserLookup{m: map[string]string{"u2": "João"}}
	uc := application.NewGetTenantDashboard(fp, ul, fixedClock{t: time.Now()})
	view := "someoneElse"
	out, err := uc.Execute(context.Background(), application.TenantInput{
		TenantID: "t1", UserID: "u2", IsOwner: false, ViewUserID: &view,
	})
	require.NoError(t, err)
	require.NotNil(t, fp.lastUserFilter)
	assert.Equal(t, "u2", *fp.lastUserFilter)   // travado em si mesmo, ignora ViewUserID
	assert.Equal(t, "João", out.CurrentUserName)
}
```

- [ ] **Step 1.3: rodar testes (falham)** — `go test ./internal/dashboard/application/ -run DrillsIntoOperator -v` → FAIL (campo `ViewUserID` inexistente).

- [ ] **Step 1.4: implementar** em `get_tenant_dashboard.go`:
  - adicionar `ViewUserID *string` ao `TenantInput`;
  - trocar o cálculo do filtro:
```go
var userFilter *string
switch {
case !in.IsOwner:
	uid := in.UserID
	userFilter = &uid
case in.ViewUserID != nil:
	userFilter = in.ViewUserID
}
```
  - trocar a resolução do nome para o usuário filtrado:
```go
out.ScopeIsUser = userFilter != nil
if out.ScopeIsUser {
	if name, err := uc.users.UserName(ctx, *userFilter); err == nil {
		out.CurrentUserName = name
	}
}
```

- [ ] **Step 1.5: rodar testes (passam)** — `go test ./internal/dashboard/application/ -v` → todos PASS (inclusive os 5 antigos).

- [ ] **Step 1.6: commit** — `feat(dashboard): ViewUserID no TenantInput p/ drill-down do owner (F25)`

## Task 2: Port `OperatorLister` + impl Gorm (integração)

**Files:**
- Modify: `internal/dashboard/application/providers.go`
- Modify: `internal/dashboard/infrastructure/gorm_user_lookup.go`
- Test: `internal/dashboard/infrastructure/gorm_operator_lister_test.go` (criar)

- [ ] **Step 2.1: port** em `providers.go`:
```go
// OperatorLister lista os operadores (não-owners) de um tenant p/ o seletor.
type OperatorLister interface {
	Operators(ctx context.Context, tenantID string) ([]domain.Operator, error)
}
```

- [ ] **Step 2.2: teste de integração que falha** — `gorm_operator_lister_test.go` (package `infrastructure`):
```go
package infrastructure

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// seedUserTenant associa user↔tenant com is_owner.
func seedUserTenant(t *testing.T, db *gorm.DB, userID, tenantID string, isOwner bool) {
	t.Helper()
	err := db.Exec(`INSERT INTO user_tenants (user_id, tenant_id, is_owner, created_at, updated_at)
		VALUES (?, ?, ?, NOW(), NOW())`, userID, tenantID, isOwner).Error
	require.NoError(t, err)
}

func TestOperators_ReturnsOnlyNonOwnersOrdered(t *testing.T) {
	_, db := setupStatsRepo(t)
	lister := NewGormUserLookup(db)
	tenantID := seedTenant(t, db)

	owner := seedUser(t, db, "Zélia Owner")
	op2 := seedUser(t, db, "Bruno")
	op1 := seedUser(t, db, "Ana")
	seedUserTenant(t, db, owner, tenantID, true)
	seedUserTenant(t, db, op2, tenantID, false)
	seedUserTenant(t, db, op1, tenantID, false)

	// operador de OUTRO tenant não deve aparecer
	otherTenant := seedTenant(t, db)
	otherOp := seedUser(t, db, "Fora")
	seedUserTenant(t, db, otherOp, otherTenant, false)

	ops, err := lister.Operators(context.Background(), tenantID)
	require.NoError(t, err)
	require.Len(t, ops, 2)
	assert.Equal(t, "Ana", ops[0].Name)   // ordenado por nome
	assert.Equal(t, op1, ops[0].ID)
	assert.Equal(t, "Bruno", ops[1].Name)
}

func TestOperators_EmptyTenant(t *testing.T) {
	_, db := setupStatsRepo(t)
	lister := NewGormUserLookup(db)
	tenantID := seedTenant(t, db)
	ops, err := lister.Operators(context.Background(), tenantID)
	require.NoError(t, err)
	assert.NotNil(t, ops)
	assert.Empty(t, ops)
	_ = uuid.New()
}
```

- [ ] **Step 2.3: rodar (falha)** — `go test ./internal/dashboard/infrastructure/ -run TestOperators -v` → FAIL (método `Operators` inexistente).

- [ ] **Step 2.4: implementar** em `gorm_user_lookup.go`:
```go
// compile-time check adicional
var _ application.OperatorLister = (*GormUserLookup)(nil)

// Operators lista os usuários não-owner do tenant, ordenados por nome.
func (r *GormUserLookup) Operators(ctx context.Context, tenantID string) ([]domain.Operator, error) {
	var rows []domain.Operator
	err := r.db.WithContext(ctx).
		Table("user_tenants AS ut").
		Select("u.id AS id, u.name AS name").
		Joins("JOIN users u ON u.id = ut.user_id").
		Where("ut.tenant_id = ? AND ut.is_owner = ?", tenantID, false).
		Order("u.name ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []domain.Operator{}
	}
	return rows, nil
}
```
  (adicionar import de `domain`.)

- [ ] **Step 2.5: rodar (passa)** — `go test ./internal/dashboard/infrastructure/ -run TestOperators -v` → PASS.

- [ ] **Step 2.6: commit** — `feat(dashboard): OperatorLister lista operadores do tenant p/ seletor (F25)`

## Task 3: Handler — parse/validação de `?user` + view model (unit)

**Files:**
- Modify: `internal/dashboard/interfaces/http/handler.go`
- Modify: `internal/dashboard/module.go`
- Test: `internal/dashboard/interfaces/http/tenant_handler_test.go`

- [ ] **Step 3.1: estender fakes** em `tenant_handler_test.go`:
  - dar a `fakeUserTenants` um `membersByPair map[string]*authdomain.UserTenant` e implementar `FindByUserAndTenant` retornando-o;
  - dar a `fakeUserTenants` um `FindByTenantID` configurável (opcional);
  - novo fake `fakeOperatorLister struct{ ops []domain.Operator }` com `Operators(...)` retornando `ops`;
  - atualizar `newHandler` p/ aceitar e injetar o `fakeOperatorLister`;
  - ampliar os templates stub de `newTestRouter` p/ expor seleção:
    `... CANSELECT={{.CanSelectUser}} SELECTED={{.SelectedUserID}} OPS={{range .Operators}}{{.Name}},{{end}} FULL: {{.Stats.Bloco1_Funil.Total}}`.

- [ ] **Step 3.2: testes que falham**:
```go
func TestTenantDashboard_Owner_RendersSelector(t *testing.T) {
	h := newHandler(t, map[string]bool{"o1/t1": true},
		[]domain.Operator{{ID: "op1", Name: "Ana"}, {ID: "op2", Name: "Bruno"}})
	claims := &authdomain.TokenClaims{UserID: "o1", Role: authdomain.UserRoleUser, TenantID: "t1"}
	r := newTestRouter(t, h, claims, "t1")
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "CANSELECT=true")
	assert.Contains(t, rec.Body.String(), "Ana,Bruno,")
}

func TestTenantDashboard_Owner_DrillValidUser(t *testing.T) {
	h := newHandlerWithMembers(t,
		map[string]bool{"o1/t1": true},
		[]domain.Operator{{ID: "op1", Name: "Ana"}},
		map[string]*authdomain.UserTenant{"op1/t1": {UserID: "op1", TenantID: "t1", IsOwner: false}})
	claims := &authdomain.TokenClaims{UserID: "o1", Role: authdomain.UserRoleUser, TenantID: "t1"}
	r := newTestRouter(t, h, claims, "t1")
	req := httptest.NewRequest(http.MethodGet, "/dashboard/content?user=op1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "SELECTED=op1")
	require.NotNil(t, providerFilterOf(h))      // helper que expõe o fakeTenantHTTPProvider
	assert.Equal(t, "op1", *providerFilterOf(h))
}

func TestTenantDashboard_Owner_DrillInvalidUser_Consolidated(t *testing.T) {
	h := newHandlerWithMembers(t,
		map[string]bool{"o1/t1": true},
		[]domain.Operator{{ID: "op1", Name: "Ana"}},
		map[string]*authdomain.UserTenant{}) // ninguém é membro válido
	claims := &authdomain.TokenClaims{UserID: "o1", Role: authdomain.UserRoleUser, TenantID: "t1"}
	r := newTestRouter(t, h, claims, "t1")
	req := httptest.NewRequest(http.MethodGet, "/dashboard?user=foreign", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "SELECTED=")  // vazio → consolidado
}

func TestTenantDashboard_NonOwner_NoSelector_IgnoresUserParam(t *testing.T) {
	h := newHandler(t, map[string]bool{}, nil) // não-owner
	claims := &authdomain.TokenClaims{UserID: "u9", Role: authdomain.UserRoleUser, TenantID: "t1"}
	r := newTestRouter(t, h, claims, "t1")
	req := httptest.NewRequest(http.MethodGet, "/dashboard?user=op1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "CANSELECT=false")
}
```
  > Nota: definir os helpers `newHandlerWithMembers` (variação de `newHandler` que
  > injeta `membersByPair`) e `providerFilterOf` no mesmo arquivo de teste; manter
  > o `newHandler(t, isOwnerMap, ops)` simples por trás dele.

- [ ] **Step 3.3: rodar (falha)** — `go test ./internal/dashboard/interfaces/http/ -run TenantDashboard -v` → FAIL (assinatura de `NewHandler`/`newHandler` mudou; campos ausentes).

- [ ] **Step 3.4: implementar handler** — em `handler.go`:
  - `Handler` ganha campo `operators application.OperatorLister`; `NewHandler` recebe-o (último parâmetro antes do log, ou novo);
  - em `renderTenant`, após calcular `isOwner`:
```go
var operators []domain.Operator
var selectedUserID string
var viewUserID *string
if isOwner {
	if ops, err := h.operators.Operators(ctx, tenantID); err == nil {
		operators = ops
	} else {
		h.log.Warn("dashboard tenant: list operators", zap.Error(err), zap.String("tenant_id", tenantID))
	}
	if q := strings.TrimSpace(c.Query("user")); q != "" {
		ut, err := h.userTenants.FindByUserAndTenant(ctx, q, tenantID)
		switch {
		case err != nil:
			h.log.Warn("dashboard tenant: validate selected user", zap.Error(err), zap.String("tenant_id", tenantID))
		case ut != nil && !ut.IsOwner:
			id := q
			viewUserID = &id
			selectedUserID = q
		default:
			h.log.Warn("dashboard tenant: invalid user selection ignored",
				zap.String("tenant_id", tenantID), zap.String("requested_user", q))
		}
	}
}
```
  - passar `ViewUserID: viewUserID` ao `Execute`;
  - `c.HTML(..., gin.H{"Stats": vm, "TenantID": tenantID, "Operators": operators, "SelectedUserID": selectedUserID, "CanSelectUser": isOwner})`;
  - log `dashboard_rendered` ganha `zap.String("viewed_user_id", selectedUserID)`.
  - imports: `strings`, `domain`.

- [ ] **Step 3.5: wiring** — em `module.go`, `NewHandler(tenantUC, adminUC, userTenants, ul, log)` (passa `ul` como `OperatorLister`). Atualizar a chamada em `owasp_test.go` também (Task 5 cobre).

- [ ] **Step 3.6: rodar (passa)** — `go test ./internal/dashboard/interfaces/http/ -run TenantDashboard -v` → PASS.

- [ ] **Step 3.7: commit** — `feat(dashboard): handler valida ?user e expõe seletor ao owner (F25)`

## Task 4: Templates — seletor + indicador de escopo no fragmento

**Files:**
- Modify: `web/templates/tenant/dashboard/page.html`
- Modify: `web/templates/tenant/dashboard/content.html`

- [ ] **Step 4.1: page.html** — no `head-actions`, antes do botão Atualizar:
```html
{{if .CanSelectUser}}
<select name="user" class="dashboard-user-select"
        aria-label="Filtrar por operador"
        hx-get="/dashboard/content"
        hx-target="#dashboard-content"
        hx-swap="outerHTML"
        hx-trigger="change">
    <option value="">Consolidado (todos)</option>
    {{range .Operators}}
    <option value="{{.ID}}" {{if eq .ID $.SelectedUserID}}selected{{end}}>{{.Name}}</option>
    {{end}}
</select>
{{end}}
```
  E no botão Atualizar adicionar `hx-include="[name='user']"`.
  Remover o `{{if .Stats.ScopeIsUser}}<span class="scope-badge">...</span>{{end}}` do `<h1>` (vai pro fragmento).

- [ ] **Step 4.2: content.html** — no topo do `#dashboard-content`:
```html
<div id="dashboard-content" class="dashboard-grid">
    {{if .Stats.ScopeIsUser}}
    <div class="dashboard-scope-note">Vendo dados de <strong>{{.Stats.CurrentUserName}}</strong></div>
    {{end}}
    {{template "tenant/dashboard/_bloco1_funil.html" .}}
    ...
</div>
```

- [ ] **Step 4.3: verificar parse** — `go build ./...` e rodar app local; checar que `/dashboard` renderiza o `<select>` p/ owner e troca o fragmento ao selecionar (verificação manual no Task 5).

- [ ] **Step 4.4: commit** — `feat(dashboard): seletor de operador no header + indicador de escopo (F25)`

## Task 5: OWASP, .http, observabilidade e fechamento

**Files:**
- Modify: `internal/dashboard/interfaces/http/owasp_test.go`
- Modify: `rest/dashboard.http`
- Modify: `docs/artefatos/F25-dashboard-filtro-por-usuario/status.md`

- [ ] **Step 5.1: atualizar `setupOWASPRouter`** — passar o novo `OperatorLister` (fake vazio) ao `NewHandler`; dar à `fakeUserTenants` os membros necessários.

- [ ] **Step 5.2: testes OWASP que falham/passam**:
```go
func TestOWASP_Tenant_NonOwner_UserParam_StaysSelfScoped(t *testing.T) {
	e := setupOWASPRouter(t)
	userID := uuid.NewString()
	tok, _ := e.provider.Generate(authdomain.TokenClaims{UserID: userID, Role: authdomain.UserRoleUser, TenantID: e.tenantID})
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, owaspGet("/dashboard?user="+uuid.NewString(), owaspCookie(tok)))
	assert.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, e.tenantFake.lastUserFilter)
	assert.Equal(t, userID, *e.tenantFake.lastUserFilter) // ignora ?user, escopo travado em si
}

func TestOWASP_Tenant_UserParam_SQLi_NoEffect(t *testing.T) {
	e := setupOWASPRouter(t)
	tok := e.tokenUser(t, e.tenantID)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, owaspGet("/dashboard?user=1%27%20OR%20%271%27%3D%271", owaspCookie(tok)))
	assert.Equal(t, http.StatusOK, rec.Code)
}
```
  (owner+foreign→consolidado já é coberto pelo unit `DrillInvalidUser`; opcionalmente
  replicar aqui com token de owner se o `fakeUserTenants` do owasp expuser `IsOwner`.)

- [ ] **Step 5.3: rodar OWASP** — `go test ./internal/dashboard/interfaces/http/ -run OWASP -v` → PASS.

- [ ] **Step 5.4: `rest/dashboard.http`** — adicionar requests: `GET /dashboard?user=<id>` (owner drill), `GET /dashboard/content?user=<id>`, e nota OWASP (`?user` de não-owner ignorado).

- [ ] **Step 5.5: verificação local completa**:
  - `go build ./...`
  - `go test -short ./internal/dashboard/...`
  - `golangci-lint run ./internal/dashboard/...`
  - `go vet ./internal/dashboard/...`
  - integração: `go test -p 1 -count=1 ./internal/dashboard/...` (testcontainers)
  - cobertura: `go test -short -cover ./internal/dashboard/...` ≥ 80%
  - app local: `docker compose -f docker-compose.dev.yml up -d --build app`; logar como owner (`ana@mendescosta.adv.br`), abrir `/dashboard`, conferir seletor + troca de fragmento + retorno a Consolidado.

- [ ] **Step 5.6: atualizar `status.md`** (steps concluídos, commits) e **commit** — `docs(f25): status final + rest .http`.

## Self-review (cobertura do spec)

- Drill-down 1 por vez + Consolidado → Tasks 1, 3, 4. ✔
- Lista só operadores (não-owners) → Task 2 (query `is_owner=false`). ✔
- Seleção efêmera (query param) → Task 3 (`c.Query("user")`), sem cookie/sessão. ✔
- Seletor só p/ owner → Task 3 (`CanSelectUser=isOwner`) + Task 4 (`{{if .CanSelectUser}}`). ✔
- OWASP (não-owner ignora, owner foreign→consolidado, SQLi) → Tasks 3, 5. ✔
- Observabilidade (`viewed_user_id`) → Task 3. ✔
- Sem migration → confirmado (usa `users`+`user_tenants`). ✔
- Sincronia do indicador de escopo → Task 4 (movido p/ fragmento). ✔
