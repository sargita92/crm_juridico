# F24 — Notas do cliente na tela de WhatsApp (design)

**Data:** 2026-05-25
**Status:** design aprovado
**Branch:** `feature/F24-notas-no-whatsapp`

## Problema

Hoje as notas sobre o cliente (`LeadNote`) só são acessíveis pelo drawer do Kanban
(aba Leads). Quem está atendendo na tela de WhatsApp não consegue ver nem registrar
notas sem sair do chat. Queremos acesso às notas direto na tela de WhatsApp.

## Decisões (brainstorming)

- **Escopo:** ver + adicionar notas no chat (mesma capacidade do Kanban hoje — que só
  cria e lista, sem editar/excluir). Editar/excluir fica fora de escopo.
- **Layout:** drawer deslizante. Botão "📝 Notas" no cabeçalho do chat abre um painel
  que desliza por cima da direita; fica escondido até ser acionado.
- **Cross-sell (vários leads por conversa):** mostrar e criar notas no **lead atual da
  conversa** = o lead mais recente por `created_at`. Em cross-sell, o lead de origem
  recebe `outcome = cross_sell` e o destino nasce `em_andamento` mais novo, então o
  "mais recente" é sempre o lead em que a conversa está no momento.
- **Abordagem:** porta no `whatsapp/domain` implementada por um adapter no `funnel`,
  espelhando o padrão já existente do `LeadCreator`. Reusa entidade, tabela e use case
  de criação de notas. Sem migration nova.

## Arquitetura

As notas pertencem ao módulo `funnel` (são por lead); a tela pertence ao `whatsapp`.
O `funnel` já importa `whatsapp/domain` (ex.: `whatsapp_message_adapter.go`) e o
`whatsapp` **não** importa `funnel` — então o adapter mora no `funnel` sem ciclo.

Wiring atual de referência: `whatsappMod.SetLeadCreator(funnelMod.LeadCreator())`
(`cmd/api/main.go:133`).

### Porta (módulo whatsapp)

```go
// internal/whatsapp/domain/lead_notes_service.go
type ConversationNote struct {
    ID         string
    Content    string
    AuthorName string
    CreatedAt  time.Time
}

type LeadNotesService interface {
    // Resolve o lead atual da conversa e lista as notas dele.
    // hasLead=false quando não há lead para a conversa naquele tenant.
    NotesForConversation(ctx context.Context, tenantID, conversationID string) (hasLead bool, notes []ConversationNote, err error)
    // Resolve o lead atual e cria a nota nele; devolve a lista atualizada.
    AddNote(ctx context.Context, tenantID, conversationID, content, createdBy string) (notes []ConversationNote, err error)
}
```

### Adapter (módulo funnel)

`internal/funnel/infrastructure/whatsapp_notes_adapter.go` implementa
`whatsappdomain.LeadNotesService`. Dependências (todas já existem): `LeadRepository`,
`LeadNoteRepository`, `UserNameProvider`, `*application.CreateLeadNoteUseCase`.

- `NotesForConversation`: resolve lead atual via `FindCurrentByConversationID`; se
  `ErrLeadNotFound` → `hasLead=false`, lista vazia, sem erro. Senão lista
  `noteRepo.FindByLeadID` e enriquece `AuthorName` com `userNameProvider.FindNameByID`
  (mesma lógica de `get_lead_detail.go:198`).
- `AddNote`: resolve lead atual; se não houver → erro de domínio
  (`ErrLeadNotFound`/sentinela amigável). Senão chama `CreateLeadNoteUseCase.Execute`
  e retorna a lista atualizada (reaproveita a leitura).

### Query nova (módulo funnel)

```go
// domain/repository.go (+1 método em LeadRepository)
FindCurrentByConversationID(ctx context.Context, tenantID, conversationID string) (*Lead, error)
// gorm: WHERE conversation_id = ? AND tenant_id = ? ORDER BY created_at DESC LIMIT 1
// ErrLeadNotFound quando não há linha.
```

O `FindByConversationID` atual (usado pela IA) permanece **intocado**. O novo método
inclui `tenant_id` por isolamento — é o que garante a segurança de tenant do recurso.

### funnel module

Expõe `func (m *Module) LeadNotesService() whatsappdomain.LeadNotesService`.

### whatsapp module

- Handler guarda `notesService domain.LeadNotesService`.
- `func (m *Module) SetLeadNotesService(s domain.LeadNotesService)` (igual `SetLeadCreator`).
- `cmd/api/main.go`: `whatsappMod.SetLeadNotesService(funnelMod.LeadNotesService())`.

## Rotas e handlers (módulo whatsapp)

Grupo existente `/tenant/whatsapp` com `authMw` + `tenantMw`:

```
GET  /tenant/whatsapp/conversations/:id/notes   → RenderNotesPanel
POST /tenant/whatsapp/conversations/:id/notes   → HandleCreateNote
```

- `RenderNotesPanel`: `tenantID` (middleware) + `:id` → `NotesForConversation` →
  renderiza `whatsapp/notes_panel.html` com `{ConversationID, HasLead, Notes}`.
- `HandleCreateNote`: `tenantID` + `:id` + `content` (form) + `user_id` (contexto) →
  `AddNote` → re-renderiza o partial com a lista atualizada. Erro de validação
  (vazio / >2000 chars) volta como mensagem dentro do partial, preservando o texto.

## Frontend (HTMX-first)

Cabeçalho do chat (`web/templates/whatsapp/chat.html`) ganha o botão:

```html
<button class="wa-notes-btn"
        hx-get="/tenant/whatsapp/conversations/{{.ConversationID}}/notes"
        hx-target="#wa-notes-drawer-body"
        hx-on:click="document.getElementById('wa-notes-drawer').classList.add('open')">
  📝 Notas
</button>
```

Drawer dentro de `.wa-chat-container`:

```html
<aside id="wa-notes-drawer" class="wa-notes-drawer">
  <div id="wa-notes-drawer-body"><!-- notes_panel.html --></div>
</aside>
```

Partial `web/templates/whatsapp/notes_panel.html`: cabeçalho "📝 Notas do cliente" +
botão ✕ (fecha removendo a classe `.open`); se `HasLead=false` → estado vazio sem form;
senão → form (textarea + enviar, `hx-post` no próprio drawer body, `hx-swap=innerHTML`)
+ lista de notas (autor · data · conteúdo) no mesmo estilo amarelo do Kanban.

Abrir/fechar é só toggle de classe CSS via `hx-on:click` (atributo do HTMX, sem JS
custom). CSS novo em `web/static/css/whatsapp.css`: `.wa-notes-drawer` (absolute à
direita, `translateX(100%)` + transition), `.wa-notes-drawer.open` (`translateX(0)`),
`.wa-notes-btn`, itens de nota; `.wa-chat-container { position: relative }`.

## Edge cases / erros

- Conversa sem lead → estado vazio, sem form.
- `AddNote` sem lead (defensivo) → mensagem amigável no partial.
- Conteúdo vazio / >2000 chars → reusa `ErrNoteContentRequired` / `ErrNoteContentTooLong`.
- `createdBy` = `user_id` do contexto (igual Kanban).
- Conversa de outro tenant → query filtra por `tenant_id` → "sem lead", sem vazamento.

## Segurança (OWASP — regra 13)

Testes em ambas as rotas:
- 401/403 sem auth / sem tenant.
- Isolamento de tenant: usuário do tenant A não lê nem cria nota em conversa/lead do
  tenant B.
- Anti-injection: payload XSS no conteúdo renderiza escapado pelo `html/template`.

## Testes (TDD, cobertura ≥80%)

- Repo `FindCurrentByConversationID`: vários leads na mesma conversa → retorna o mais
  recente; filtro de tenant; not found. (gorm + testcontainers, padrão
  `gorm_repositories_test.go`).
- Adapter: `NotesForConversation` e `AddNote` com mocks (lead achado/não achado,
  enriquecimento de autor, erro de validação, sem lead).
- Handlers: `RenderNotesPanel` + `HandleCreateNote` (sucesso, vazio, sem lead) + bloco
  OWASP (padrão `owasp_test.go`).
- Teste primeiro (red) → implementação (green).

## Observabilidade (regra 11)

Logs com `request_id`/`tenant_id`/`user_id`; métricas nas 2 rotas (padrão dos handlers
do whatsapp); span de trace no adapter (reusa o span `funnel.usecase.create_lead_note`
já existente no create).

## Entregáveis

- `rest/` atualizado com os 2 endpoints (regra 14).
- Branch `feature/F24-notas-no-whatsapp` + PR.

## Fora de escopo

- Editar/excluir notas (não existe nem no Kanban hoje).
- Notas a nível de Contato (refator do modelo).
- Seletor manual de lead na conversa.
- Atualização em tempo real do painel (SSE) — carrega ao abrir e após criar.
