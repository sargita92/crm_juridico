# F21 - Saneamento Técnico (Faxina Mecânica)

## Objetivo
Zerar a dívida técnica acumulada de natureza mecânica (cobertura, lint, dependências, vulnerabilidades, código morto, TODOs órfãos) e estabelecer as automações que sustentam o processo recorrente de manutenção.

Refactors arquiteturais identificados na auditoria **não entram nesta feature** — viram features próprias (F23, F24, ...) com escopo dedicado.

## Pré-requisitos
- F01 (setup inicial — Makefile, golangci-lint configurados)

## Status: concluído (2026-05-07)

## Steps

### Step 1: Auditoria automatizada
- [ ] script `scripts/audit.sh` que executa em sequência:
  - `go test ./... -coverprofile=coverage.out` + relatório por pacote
  - `golangci-lint run ./...`
  - `go vet ./...`
  - `govulncheck ./...`
  - `go list -u -m all` (deps com versão nova disponível)
  - `grep -rn "TODO\|FIXME\|XXX" internal/ cmd/ pkg/`
- [ ] alvo `make audit` no Makefile que invoca o script e gera `audit-report.md`
- [ ] testes do próprio script (validar formato de saída)

### Step 2: Relatório inicial e priorização
- [ ] rodar `make audit` e capturar baseline em `docs/manutencao/audit-baseline-YYYY-MM-DD.md`
- [ ] listar pacotes com cobertura < 80%
- [ ] listar deps com CVE conhecida (govulncheck)
- [ ] listar deps com versão nova >= minor atrás
- [ ] listar TODOs/FIXMEs com idade > 90 dias (via `git blame`)
- [ ] **separar refactors arquiteturais identificados → abrir tickets F23/F24/... no backlog (não tratar nesta feature)**
- [ ] priorizar achados em **crítico / médio / baixo**

### Step 3: Correção de vulnerabilidades (crítico)
- [ ] aplicar updates de deps com CVE
- [ ] revalidar com `govulncheck` até zero achados
- [ ] testes existentes devem continuar passando

### Step 4: Atualização de dependências (médio)
- [ ] aplicar updates minor/patch das deps listadas no step 2
- [ ] **não** aplicar updates major (esses viram features próprias se exigirem mudança de API)
- [ ] revalidar build, testes e lint após cada update

### Step 5: Cobertura — subir pacotes < 80%
- [ ] para cada pacote abaixo do alvo, escrever testes que faltam
- [ ] meta: **todos os pacotes >= 80%**, média global >= 85%
- [ ] usar TDD — não criar testes "trapaça" só pra atingir métrica

### Step 6: Lint e vet — zerar warnings
- [ ] corrigir todos os warnings do `golangci-lint run`
- [ ] corrigir todos os warnings do `go vet`
- [ ] se algum warning for falso positivo justificável, suprimir com `//nolint:<linter> // motivo` documentado (não suprimir genérico)

### Step 7: Código morto e TODOs órfãos
- [ ] rodar `staticcheck` (subset não coberto pelo golangci-lint default) ou `deadcode`
- [ ] remover funções/tipos não referenciados
- [ ] resolver ou converter em ticket cada TODO/FIXME com idade > 90 dias
- [ ] documentação inline removida quando refere a comportamento que não existe mais

### Step 8: Gates no CI
- [ ] adicionar step no GitHub Actions: `make audit` deve passar
- [ ] gate de cobertura: PR é bloqueado se cobertura geral < 85% ou se algum pacote cair abaixo de 80%
- [ ] gate de govulncheck: PR bloqueado se introduzir vulnerabilidade
- [ ] gate de lint: PR bloqueado se houver warning novo

### Step 9: Processo recorrente
- [ ] criar `docs/processo/manutencao-tecnica.md` com a rotina trimestral (template já existirá; este step só formaliza e linka)
- [ ] atualizar `docs/processo/fluxo-entrega.md` referenciando o processo
- [ ] adicionar lembrete (issue template ou cron job interno) para abrir auditoria a cada início de trimestre

### Step 10: Relatório final
- [ ] rodar `make audit` novamente
- [ ] capturar `docs/manutencao/audit-final-YYYY-MM-DD.md`
- [ ] relatório de antes/depois (cobertura, vulns, deps, warnings, TODOs) no PR
- [ ] entrada no changelog
- [ ] atualizar backlog

## Critérios de aceite
- `make audit` passa sem achados críticos ou médios
- cobertura geral >= 85%, todos os pacotes >= 80%
- zero vulnerabilidades em `govulncheck`
- zero warnings em lint/vet
- zero TODOs/FIXMEs com idade > 90 dias
- gates de CI ativos e funcionando
- processo recorrente documentado em `docs/processo/manutencao-tecnica.md`

## Decisões técnicas

### Por que escopo "mecânico" e refactors fora
- Refactors arquiteturais (quebrar arquivo grande, extrair pacote, mudar abstração) exigem **decisão de design**. Misturar com faxina dilui escopo, esconde risco e atrasa entrega.
- Saneamento mecânico tem critério objetivo (métrica passa ou não passa). Refactor exige debate. São naturezas diferentes.
- Refactors identificados viram **F23, F24, ...** com objetivo, steps e DoD próprios. Mais rastreável.

### Updates major fora do escopo
- Update major (ex.: Gin v1 → v2) costuma exigir mudança de API. Não é "faxina" — é migração. Vira feature dedicada.

### Por que gates no CI nesta feature, e não depois
- Sem gate, a dívida volta. O processo recorrente trimestral confia nos gates para manter o estado limpo entre rodadas. Gate é parte da entrega, não follow-up.

### Cobertura: meta global vs por pacote
- Só meta global (ex.: ">=85%") permite pacotes críticos ficarem com 30% mascarados por pacotes triviais com 100%. Por isso a meta é dupla: global **e** por pacote.
