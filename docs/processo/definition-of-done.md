# Definition of Done, Checklist e Fluxo Git

## Definition of Done — por step (obrigatória)

Cada step da feature deve ser validado **individualmente** antes de avançar:

1. testes do step passando
2. **todos** os testes anteriores continuam passando (sem regressão)
3. a aplicação compilar sem erro
4. commit atômico do step realizado
5. sistema em estado funcional (o step não quebra o que já existe)

## Definition of Done — por feature (obrigatória)

Só considerar a feature concluída quando, **simultaneamente**:

1. todos os steps implementados e validados
2. todos os testes passando (unitários, integração, OWASP)
3. a cobertura de testes estiver em pelo menos 80%
4. a aplicação compilar sem erro
5. os containers obrigatórios estiverem de pé e sem erros de runtime

Se qualquer item falhar, retornar ao ciclo de correção.

---

## Checklist operacional rápido

- [ ] regras de negócio no lugar certo (domínio/casos de uso)
- [ ] contratos HTTP consistentes
- [ ] persistência sem vazamento de infraestrutura para domínio
- [ ] migrations versionadas e aplicadas
- [ ] testes de integração com `testcontainers-go`
- [ ] cobertura de testes >= 80%
- [ ] build/compilação ok
- [ ] runtime em container sem erro
- [ ] logs e erros úteis, sem vazar dados sensíveis
- [ ] testes de segurança OWASP para endpoints da feature
- [ ] isolamento de tenant validado por teste
- [ ] arquivos `.http` em `rest/` atualizados com os novos endpoints
- [ ] testes manuais OWASP em `rest/99-seguranca-owasp.http` atualizados
- [ ] interface intuitiva e consistente (quando aplicável)
- [ ] HTMX sem JavaScript desnecessário (quando aplicável)

---

## Fluxo Git obrigatório por feature

Antes de iniciar uma feature:

1. criar branch da feature antes de alterar código
   - formato: `feature/<nome-da-feature>` (ex: `feature/F01-setup-inicial`)
2. executar implementação e validações
3. subir branch para remoto
4. abrir Pull Request

**Sem branch da feature e PR aberta, não considerar entrega concluída.**

### Convenção de commits

- usar commits descritivos e atômicos
- formato sugerido: `tipo(escopo): descrição`
  - `feat(tenant): adicionar CRUD de tenants`
  - `fix(kanban): corrigir movimentação de lead`
  - `test(auth): adicionar testes de login`
  - `docs(processo): atualizar backlog`

---

## Testes manuais (.http)

A pasta `rest/` contém arquivos `.http` (formato JetBrains HTTP Client) para testes manuais de todos os fluxos da API.

| Arquivo | Fluxo |
|---------|-------|
| `rest/http-client.env.json` | Variáveis de ambiente (dev, staging, prod) |
| `rest/00-health.http` | Healthcheck, readiness, métricas |
| `rest/01-auth.http` | Login, seleção de tenant |
| `rest/02-admin-tenants.http` | CRUD tenants, bloqueio/desbloqueio |
| `rest/03-admin-especialistas.http` | CRUD especialistas, associação com tenants |
| `rest/04-especialistas-treinamento.http` | RAG, MCPs, guardrails, steps, scoring |
| `rest/05-whatsapp.http` | Conversas, mensagens, takeover |
| `rest/06-funis-kanban.http` | Funis, colunas, leads, movimentação |
| `rest/07-usuarios-permissoes.http` | Usuários, grupos, perfis, load balance |
| `rest/08-automacoes.http` | CRUD automações (4 tipos) |
| `rest/09-produtos.http` | CRUD produtos, associação a leads |
| `rest/10-pagamentos.http` | Pagamentos, status financeiro |
| `rest/11-logs.http` | Logs, filtros, exportação |
| `rest/12-arquivos.http` | Arquivos por lead, filtros, download |
| `rest/99-seguranca-owasp.http` | Testes manuais OWASP Top 10 |

### Regra

- a cada feature entregue, os arquivos `.http` correspondentes devem ser atualizados com os novos endpoints
- novos endpoints de segurança devem ser adicionados ao `rest/99-seguranca-owasp.http`
- manter variáveis de ambiente atualizadas em `rest/http-client.env.json`

---

## Documentos operacionais

| Documento | Função |
|-----------|--------|
| `docs/processo/backlog.md` | tudo que ainda precisa ser feito |
| `docs/artefatos/FXX-*/status.md` | progresso da feature atual |
| `docs/artefatos/FXX-*/po-stories/` | stories versionadas (PO) |
| `docs/artefatos/FXX-*/uiux-wireframes/` | wireframes versionados (UI/UX) |
| `docs/artefatos/FXX-*/arquiteto-design/` | design técnico versionado (Arquiteto) |
| `docs/processo/changelog.md` | registro histórico de entregas |
