# Auditoria Inicial — 2026-05-07

> Captura do estado do código no início da F21 (Saneamento Técnico).
> Comando: `make audit`. Branch: `feature/F21-saneamento-tecnico`.

## Resumo executivo

| Categoria | Estado | Meta | Gap |
|---|---|---|---|
| **Vulnerabilidades** | 14 (11 stdlib + 1 dep + 2 transitivas) | 0 | 14 |
| **Cobertura global** | ~71% (estimado, com falhas) | ≥ 85% | ~14 pp |
| **Pacotes < 80%** | 10 produtivos | 0 | 10 |
| **Lint issues** | 199 | 0 | 199 |
| **`go vet` (standalone)** | clean | clean | 0 |
| **Deps com update minor/patch** | dezenas (transitivas) | informativo | n/a |
| **TODOs/FIXMEs > 90 dias** | 0 (3 totais, todos < 90 dias) | 0 | 0 |
| **Regressão de testes** | 10 testes falhando em `internal/audit/interfaces/http` | 0 | 10 |

## 1. Vulnerabilidades (`govulncheck`)

14 vulnerabilidades reportadas com call paths ativos:

### Standard library — fix via toolchain Go 1.26.2/1.26.3
| ID | Pacote stdlib | Fixed in |
|---|---|---|
| GO-2026-4986 | net/mail | go1.26.3 |
| GO-2026-4982 | html/template | go1.26.3 |
| GO-2026-4980 | html/template | go1.26.3 |
| GO-2026-4977 | net/mail | go1.26.3 |
| GO-2026-4971 | net | go1.26.3 |
| GO-2026-4947 | crypto/x509 | go1.26.2 |
| GO-2026-4946 | crypto/x509 | go1.26.2 |
| GO-2026-4870 | crypto/tls | go1.26.2 |
| GO-2026-4869 | archive/tar | go1.26.2 |
| GO-2026-4866 | crypto/x509 | go1.26.2 |
| GO-2026-4865 | html/template | go1.26.2 |

**Plano:** atualizar `go` directive em `go.mod` para `1.26.3`.

### Dependências
| ID | Módulo | Fixed in |
|---|---|---|
| GO-2026-4918 | `golang.org/x/net@v0.52.0` | v0.53.0 |
| GO-2026-4887 | `github.com/docker/docker@v28.5.2+incompatible` | N/A (transitiva, sem call path) |
| GO-2026-4883 | `github.com/docker/docker@v28.5.2+incompatible` | N/A (transitiva, sem call path) |

**Plano:**
- `go get golang.org/x/net@latest`
- docker/docker: documentar como risco aceito — transitivo via `testcontainers-go`, não chamado do código produtivo (sem example traces no relatório `govulncheck`).

## 2. Cobertura

### Pacotes abaixo de 80% (produtivos)
| Cobertura | Pacote |
|---|---|
| 0.4% | `internal/permission/infrastructure` |
| 1.6% | `internal/notification/infrastructure` |
| 23.6% | `internal/permission/domain` |
| 33.7% | `internal/whatsapp/interfaces/http` |
| 42.2% | `internal/whatsapp/infrastructure` |
| 44.4% | `internal/shared/database` |
| 45.6% | `internal/specialist/interfaces/http` |
| 48.9% | `internal/permission/interfaces/http` |
| 76.5% | `internal/specialist/infrastructure` |
| 78.5% | `internal/tenant/interfaces/http` |

### Sem testes (não produtivos)
- `internal/shared/module` — wiring (excluído da meta)

### Pacotes com testes quebrados
- `internal/audit/interfaces/http` — 10 testes falham por regressão dos templates do redesign de UI (commit `af3b174`). Templates referenciam `partials/admin_topbar.html` mas o setup de teste só carrega `partials/sidebar.html`. Algumas asserções esperam textos antigos ("Logs admin") que viraram "Logs de auditoria".

## 3. Lint issues

199 issues distribuídas em 115 arquivos. Por linter:

| Linter | Issues |
|---|---|
| misspell | 79 |
| goimports | 57 |
| govet (golangci) | 33 |
| staticcheck | 17 |
| unused | 10 |
| errcheck | 2 |
| ineffassign | 1 |

Top arquivos (para priorização):
1. `internal/notification/interfaces/http/handler_json_test.go` (10)
2. `internal/dashboard/interfaces/http/view.go` (8)
3. `internal/funnel/application/mocks_test.go` (5)
4. `internal/tenant/interfaces/http/handler.go` (5)
5. `internal/dashboard/infrastructure/gorm_admin_stats_repo.go` (5)

## 4. `go vet`

`go vet ./cmd/... ./internal/...` clean. As 33 issues `govet` reportadas pelo golangci-lint vêm de checks adicionais (shadow, fieldalignment) que `go vet` standalone não roda por padrão.

## 5. Dependências desatualizadas

Muitas (>50) deps com updates minor/patch — em sua maioria transitivas (cloud SDKs, Azure, AWS, etc.). Nenhuma direta de produção quebrada. **Foco do Step 4:** atualizar diretas e tidy, sem perseguir transitivas.

## 6. TODOs / FIXMEs

3 ocorrências, **todas com idade < 90 dias**. Nenhuma ação requerida.

## Plano de ataque (mapeamento → Steps F21)

1. **Step 2.5** — Fix regressão de templates (load `partials/*.html`, ajustar 2-3 assertions)
2. **Step 3** — Atualizar Go 1.26.1 → 1.26.3 + `golang.org/x/net` → mais nova versão; documentar docker/docker
3. **Step 4** — `go get -u` em deps diretas + `go mod tidy`
4. **Step 5** — Subir cobertura dos 10 pacotes
5. **Step 6** — Corrigir 199 lint issues por categoria (mecânicos primeiro: misspell, goimports)
6. **Step 7** — Confirmar 0 código morto (após `unused` resolver)
7. **Step 8** — Gates CI rodando `make audit`
8. **Step 9** — `docs/processo/manutencao-tecnica.md`
9. **Step 10** — Re-rodar `make audit` e gerar `audit-final-YYYY-MM-DD.md`
