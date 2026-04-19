# F14 - Arquivos por Lead

## Objetivo
Permitir que o tenant visualize todos os arquivos enviados e recebidos nas conversas do WhatsApp, organizados por lead.

## Pré-requisitos
- F06 (integração WhatsApp)
- F07 (funis/kanban — para ter leads)

## Steps

### Step 1: Domínio de arquivos
- [x] criar entidade File (id, tenant_id, lead_id, conversation_id, message_id, nome, tipo, tamanho, path/url, direcao, created_at)
- [x] tipos: imagem, documento, áudio, vídeo, outro
- [x] migration
- [x] testes unitários do domínio

### Step 2: Captura automática de arquivos
- [x] ao receber mensagem com mídia no WhatsApp (whatsmeow), persistir arquivo
- [x] ao enviar mensagem com mídia, persistir arquivo
- [x] associar arquivo ao contato, conversa e lead automaticamente
- [x] armazenamento local (disco) ou object storage (configurável)
- [x] testes

### Step 3: Casos de uso
- [x] listar arquivos do tenant (com paginação)
- [x] listar arquivos por lead
- [x] filtrar por tipo de arquivo (imagem, documento, áudio, vídeo)
- [x] filtrar por período
- [x] buscar por nome de arquivo
- [x] download de arquivo
- [x] testes

### Step 4: Telas (HTMX)
- [x] aba dedicada de arquivos no tenant
- [x] listagem com filtros (por lead, tipo, período)
- [x] preview inline de imagens
- [x] ícones por tipo de arquivo
- [x] botão de download
- [x] indicação de qual conversa/mensagem originou o arquivo (link direto)
- [x] paginação
- [x] no detalhe do lead: seção de arquivos associados

### Step 5: Observabilidade
- [x] métrica: `files_stored_total` (counter por tipo)
- [x] métrica: `file_storage_bytes` (gauge total)
- [x] log de upload/download com tenant_id e lead_id
- [x] testes

## Critérios de aceite
- arquivos enviados e recebidos são capturados automaticamente
- listagem por lead funciona com filtros
- preview de imagens funciona
- download funciona
- link para conversa de origem funciona
- isolamento por tenant
- cobertura >= 80%
