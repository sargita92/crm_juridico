# F14 - Arquivos por Lead

## Objetivo
Permitir que o tenant visualize todos os arquivos enviados e recebidos nas conversas do WhatsApp, organizados por lead.

## Pré-requisitos
- F06 (integração WhatsApp)
- F07 (funis/kanban — para ter leads)

## Steps

### Step 1: Domínio de arquivos
- [ ] criar entidade File (id, tenant_id, lead_id, conversation_id, message_id, nome, tipo, tamanho, path/url, direcao, created_at)
- [ ] tipos: imagem, documento, áudio, vídeo, outro
- [ ] migration
- [ ] testes unitários do domínio

### Step 2: Captura automática de arquivos
- [ ] ao receber mensagem com mídia no WhatsApp (whatsmeow), persistir arquivo
- [ ] ao enviar mensagem com mídia, persistir arquivo
- [ ] associar arquivo ao contato, conversa e lead automaticamente
- [ ] armazenamento local (disco) ou object storage (configurável)
- [ ] testes

### Step 3: Casos de uso
- [ ] listar arquivos do tenant (com paginação)
- [ ] listar arquivos por lead
- [ ] filtrar por tipo de arquivo (imagem, documento, áudio, vídeo)
- [ ] filtrar por período
- [ ] buscar por nome de arquivo
- [ ] download de arquivo
- [ ] testes

### Step 4: Telas (HTMX)
- [ ] aba dedicada de arquivos no tenant
- [ ] listagem com filtros (por lead, tipo, período)
- [ ] preview inline de imagens
- [ ] ícones por tipo de arquivo
- [ ] botão de download
- [ ] indicação de qual conversa/mensagem originou o arquivo (link direto)
- [ ] paginação
- [ ] no detalhe do lead: seção de arquivos associados

### Step 5: Observabilidade
- [ ] métrica: `files_stored_total` (counter por tipo)
- [ ] métrica: `file_storage_bytes` (gauge total)
- [ ] log de upload/download com tenant_id e lead_id
- [ ] testes

## Critérios de aceite
- arquivos enviados e recebidos são capturados automaticamente
- listagem por lead funciona com filtros
- preview de imagens funciona
- download funciona
- link para conversa de origem funciona
- isolamento por tenant
- cobertura >= 80%
