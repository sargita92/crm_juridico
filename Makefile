.PHONY: help test test-short build vet lint cover audit audit-vuln audit-lint audit-vet audit-cover audit-deps audit-todos tools

GO          ?= go
GOBIN       ?= $(shell go env GOPATH)/bin
GOLANGCI    ?= $(GOBIN)/golangci-lint
GOVULNCHECK ?= $(GOBIN)/govulncheck

# Padrões fonte (evita walk em storage/ que pode ter dirs root-owned do Docker)
SRC_PATTERNS := ./cmd/... ./internal/...

# Pacotes não considerados na meta de cobertura (wiring/main)
COVER_EXCLUDE := github.com/sasrgita/crm-juridico/cmd/... github.com/sasrgita/crm-juridico/internal/shared/module/...

help:
	@echo "Targets:"
	@echo "  test           Roda toda a suite de testes"
	@echo "  test-short     Roda apenas testes -short"
	@echo "  build          Compila a aplicacao"
	@echo "  vet            Roda go vet"
	@echo "  lint           Roda golangci-lint"
	@echo "  cover          Roda testes com cobertura"
	@echo "  audit          Roda toda a auditoria mecanica (vuln, lint, vet, cover, deps, todos)"
	@echo "  tools          Instala ferramentas necessarias (govulncheck)"

tools:
	$(GO) install golang.org/x/vuln/cmd/govulncheck@latest

test:
	$(GO) test $(SRC_PATTERNS) -count=1

test-short:
	$(GO) test $(SRC_PATTERNS) -short -count=1

build:
	$(GO) build $(SRC_PATTERNS)

vet:
	$(GO) vet $(SRC_PATTERNS)

lint:
	$(GOLANGCI) run $(SRC_PATTERNS)

cover:
	$(GO) test -coverprofile=coverage.out -covermode=atomic $(SRC_PATTERNS)

audit-vuln:
	@echo "[audit] vulnerabilities (govulncheck)"
	@$(GOVULNCHECK) $(SRC_PATTERNS) || (echo "[audit] FAIL: vulnerabilities found" && exit 1)
	@echo "[audit] OK: no vulnerabilities"

audit-lint:
	@echo "[audit] lint (golangci-lint)"
	@$(GOLANGCI) run $(SRC_PATTERNS) || (echo "[audit] FAIL: lint issues" && exit 1)
	@echo "[audit] OK: lint clean"

audit-vet:
	@echo "[audit] vet (go vet)"
	@$(GO) vet $(SRC_PATTERNS) || (echo "[audit] FAIL: vet issues" && exit 1)
	@echo "[audit] OK: vet clean"

audit-cover:
	@echo "[audit] coverage (>=80% per package, >=85% global)"
	@bash scripts/audit-cover.sh

audit-deps:
	@echo "[audit] outdated deps (minor/patch)"
	@bash scripts/audit-deps.sh

audit-todos:
	@echo "[audit] TODO/FIXME age"
	@bash scripts/audit-todos.sh

audit: audit-vuln audit-vet audit-lint audit-cover audit-deps audit-todos
	@echo "[audit] ALL CHECKS PASSED"
