# Definition of Done, Checklist e Fluxo Git

## Definition of Done (obrigatória)

Só considerar uma entrega concluída quando, **simultaneamente**:

1. todos os testes necessários estiverem passando
2. a cobertura de testes estiver em pelo menos 80%
3. a aplicação compilar sem erro
4. os containers obrigatórios estiverem de pé e sem erros de runtime

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

## Documentos operacionais

| Documento | Função |
|-----------|--------|
| `docs/processo/backlog.md` | tudo que ainda precisa ser feito |
| `docs/processo/feature-em-andamento.md` | escopo da feature atual (limpar ao concluir) |
| `docs/processo/changelog.md` | registro histórico de entregas |
