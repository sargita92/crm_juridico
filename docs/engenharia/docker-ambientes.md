# Docker e Ambientes

## Arquivos

| Arquivo | Ambiente | Descrição |
|---------|----------|-----------|
| `Dockerfile.dev` | Desenvolvimento | Com Air (hot reload) |
| `Dockerfile` | Produção | Sem dependências de desenvolvimento |
| `docker-compose.dev.yml` | Desenvolvimento | App + MySQL + volumes |
| `docker-compose.yml` | Produção | App + MySQL + healthchecks |

## Desenvolvimento

- hot reload com Air (`air` configurado no container)
- volumes para código fonte (editar local, refletir no container)
- MySQL com porta exposta para acesso local
- variáveis de ambiente via `.env` + `godotenv`
- migrations executam automaticamente no startup

## Produção

- build multi-stage (compilar + imagem mínima)
- sem dependências de desenvolvimento no container final
- healthchecks para serviços críticos (app e MySQL)
- logs estruturados (Zap) para observabilidade
- graceful shutdown configurado
- variáveis de ambiente via variáveis do ambiente de deploy (não `.env`)

## Healthchecks

- app: `GET /health` retorna 200 com status dos serviços
- MySQL: conexão de teste periódica

## Regras

- nunca incluir `.env` na imagem Docker
- manter `.dockerignore` atualizado
- separar concerns entre containers (app, banco, etc.)
- logs vão para stdout/stderr (container-friendly)
