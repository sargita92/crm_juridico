#!/bin/bash
set -euo pipefail

# Script para testar o playground via HTTP
# Uso: ./scripts/test-playground.sh [comando]

HOST="${HOST:-http://localhost:8080}"
CONTACT_ID="550e8400-e29b-41d4-a716-4466554400fe"  # Contato de teste padrão

# Funções de teste
test_page() {
    echo "🔍 Testando acesso à página do playground..."
    curl -s -w "\nStatus: %{http_code}\n" \
        "$HOST/tenant/ai/playground" \
        -H "Cookie: token=$TOKEN" | head -20
}

test_conversation() {
    echo "🔍 Testando carregamento da conversa..."
    curl -s -w "\nStatus: %{http_code}\n" \
        "$HOST/tenant/ai/playground/conversation/$CONTACT_ID" \
        -H "Cookie: token=$TOKEN" | head -30
}

test_send() {
    local msg="${1:-Teste message}"
    echo "🔍 Testando envio de mensagem: '$msg'"

    response=$(curl -s -w "\n%{http_code}" \
        -X POST \
        "$HOST/tenant/ai/playground/conversation/$CONTACT_ID/send" \
        -H "Cookie: token=$TOKEN" \
        -d "content=$msg")

    # Extract status code (última linha)
    status=$(echo "$response" | tail -1)
    body=$(echo "$response" | sed '$d')

    echo "Status: $status"
    if [[ -n "$body" ]]; then
        echo "Response: $body"
    fi

    if [[ "$status" == "204" ]]; then
        echo "✅ Envio bem-sucedido! (204 No Content)"
    else
        echo "❌ Erro: Status $status"
        return 1
    fi
}

test_reset() {
    echo "🔍 Testando reset da conversa..."

    response=$(curl -s -w "\n%{http_code}" \
        -X POST \
        "$HOST/tenant/ai/playground/conversation/$CONTACT_ID/reset" \
        -H "Cookie: token=$TOKEN")

    status=$(echo "$response" | tail -1)

    echo "Status: $status"
    if [[ "$status" == "204" ]]; then
        echo "✅ Reset bem-sucedido!"
    else
        echo "❌ Erro: Status $status"
    fi
}

# Valida TOKEN
if [[ -z "${TOKEN:-}" ]]; then
    echo "❌ Erro: TOKEN não definida"
    echo ""
    echo "Uso: TOKEN=seu-token ./scripts/test-playground.sh [comando]"
    echo ""
    echo "Comandos:"
    echo "  page          - Testa acesso à página"
    echo "  conversation  - Testa carregamento da conversa"
    echo "  send [msg]    - Testa envio de mensagem"
    echo "  reset         - Testa reset"
    echo "  all           - Executa todos os testes"
    echo ""
    echo "Exemplo:"
    echo "  TOKEN=abc123 ./scripts/test-playground.sh send 'Olá!'"
    exit 1
fi

# Executa comando
cmd="${1:-all}"
case "$cmd" in
    page)
        test_page
        ;;
    conversation)
        test_conversation
        ;;
    send)
        test_send "${2:-Test message from script}"
        ;;
    reset)
        test_reset
        ;;
    all)
        echo "🧪 Executando suite de testes..."
        echo ""
        test_page && echo "" && \
        test_conversation && echo "" && \
        test_send "Teste 1" && echo "" && \
        test_send "Teste 2" && echo "" && \
        test_reset
        echo ""
        echo "✅ Todos os testes completados!"
        ;;
    *)
        echo "❌ Comando desconhecido: $cmd"
        exit 1
        ;;
esac
