# Banco de Dados e Migrations

## Banco principal

- MySQL como banco padrão
- banco único para todos os tenants (multitenancy com isolamento lógico)
- toda entidade relevante deve ter coluna `tenant_id`

## Migrations

- evoluir schema com `golang-migrate`
- migrations versionadas em `migrations/`
- formato: `XXXXXX_descricao.up.sql` / `XXXXXX_descricao.down.sql`
- evitar `AutoMigrate` do Gorm como estratégia principal de produção
- executar migrations no startup da aplicação (em container)
- toda migration deve ter seu `down` correspondente (reversível)

## Regras

- nunca alterar uma migration já aplicada em produção (criar uma nova)
- testar migrations de ida e volta localmente antes de subir
- manter migrations atômicas (uma responsabilidade por migration)
- nomear migrations de forma descritiva (`000001_create_tenants_table`)

## Ambiente de desenvolvimento

- MySQL sobe via `docker-compose.dev.yml` (cópia local do `docker-compose.dev.yml.dist`)
- migrations executam automaticamente no startup
- seed de dados de desenvolvimento (quando aplicável)

## Ambiente de produção

- MySQL em container ou serviço gerenciado
- migrations executam no deploy (antes do app subir)
- backup antes de migrations destrutivas
