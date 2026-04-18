#!/bin/bash
set -euo pipefail

# Script para criar lead fake no playground
# Uso: ./scripts/create-playground-lead.sh "Nome" "5511999999999"

TENANT_ID="550e8400-e29b-41d4-a716-446655440001"
CONTACT_TESTE_ID="550e8400-e29b-41d4-a716-4466554400fe"

if [[ $# -lt 2 ]]; then
    echo "Uso: $0 \"Nome Lead\" \"11999999999\""
    echo ""
    echo "Exemplos:"
    echo "  $0 \"João Silva\" \"11987654321\""
    echo "  $0 \"Maria Santos\" \"11912345678\""
    exit 1
fi

NAME="$1"
PHONE="$2"
WHATSAPP_ID="${PHONE}@s.whatsapp.net"

# Gera UUIDs para novo contato
CONTACT_ID=$(python3 -c "import uuid; print(str(uuid.uuid4()))")
CONVERSATION_ID=$(python3 -c "import uuid; print(str(uuid.uuid4()))")

echo "📱 Criando lead fake..."
echo "  Nome: $NAME"
echo "  Telefone: $PHONE"
echo "  Contact ID: $CONTACT_ID"
echo "  Conversation ID: $CONVERSATION_ID"

# Cria contato
mysql -h localhost -u root -proot crm_juridico << EOF
INSERT INTO contacts (id, tenant_id, name, phone, whatsapp_id, created_at, updated_at)
VALUES ('$CONTACT_ID', '$TENANT_ID', '$NAME', '$PHONE', '$WHATSAPP_ID', NOW(), NOW())
ON DUPLICATE KEY UPDATE name = VALUES(name);
EOF

# Cria conversa
mysql -h localhost -u root -proot crm_juridico << EOF
INSERT INTO conversations (id, tenant_id, contact_id, status, last_message_at, unread_count, created_at, updated_at)
VALUES ('$CONVERSATION_ID', '$TENANT_ID', '$CONTACT_ID', 'open', NOW(), 0, NOW(), NOW())
ON DUPLICATE KEY UPDATE status = VALUES(status);
EOF

echo "✅ Lead criado com sucesso!"
echo ""
echo "🎮 Para usar no playground:"
echo "  1. Acesse http://localhost:8080/tenant/ai/playground"
echo "  2. Clique em '$NAME' na lista de contatos"
echo "  3. Digite uma mensagem e envie"
