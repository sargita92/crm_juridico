# Processo de Manutenção Técnica

Rotina recorrente para manter a saúde técnica do código, evitando acúmulo de dívida entre entregas. Complementa os gates de CI (que previnem regressão a cada PR) com uma revisão periódica mais ampla.

## Frequência
**Trimestral** — primeira semana de cada trimestre (jan, abr, jul, out).

Rodadas extras podem ser disparadas sob demanda quando:
- Vulnerabilidade crítica é divulgada em dependência usada
- Métrica de cobertura cai abaixo do alvo no dashboard
- Acumulam-se 5+ TODOs novos no trimestre

## Pré-requisito
- F21 concluída (script `make audit`, gates de CI e baseline existem)

## Rotina

### 1. Disparo
- Abrir issue `Manutenção Técnica YYYY-Qn` usando template em `.github/ISSUE_TEMPLATE/manutencao-trimestral.md`
- Atribuir responsável (rodízio entre devs do time)

### 2. Auditoria
```bash
make audit
```
Gera `audit-report.md` com:
- cobertura por pacote
- lint/vet warnings
- vulnerabilidades (`govulncheck`)
- deps desatualizadas
- TODOs/FIXMEs com idade > 90 dias

Salvar em `docs/manutencao/audit-YYYY-Qn.md`.

### 3. Triagem
Classificar achados em:

| Categoria | Ação |
|-----------|------|
| Vulnerabilidade crítica/alta | Corrigir nesta rodada (obrigatório) |
| Cobertura caiu abaixo de 80% em algum pacote | Corrigir nesta rodada (obrigatório) |
| Dep com update minor/patch | Aplicar nesta rodada |
| Dep com update major | Avaliar — se exige mudança de API, abrir feature dedicada (FXX) |
| Lint/vet warning novo | Corrigir nesta rodada |
| TODO órfão > 90 dias | Resolver ou converter em ticket no backlog |
| Refactor arquitetural identificado | Abrir feature dedicada (FXX), **não tratar aqui** |

### 4. Execução
- Branch `manutencao/YYYY-Qn`
- Cada categoria de correção pode virar um commit ou PR pequeno separado
- Rodar `make audit` novamente ao final
- Capturar relatório de antes/depois

### 5. Encerramento
- PR único (ou série de PRs pequenos) referenciando a issue de manutenção
- Atualizar `docs/manutencao/audit-YYYY-Qn.md` com seção "depois"
- Entrada no changelog: `chore(manutencao): rodada YYYY-Qn — N vulns corrigidas, cobertura X% → Y%`
- Fechar issue

## O que **não** faz parte deste processo

- **Refactors arquiteturais** — viram features no backlog
- **Updates major de deps** — viram features no backlog se exigirem mudança de API
- **Migração de provider/biblioteca** — feature dedicada (ex.: F20)
- **Novos linters ou ferramentas** — adoção de novo tooling é feature em si

## Sinais de que o processo está funcionando

- Cada rodada trimestral leva < 2 dias de trabalho
- Achados críticos por rodada estão decrescendo
- Cobertura geral se mantém estável >= 85%
- Sem rodadas "extras" disparadas por emergência

## Sinais de que algo está errado

- Rodada trimestral leva > 1 semana → gates de CI estão fracos, dívida está vazando entre rodadas
- Mesmos TODOs reaparecem em rodadas consecutivas → falta dono ou são ticket disfarçado de comentário
- Cobertura geral caindo entre rodadas → revisar política de PR review

## Referências
- F21 — Saneamento Técnico (one-shot inicial que estabeleceu este processo)
- `docs/engenharia/testes.md` — alvos de cobertura
- `docs/processo/fluxo-entrega.md` — onde este processo se encaixa no ciclo geral
