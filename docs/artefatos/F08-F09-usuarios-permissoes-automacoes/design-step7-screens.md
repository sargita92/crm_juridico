# F09 Step 7 — Telas de Automações (HTMX)

## Contexto

O backend de automações (F09 Steps 1-6) está completo: domínio, 7 tipos de executor, motor de execução, API REST com CRUD + toggle + logs. Falta o frontend HTMX para que o usuário gerencie automações pela interface.

## Decisões de design

| Decisão | Escolha |
|---------|---------|
| Navegação | Item próprio "Automações" no sidebar do tenant |
| Layout | Tabela (padrão do projeto) com dropdown de funil no topo |
| Formulário | Modal com campos dinâmicos por tipo via HTMX |
| Logs | Botão na tabela, abre modal com execuções paginadas |
| Toggle | Clique inline no badge ativo/inativo direto na tabela |
| Campos dinâmicos | Backend retorna partial HTML por tipo (HTMX-first) |

## Navegação

- Novo item **"Automações"** no sidebar do tenant (`partials/tenant_sidebar.html`), posicionado após "Leads"
- Ícone: raio/lightning bolt (consistente com o ícone ⚡ usado nos docs)
- Rota: `GET /tenant/automations`
- Visível apenas para quem tem permissão `automations:manage`

## Página principal (listagem)

### Estrutura

- Layout tenant padrão (sidebar + conteúdo)
- Header: título "Automações"
- Filtros: dropdown de funis (carregado do backend) + botão "+ Nova Automação"
- Tabela dentro de `div#automations-table` (target HTMX)

### Tabela

| Coluna | Conteúdo |
|--------|----------|
| Tipo | Ícone + nome legível (ex: "⏰ Exclusão por tempo") |
| Coluna | Nome da coluna do funil (ou "—" para rate_limit) |
| Configuração | Resumo legível da config (ex: "Arquivar após 48h") |
| Prioridade | Número inteiro |
| Status | Badge clicável "Ativo" (verde) ou "Inativo" (vermelho) |
| Ações | Editar (✏️), Logs (📊), Excluir (🗑️ com hx-confirm) |

### Interações HTMX

- Trocar funil: `hx-get="/tenant/automations/table?funnel_id=X"` → `hx-target="#automations-table"`
- Toggle status: `hx-post="/tenant/leads/automations/:id/toggle"` → `hx-target="#automations-table"`
- Excluir: `hx-delete="/tenant/leads/automations/:id"` → `hx-target="#automations-table"` + `hx-confirm="Excluir esta automação?"`

### Empty state

Quando o funil não tem automações: ícone + "Nenhuma automação configurada para este funil" + botão "Criar primeira automação".

## Modal de criação/edição

### Campos fixos (sempre visíveis)

- **Coluna** — select com colunas do funil selecionado (não aparece para rate_limit)
- **Tipo** — select com os 7 tipos disponíveis
- **Prioridade** — input number (default: próximo número)

### Campos dinâmicos (mudam por tipo via HTMX)

Ao trocar o select de Tipo: `hx-get="/tenant/automations/fields?type=X&funnel_id=Y"` → `hx-target="#dynamic-fields"`

Campos por tipo:

| Tipo | Campos |
|------|--------|
| `expiration` | Ação (select: arquivar / excluir) + Duração em horas (number) |
| `move_funnel` | Funil destino (select) + Coluna destino (select, carrega ao trocar funil) |
| `auto_message` | Template (textarea, monospace) + dica de variáveis disponíveis |
| `auto_note` | Template da nota (textarea) |
| `switch_specialist` | Especialista (select, carregado do backend) |
| `rate_limit` | Limite de mensagens (number) + Período em horas (number) |
| `detect_product` | Checkbox "Trocar especialista automaticamente" |

### Submit

- Criar: `hx-post="/tenant/leads/funnels/:funnel_id/automations"`
- Editar: `hx-put="/tenant/leads/automations/:id"`
- Ambos: `hx-target="#automations-table"` + fecha modal via evento `htmx:afterSwap`
- Loading state: botão desabilitado + spinner (padrão do projeto com `hx-disabled-elt`)

### Edição

Ao clicar ✏️: `hx-get="/tenant/automations/:id/form"` carrega o modal com dados preenchidos.

## Modal de logs

- Aberto pelo botão 📊 na tabela
- `hx-get="/tenant/leads/automations/:id/logs?limit=20&offset=0"` carrega conteúdo
- Tabela: Data (formatada), Lead ID, Status (badge success/error), Mensagem de erro
- Paginação: botões "Anterior" / "Próximo" via HTMX com offset

## Templates

```
web/templates/
  automation/
    list.html              — página principal (extends tenant layout)
    table.html             — fragmento da tabela (recarregável via HTMX)
    modal_form.html        — modal de criar/editar
    modal_logs.html        — modal de logs de execução
    fields/
      expiration.html      — campos para exclusão por tempo
      move_funnel.html     — campos para mover funil
      auto_message.html    — campos para mensagem automática
      auto_note.html       — campos para anotação automática
      switch_specialist.html — campos para trocar especialista
      rate_limit.html      — campos para rate limit
      detect_product.html  — campos para detectar produto
```

## Rotas HTML (novas)

| Método | Rota | Retorna | Descrição |
|--------|------|---------|-----------|
| GET | `/tenant/automations` | HTML (página completa) | Página principal |
| GET | `/tenant/automations/table` | HTML (fragmento) | Tabela filtrada por funnel_id |
| GET | `/tenant/automations/fields` | HTML (fragmento) | Campos dinâmicos por tipo |
| GET | `/tenant/automations/:id/form` | HTML (fragmento) | Modal preenchido para edição |

As rotas de API existentes (CRUD + toggle + logs) continuam retornando JSON. Os novos handlers HTML consomem o `CRUDUseCase` e renderizam templates.

## Mapeamento tipo → label/ícone

| Type (backend) | Label (frontend) | Ícone |
|----------------|-------------------|-------|
| `expiration` | Exclusão por tempo | ⏰ |
| `move_funnel` | Mover para funil | 🔀 |
| `auto_message` | Mensagem automática | 💬 |
| `auto_note` | Anotação automática | 📋 |
| `switch_specialist` | Trocar especialista | 👤 |
| `rate_limit` | Limite de mensagens | 🚫 |
| `detect_product` | Detectar produto | 🔍 |

## Permissionamento

- Todas as rotas protegidas por `authMw` + `tenantMw` + `requirePerm("automations", "manage")`
- Item no sidebar só renderiza se o usuário tem a permissão
- Consistente com as rotas de API existentes

## Fora de escopo

- CSS novo (usar classes existentes de `main.css`)
- JavaScript customizado além do mínimo para modal (reutilizar `openModal`/`closeModal` de `admin.js`)
- Reordenação de prioridade por drag-and-drop (pode ser adicionado depois)
