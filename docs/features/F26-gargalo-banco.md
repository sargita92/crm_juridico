# F26 — Bug: gargalo intermitente de banco (delays até ~19s)

- **Épico**: 11 — Manutenção e Qualidade Técnica
- **Prioridade**: alta
- **Dependência**: —
- **Status**: em andamento
- **Reportado**: 2026-05-25
- **Artefatos**: [`docs/artefatos/F26-gargalo-banco/`](../artefatos/F26-gargalo-banco/)

## Relato

Em alguns momentos o banco gargala e o tempo de resposta da aplicação chega a
~19s. O comportamento é **intermitente** e não tem passo de reprodução
confiável. Suspeita inicial do reporter: AI Playground (F17). **A causa-raiz não
está confirmada.**

## Objetivo de negócio

Eliminar os picos de latência percebidos pelo usuário, restaurando tempos de
resposta consistentes. Como efeito colateral, dotar o sistema da observabilidade
de banco necessária para diagnosticar e prevenir reincidências.

## Abordagem (debugging sistemático)

Por ser um bug de performance intermitente, **não se chuta a correção**. O fluxo
acordado é:

1. **Instrumentar/profilar** para tornar o gargalo visível (slow-query, pool de
   conexões, tracing por query, pprof). — *entrega atual*
2. **Observar/reproduzir** e localizar a causa-raiz com base em evidência.
3. **Corrigir** apenas a causa-raiz confirmada (em entrega subsequente).

A investigação da Fase 1 (estática) está registrada em
[`arquiteto-design/investigacao-v1.md`](../artefatos/F26-gargalo-banco/investigacao-v1.md).

## Critérios de aceite (entrega de instrumentação)

- [ ] Métricas do pool de conexões (`sql.DBStats`) expostas em `/metrics`
      (open/in-use/idle, `wait_count`, `wait_duration`).
- [ ] Slow-query log estruturado (zap) com limiar configurável e contexto
      (request_id, tenant_id, user_id).
- [ ] Tracing OTel por query (spans por operação do Gorm).
- [ ] Endpoint pprof disponível, **protegido** (flag + admin), desabilitado por
      padrão em produção.
- [ ] Painel/queries de Grafana para pool e slow-query documentados.
- [ ] Build, lint e testes verdes; cobertura ≥ 80% nos pacotes tocados.

## Fora de escopo (próxima entrega)

- A correção definitiva (ex.: tuning de pool, índice, query, statement timeout)
  só será proposta após a causa-raiz ser confirmada por evidência de runtime.
