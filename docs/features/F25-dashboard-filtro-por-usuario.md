# F25 — Filtro de Dashboard por Usuário (Tenant)

- **Épico**: Dashboards / Gestão do Escritório
- **Prioridade**: média
- **Dependência**: F19 (dashboards), F08 (usuários e permissões)
- **Status**: em andamento
- **Iniciado**: 2026-05-26
- **Artefatos**: [`docs/artefatos/F25-dashboard-filtro-por-usuario/`](../artefatos/F25-dashboard-filtro-por-usuario/)

## Relato

Hoje o dashboard do tenant mostra ao **owner/admin** os dados **consolidados** de
todo o escritório, enquanto cada **operador** vê apenas os próprios dados
(filtrados por `responsible_user_id`). O owner não tem como olhar o desempenho de
**um operador específico** sem que o próprio operador faça login.

## Objetivo de negócio

Permitir que o owner do escritório acompanhe o desempenho individual de cada
operador (leads, conversão, WhatsApp, tempo no funil, produtos) diretamente do
seu dashboard, escolhendo o operador num seletor — sem deixar de ter a visão
consolidada como padrão.

## História de usuário

> Como **owner do escritório**, quero **selecionar um operador no dashboard**
> para **ver as métricas só daquele operador**, e poder **voltar à visão
> consolidada** quando quiser.

## Decisões de produto

- **Drill-down de um por vez**: o owner escolhe **um** operador e vê só os dados
  dele; o padrão é **Consolidado (todos)**. Sem comparação multi-usuário.
- **Lista do seletor**: apenas **operadores (não-owners)** do escritório. Owners
  e admins não aparecem na lista.
- **Seleção efêmera**: vive na URL (`?user=<id>`); recarregar a página inteira
  ou navegar para outra aba volta a "Consolidado". Sem persistência.
- **Quem vê o seletor**: apenas owner do tenant (e admin de plataforma, que já é
  tratado como owner). Operador comum continua vendo só o próprio dashboard, sem
  seletor.

## Critérios de aceite

- [ ] O owner vê, no header do dashboard, um seletor com "Consolidado (todos)" +
      a lista de operadores do escritório (ordenada por nome).
- [ ] Selecionar um operador recarrega os 5 blocos filtrados por aquele operador
      (via HTMX, sem reload da página inteira) e indica de quem é a visão.
- [ ] "Consolidado (todos)" retorna à visão de todo o escritório.
- [ ] O botão "Atualizar" preserva a seleção atual.
- [ ] Operador comum **não** vê o seletor e continua restrito aos próprios dados.
- [ ] Padrão ao abrir a página: **Consolidado**.

## Critérios de segurança (OWASP — regra 13)

- [ ] Operador comum que enviar `?user=<outro>` continua vendo **apenas o
      próprio** dashboard (parâmetro ignorado).
- [ ] Owner que enviar `?user=<id>` de **outro tenant**, de **não-membro** ou de
      um **owner** → tratado como "Consolidado" (sem vazamento; queries já são
      tenant-scoped). Tentativa registrada em log.
- [ ] Acesso não autenticado → 401 (middleware existente).
- [ ] Isolamento de tenant preservado em todas as queries.

## Critérios técnicos / DoD

- [ ] TDD: testes antes da implementação.
- [ ] Cobertura ≥ 80% nos pacotes tocados.
- [ ] Build, `go vet`, `golangci-lint` verdes.
- [ ] `rest/` atualizado com o endpoint (`?user=<id>`) e casos OWASP.
- [ ] Observabilidade: log `dashboard_rendered` com `viewed_user_id` no drill-down.
- [ ] **Sem migration nova** (usa `users` + `user_tenants` existentes).

## Fora de escopo (YAGNI)

- Comparação lado a lado de múltiplos operadores.
- Persistência da seleção (cookie/sessão).
- Filtros de período/data, exportação.
- Seletor de usuário no dashboard **admin** (esta feature é só do tenant).
