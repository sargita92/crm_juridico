# Produtos

## Visão geral

Cada tenant pode cadastrar produtos e associá-los a leads para saber qual produto/serviço gerou o interesse do contato.

---

## Funcionalidades

### Cadastro
- CRUD de produtos no tenant

### Associação a leads
- cada lead pode ser associado a um produto
- associação pode ser automática ou manual

### Associação automática
- geralmente pela primeira mensagem já se identifica o produto
- formas possíveis de detecção automática:
  - palavras-chave na mensagem
  - especialista vinculado ao produto
  - número de WhatsApp de entrada (se o tenant usar números diferentes por produto)
- a lógica de associação deve ser simples e configurável pela interface
