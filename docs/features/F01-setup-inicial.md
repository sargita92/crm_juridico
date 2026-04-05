# F01 - Setup Inicial do Projeto

## Objetivo
Criar a base do projeto com estrutura de pastas, Docker, banco de dados e configuração mínima para desenvolvimento.

## Pré-requisitos
- nenhum

## Steps

### Step 1: Estrutura do projeto Go
- [x] inicializar módulo Go
- [x] criar estrutura de pastas (cmd/, internal/, pkg/, web/)
- [x] configurar Gin com rota de healthcheck
- [x] configurar Zap para logging estruturado
- [x] configurar Viper para leitura de configuração
- [x] configurar godotenv para ambiente local
- [x] criar `main.go` mínima com graceful shutdown

### Step 2: Docker e banco de dados
- [x] criar `Dockerfile.dev` para desenvolvimento (com Air)
- [x] criar `Dockerfile` para produção
- [x] criar `docker-compose.dev.yml.dist` com app + MySQL
- [x] criar `docker-compose.prod.yml.dist`
- [x] criar `.env.dist` com variáveis padrão para desenvolvimento
- [x] configurar healthcheck para MySQL
- [x] configurar conexão Gorm com MySQL

### Step 3: Migrations
- [x] configurar `golang-migrate`
- [x] criar migration inicial (tabela de controle)
- [x] integrar execução de migrations no startup da aplicação

### Step 4: Testes base
- [x] configurar `testcontainers-go` para MySQL
- [x] criar helper de teste para setup/teardown do container
- [x] criar teste de integração básico (conexão com banco)
- [x] verificar cobertura mínima

### Step 5: Observabilidade base
- [x] configurar middleware Gin para métricas Prometheus (requests, latência, status)
- [x] expor endpoint `GET /metrics` (formato Prometheus)
- [x] configurar middleware de request_id (UUID por request, propagado via context)
- [x] configurar logs Zap com campos padrão (request_id, timestamp, module, duration_ms)
- [x] configurar OpenTelemetry com exporter básico
- [x] adicionar Prometheus e Grafana ao `docker-compose.dev.yml.dist`
- [x] criar dashboard Grafana básico (overview: requests/s, latência, erros)
- [x] configurar `GET /health` e `GET /ready`

### Step 6: Frontend base
- [x] configurar Go `html/template` com layouts
- [x] incluir HTMX via CDN no layout base
- [x] criar página de healthcheck visual
- [x] configurar serving de arquivos estáticos (css/js)

## Critérios de aceite
- aplicação sobe com `docker-compose up`
- healthcheck responde 200
- migrations executam sem erro
- testes passam com testcontainers
- página base renderiza com HTMX carregado
- `/metrics` expõe métricas Prometheus
- `/health` e `/ready` respondem corretamente
- Grafana acessível com dashboard básico
- logs estruturados com request_id em toda request
