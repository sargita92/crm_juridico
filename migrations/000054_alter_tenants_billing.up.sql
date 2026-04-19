ALTER TABLE tenants
  ADD COLUMN plano VARCHAR(20) NOT NULL DEFAULT 'mensal',
  ADD COLUMN valor_cobranca_cents BIGINT NULL,
  ADD COLUMN dia_vencimento TINYINT UNSIGNED NULL,
  ADD COLUMN data_inicio_cobranca DATE NULL,
  ADD COLUMN exibir_pagamentos BOOLEAN NOT NULL DEFAULT TRUE;
