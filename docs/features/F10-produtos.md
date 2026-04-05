# F10 - Produtos

## Objetivo
Implementar o cadastro de produtos e associação automática/manual a leads.

## Pré-requisitos
- F07 (funis/kanban)

## Steps

### Step 1: Domínio de produtos
- [ ] criar entidade Product (id, tenant_id, nome, descricao, palavras_chave, ativo, created_at)
- [ ] migration
- [ ] testes unitários

### Step 2: Casos de uso
- [ ] CRUD de produtos
- [ ] listar produtos do tenant
- [ ] associar produto a lead manualmente
- [ ] testes

### Step 3: Associação automática
- [ ] ao receber primeira mensagem, analisar conteúdo para detectar produto
- [ ] formas de detecção:
  - palavras-chave cadastradas no produto
  - especialista vinculado a produto específico
  - número de WhatsApp de entrada (se diferente por produto)
- [ ] se detectado → associar automaticamente
- [ ] se não detectado → lead fica sem produto (associação manual depois)
- [ ] testes

### Step 4: Telas (HTMX)
- [ ] template de listagem de produtos
- [ ] template de formulário de criação/edição
- [ ] campo de palavras-chave (tags)
- [ ] no detalhe do lead: seletor de produto
- [ ] filtro de leads por produto no kanban

## Critérios de aceite
- CRUD de produtos funciona
- associação automática por palavras-chave funciona
- associação manual funciona
- filtro por produto no kanban
- cobertura >= 80%
