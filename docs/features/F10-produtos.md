# F10 - Produtos

## Objetivo
Implementar o cadastro de produtos e associação automática/manual a leads.

## Pré-requisitos
- F07 (funis/kanban)

## Status: concluído

## Steps

### Step 1: Domínio de produtos
- [x] criar entidade Product (id, tenant_id, nome, descricao, palavras_chave, ativo, created_at)
- [x] migration
- [x] testes unitários

### Step 2: Casos de uso
- [x] CRUD de produtos
- [x] listar produtos do tenant
- [x] associar produto a lead manualmente
- [x] testes

### Step 3: Associação automática
- [x] ao receber primeira mensagem, analisar conteúdo para detectar produto
- [x] formas de detecção:
  - [x] palavras-chave cadastradas no produto (case-insensitive)
  - [x] especialista vinculado a produto específico (entregue em F16: `SpecialistRouter` + `specialist_products`)
  - [x] número de WhatsApp de entrada (product_phone_numbers — preparado para Meta Business API)
- [x] se detectado → associar automaticamente
- [x] se não detectado → lead fica sem produto (associação manual depois)
- [x] testes

### Step 4: Telas (HTMX)
- [x] template de listagem de produtos
- [x] template de formulário de criação/edição
- [x] campo de palavras-chave (tags)
- [x] toggle ativo/inativo
- [x] no detalhe do lead: seletor de produto
- [x] filtro de leads por produto no kanban
- [x] vincular/desvincular funil com prioridade

## Decisões técnicas
- Módulo `internal/product/` com DDD + Clean Architecture
- Tabela N:N `funnel_products` com campo `priority` para ordenar funis por produto
- Interfaces cross-module: `ProductDetector` (detecção automática), `ProductProvider` (consulta por ID), `FunnelProductRouter` (roteamento de lead ao funil correto pelo produto)
- Detecção por palavras-chave: case-insensitive, busca substring nas keywords do produto
- Tabela `product_phone_numbers` preparada para futura integração Meta Business API (número de entrada por produto)
- Toggle ativo/inativo via POST (HTMX compatible, sem JS customizado)
- Seletor de produto no drawer do lead via HTMX swap

## Relação Funil ↔ Produto
Um produto pode estar vinculado a um ou mais funis. Ao criar um lead por detecção automática, o sistema usa a tabela `funnel_products` para rotear o lead ao funil correto (pelo produto detectado e prioridade configurada). A prioridade é editável via UI.

## Critérios de aceite
- [x] CRUD de produtos funciona
- [x] associação automática por palavras-chave funciona
- [x] associação manual funciona
- [x] filtro por produto no kanban
- [x] cobertura >= 80% (domain 100%, application 88.8%, http 93.7%)
