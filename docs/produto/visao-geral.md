# Visão Geral do Produto

## O que é

CRM integrado com WhatsApp, voltado inicialmente para o ambiente jurídico, mas projetado para ser extensível a todas as áreas de atuação.

## Problema que resolve

Profissionais e escritórios jurídicos recebem contatos por WhatsApp sem controle estruturado. Leads se perdem, follow-ups não acontecem, e não há visibilidade do funil de vendas. Este CRM transforma conversas de WhatsApp em um pipeline de vendas visual, automatizado e com atendimento assistido por IA.

## Público-alvo

- escritórios de advocacia
- profissionais jurídicos autônomos
- extensível a qualquer área que use WhatsApp como canal de entrada

## Proposta de valor

- gestão de leads diretamente a partir de conversas do WhatsApp
- funil de vendas visual (kanban) com automações
- atendimento inicial automatizado por especialistas de IA configuráveis
- qualificação automática de leads por pontuação
- multitenancy para atender múltiplos clientes/escritórios
- interface simples, intuitiva e bonita

## Arquitetura macro

- **Multitenant**: banco de dados único, isolamento lógico por tenant
- **Duas grandes áreas**: Admin e Tenant
- **Backend**: Go (Gin, Gorm, MySQL)
- **Frontend**: HTMX + Go templates (server-side rendering)
- **IA**: especialistas configuráveis com prompt, RAG, MCPs, guardrails e steps

## Áreas do sistema

### Área Admin
| Módulo | Descrição |
|--------|-----------|
| Tenants | CRUD de tenants (PF/PJ), bloqueio/desbloqueio |
| Especialistas | CRUD de agentes de IA, treinamento pela interface |
| Associação | Vincular especialistas a tenants |
| Configuração | Perfis, grupos de permissão por tenant |
| Pagamentos | Controle automático ou manual |
| Logs | Área de logs do sistema |
| Seleção de tenant | Admin navega pela visão de qualquer tenant |

### Área do Tenant
| Módulo | Descrição |
|--------|-----------|
| WhatsApp | Conversas estilo WhatsApp Web |
| Leads/Kanban | Funil(is) de vendas ligados às conversas |
| Automações | Configuração de regras automáticas (com permissão) |
| Configurações | Ajustes da conta (com permissão) |

### Transversal
| Módulo | Descrição |
|--------|-----------|
| Usuários e Permissões | Grupos, perfis de visualização, load balance |
| Produtos | Cadastro e associação a leads |
| Landing Page | Página de apresentação (prioridade baixa) |
