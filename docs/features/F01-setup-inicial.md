# F01 - Setup Inicial do Projeto

## Objetivo
Criar a base do projeto com estrutura de pastas, Docker, banco de dados e configuração mínima para desenvolvimento.

## Pré-requisitos
- nenhum

## Steps

### Step 1: Estrutura do projeto Go
- [ ] inicializar módulo Go
- [ ] criar estrutura de pastas (cmd/, internal/, pkg/, web/)
- [ ] configurar Gin com rota de healthcheck
- [ ] configurar Zap para logging estruturado
- [ ] configurar Viper para leitura de configuração
- [ ] configurar godotenv para ambiente local
- [ ] criar `main.go` mínima com graceful shutdown

### Step 2: Docker e banco de dados
- [ ] criar `Dockerfile` para desenvolvimento (com Air)
- [ ] criar `Dockerfile` para produção
- [ ] criar `docker-compose.dev.yml` com app + MySQL
- [ ] criar `docker-compose.prod.yml`
- [ ] configurar healthcheck para MySQL
- [ ] configurar conexão Gorm com MySQL

### Step 3: Migrations
- [ ] configurar `golang-migrate`
- [ ] criar migration inicial (tabela de controle)
- [ ] integrar execução de migrations no startup da aplicação

### Step 4: Testes base
- [ ] configurar `testcontainers-go` para MySQL
- [ ] criar helper de teste para setup/teardown do container
- [ ] criar teste de integração básico (conexão com banco)
- [ ] verificar cobertura mínima

### Step 5: Observabilidade base
- [ ] configurar middleware Gin para métricas Prometheus (requests, latência, status)
- [ ] expor endpoint `GET /metrics` (formato Prometheus)
- [ ] configurar middleware de request_id (UUID por request, propagado via context)
- [ ] configurar logs Zap com campos padrão (request_id, timestamp, module, duration_ms)
- [ ] configurar OpenTelemetry com exporter básico
- [ ] adicionar Prometheus e Grafana ao `docker-compose.dev.yml`
- [ ] criar dashboard Grafana básico (overview: requests/s, latência, erros)
- [ ] configurar `GET /health` e `GET /ready`

### Step 6: Frontend base
- [ ] configurar Go `html/template` com layouts
- [ ] incluir HTMX via CDN no layout base
- [ ] criar página de healthcheck visual
- [ ] configurar serving de arquivos estáticos (css/js)

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
