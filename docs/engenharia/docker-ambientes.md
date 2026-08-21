# Docker e Ambientes

## Arquivos

| Arquivo | Ambiente | Versionado? | Descrição |
|---------|----------|-------------|-----------|
| `Dockerfile.dev` | Desenvolvimento | Sim | Com Air (hot reload) |
| `Dockerfile` | Produção | Sim | Sem dependências de desenvolvimento |
| `docker-compose.dev.yml.dist` | Desenvolvimento | Sim | Referência funcional para dev (copiar para `docker-compose.dev.yml`) |
| `docker-compose.dev.yml` | Local | **Não** (.gitignore) | Cópia local, pode ter ajustes pessoais |
| `docker-compose.prod.yml.dist` | Produção | Sim | Referência funcional para prod (copiar para `docker-compose.prod.yml`) |
| `docker-compose.prod.yml` | Local | **Não** (.gitignore) | Cópia local, pode ter ajustes pessoais |
| `.env.dist` | Desenvolvimento | Sim | Referência funcional com valores padrão para dev |
| `.env` | Local | **Não** (.gitignore) | Cópia local com credenciais reais |

### Setup inicial

```bash
cp .env.dist .env
cp docker-compose.dev.yml.dist docker-compose.dev.yml
```

> Os arquivos `.dist` devem estar **sempre funcionais** para desenvolvimento — basta copiar e rodar.

## Desenvolvimento

- hot reload com Air (`air` configurado no container)
- volumes para código fonte (editar local, refletir no container)
- MySQL com porta exposta para acesso local
- variáveis de ambiente via `.env` + `godotenv`
- migrations executam automaticamente no startup

### Script de refresh (reset completo do banco)

```bash
./scripts/refresh.sh
```

O script dropa e recria o banco, roda todas as migrations e carrega as fixtures (`fixture/fixtures.sql`) com dados iniciais para dev.

**Credenciais padrão após refresh:**
- Email: `admin@teste.com`
- Senha: `admin123`
- URL: `http://localhost:8533/login`

## Produção

- build multi-stage (compilar + imagem mínima)
- sem dependências de desenvolvimento no container final
- healthchecks para serviços críticos (app e MySQL)
- logs estruturados (Zap) para observabilidade
- graceful shutdown configurado
- variáveis de ambiente via variáveis do ambiente de deploy (não `.env`)

### Persistência (obrigatória)

Todo dado que precisa sobreviver a um deploy mora num volume nomeado. Sem isso a
atualização apaga dados silenciosamente — sem erro, como se fosse instalação nova.

| Volume | Montagem | Guarda | Se faltar |
|---|---|---|---|
| `crm-juridico-mysql-data` | `/var/lib/mysql` | banco | sobe vazio a cada deploy |
| `crm-juridico-app-storage` | `/storage` | anexos (`files/`) e sessão do WhatsApp (`whatsmeow/`) | arquivo "quebra pra abrir" e o WhatsApp pede QR de novo |

Os dois usam `name:` fixo **de propósito**. Sem `name:`, o Docker prefixa o volume
com o nome do projeto compose, que por padrão vem do nome do diretório. Um
orquestrador que publique cada deploy num diretório novo (Dokploy, por exemplo)
gera um nome de projeto novo a cada vez, o volume anterior fica órfão e o serviço
sobe do zero.

Conferir o que está em uso no host:

```bash
docker volume ls | grep crm-juridico
docker volume inspect crm-juridico-mysql-data
```

Antes de qualquer mudança de volume em produção, **fazer backup**:

```bash
docker exec <container-mysql> mysqldump -u root -p"$DATABASE_ROOT_PASSWORD" \
  --all-databases > backup-$(date +%F).sql
```

## Healthchecks

- app: `GET /health` retorna 200 com status dos serviços
- MySQL: conexão de teste periódica

## Portas

- **não usar portas padrão** (3306, 8080, 5432, etc.) para evitar conflitos com serviços locais
- usar portas aleatórias/não-convencionais (ex: `8533` para a API, `3307` para MySQL)
- portas definidas no `docker-compose.dev.yml.dist` e no `.env.dist`

## Regras

- **`.env`, `docker-compose.dev.yml` e `docker-compose.prod.yml` nunca vão para o Git** — apenas os `.dist`
- os `.dist` devem estar sempre funcionais para desenvolvimento (copiar e rodar)
- `tmp/` no `.gitignore` (usado pelo Air para hot reload)
- nunca incluir `.env` na imagem Docker
- manter `.dockerignore` atualizado
- separar concerns entre containers (app, banco, etc.)
- logs vão para stdout/stderr (container-friendly)
