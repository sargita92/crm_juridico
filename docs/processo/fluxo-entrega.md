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

## Referências

- prompts individuais: `docs/agentes/*.md`
- definition of done: `docs/processo/definition-of-done.md`
- backlog: `docs/processo/backlog.md`
