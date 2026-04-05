# Fluxo de Entrega por Agentes

## Visão geral

Cada feature passa por um pipeline de agentes especializados. O fluxo garante qualidade, segurança e alinhamento com o produto.

## Fluxo

```text
1. PO transforma task em stories
2. UI/UX define wireframes, fluxos e protótipos de interface
3. Arquiteto faz planejamento técnico
4. QA define testes funcionais mínimos
5. Dev Backend implementa com TDD
6. Dev Front-end implementa telas e interações com HTMX
7. QA valida requisitos e execução dos testes
8. repetir ciclo Dev Backend → Dev Front-end → QA até estabilizar
9. Analista de Segurança valida
10. repetir ciclo Dev Backend → Dev Front-end → QA → Segurança até aprovação final
```

## Diagrama de ciclos

```text
                    ┌──────────────────────────────────┐
                    │                                  │
PO → UI/UX → Arq → QA → Dev Back → Dev Front → QA ──┤
                                                      │
                    ┌─────────────────────────────────┘
                    │
                    └→ Segurança ──→ Aprovado? ──→ SIM → Entrega
                         │                          
                         └── NÃO → volta para Dev Back
```

## Regras do fluxo

- nenhuma etapa pode ser pulada
- QA define testes antes da implementação (shift-left)
- Dev Backend e Dev Front-end trabalham em sequência na mesma feature
- QA valida após cada ciclo de implementação
- Segurança valida após QA aprovar
- se qualquer agente reprovar, retorna ao ciclo de correção
- prompts dos agentes estão em `docs/agentes/`

## Implementação step-by-step (obrigatório)

**Nunca implementar uma feature inteira de uma vez.** Cada feature deve ser quebrada em steps incrementais pelo Arquiteto e implementada um step por vez.

### Fluxo de steps

```text
1. Arquiteto define steps ordenados no design técnico
2. Dev Backend implementa APENAS o step atual
3. Ao final do step: testes passando, build ok, commit atômico
4. Validação do step (testes, cobertura, build)
5. Só após validação, avançar para o próximo step
6. Repetir até completar todos os steps
```

### Regras de steps

- **um step por vez**: não iniciar o próximo sem o anterior validado
- **commit por step**: cada step gera pelo menos um commit atômico
- **escopo fechado**: não misturar código de steps diferentes no mesmo commit
- **validação contínua**: ao final de cada step, todos os testes devem passar (não só os do step atual)
- **step independente**: cada step deve deixar o sistema em estado funcional (não quebrar o que já existe)
- **rollback simples**: se um step falhar, deve ser possível reverter sem afetar steps anteriores

### Exemplo de quebra em steps

```text
Feature F02 — Autenticação e Multitenancy

Step 1: Domain — entidades User, Tenant, UserTenant + interfaces de repositório
Step 2: Infrastructure — migrations + repositórios Gorm
Step 3: Application — caso de uso Login
Step 4: Application — caso de uso SelecionarTenant
Step 5: Interfaces/HTTP — handler de login + middleware de auth
Step 6: Interfaces/HTTP — handler de seleção de tenant + middleware de tenant
Step 7: Testes OWASP + arquivos .http
```

## Artefatos rastreáveis

Cada feature possui artefatos versionados em `docs/artefatos/FXX-nome/`:

| Agente | Pasta | Conteúdo |
|--------|-------|----------|
| PO | `po-stories/` | Stories com objetivo de negócio, escopo e critérios de aceite |
| UI/UX | `uiux-wireframes/` | Wireframes, fluxos de navegação, specs de interação |
| Arquiteto | `arquiteto-design/` | Entidades, use cases, contratos HTTP, migrations, composição |

### Estrutura

```text
docs/artefatos/FXX-nome-da-feature/
  status.md               # progresso da feature (substitui feature-em-andamento.md)
  po-stories/
    v1.md                  # versão inicial
    v2.md                  # revisão (se houver)
  uiux-wireframes/
    v1.md
  arquiteto-design/
    v1.md
```

### Regras de versionamento

- cada artefato começa como `v1.md`
- se houver revisão (ex: PO refina após feedback do Arquiteto), cria-se `v2.md`
- cada versão tem frontmatter com `feature`, `agent`, `version`, `created_at` e `reason`
- `status.md` é atualizado a cada mudança de etapa

## Referências

- prompts individuais: `docs/agentes/*.md`
- artefatos por feature: `docs/artefatos/FXX-*/`
- definition of done: `docs/processo/definition-of-done.md`
- backlog: `docs/processo/backlog.md`
