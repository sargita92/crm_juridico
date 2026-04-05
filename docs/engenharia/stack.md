# Stack Técnica

## Backend

| Tecnologia | Função |
|------------|--------|
| Go | Linguagem principal |
| Gin | Framework HTTP |
| Gorm | ORM / persistência |
| MySQL | Banco de dados principal |
| `golang-migrate` | Migrations versionadas |
| `testcontainers-go` | Testes de integração com containers |
| Zap | Logging estruturado |
| `godotenv` | Variáveis de ambiente local |
| Viper | Configuração centralizada |
| `whatsmeow` | Cliente WhatsApp (desenvolvimento, até integração com Meta) |

## Observabilidade

| Tecnologia | Função |
|------------|--------|
| Zap | Logging estruturado |
| Prometheus | Métricas da aplicação |
| OpenTelemetry | Tracing distribuído |
| Grafana | Dashboards (logs, métricas, traces) |

## Frontend

| Tecnologia | Função |
|------------|--------|
| HTMX | Interatividade frontend sem SPA |
| Go `html/template` | Renderização server-side |
| CSS | Estilização (sem framework CSS definido ainda) |

## Infraestrutura

| Tecnologia | Função |
|------------|--------|
| Docker | Containerização |
| Air | Hot reload em desenvolvimento |

## Decisões técnicas

### Por que HTMX e não SPA?

- simplicidade: sem build frontend separado
- Go templates + HTMX cobrem todas as interações necessárias
- menos complexidade operacional (um único deploy)
- HTMX resolve: navegação parcial, formulários dinâmicos, atualizações em tempo real

### Por que Gin?

- performático e minimalista
- middleware ecosystem maduro
- amplamente adotado na comunidade Go

### Por que Gorm?

- ORM mais popular em Go
- suporte a migrations (usado apenas em dev, não em prod)
- scoping fácil para multitenancy

### Por que MySQL?

- maturidade e confiabilidade
- bom suporte a multitenancy com banco único
- ferramentas de administração amplamente disponíveis

### Por que whatsmeow?

- gratuito (sem custo de API durante desenvolvimento)
- permite testar fluxos completos antes de investir na integração com Meta
- biblioteca Go nativa (sem dependência de serviço externo)
- a interface de provider abstrai a implementação — trocar para Meta Business API no futuro não impacta o domínio
- **importante**: whatsmeow é para desenvolvimento/testes; produção migrará para WhatsApp Business API (Meta)

### Observabilidade

- Zap para logs estruturados (já na stack base)
- Prometheus para métricas (latência, throughput, erros, filas)
- OpenTelemetry para tracing distribuído (rastrear request end-to-end)
- Grafana para dashboards unificados (logs, métricas, traces)
- objetivo: visibilidade completa para diagnosticar problemas rapidamente
