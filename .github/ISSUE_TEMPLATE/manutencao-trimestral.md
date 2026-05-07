---
name: Manutenção Trimestral
about: Rodada trimestral de saneamento técnico (cobertura, lint, vulns, deps)
title: 'Manutenção Técnica YYYY-Qn'
labels: ['manutencao', 'tech-debt']
assignees: ''
---

## Responsável
@<usuário>

## Pré-rodada
- [ ] Branch `manutencao/YYYY-Qn` criada
- [ ] `make audit` executado e capturado em `docs/manutencao/audit-YYYY-MM-DD.md` (estado **inicial**)

## Triagem (rodada atual)
- [ ] Vulnerabilidades críticas/altas corrigidas (`govulncheck` zerado, exceto IDs aceitas em `.audit-accepted-vulns.txt`)
- [ ] Pacotes com cobertura < 80% corrigidos
- [ ] Cobertura global ≥ 85%
- [ ] Updates minor/patch de deps diretas aplicados
- [ ] Lint/vet warnings zerados
- [ ] TODOs/FIXMEs com idade > 90 dias resolvidos ou convertidos em ticket
- [ ] Updates major identificados → abrir feature dedicada (citar abaixo)
- [ ] Refactors arquiteturais identificados → abrir feature dedicada (citar abaixo)

## Encerramento
- [ ] `make audit` re-executado e capturado em `docs/manutencao/audit-YYYY-MM-DD.md` (estado **final**)
- [ ] Diff antes/depois documentado
- [ ] Entrada em `docs/processo/changelog.md`
- [ ] PR mergeado

## Itens postergados (se houver)
- (lista de features novas abertas para tratar refactors/major upgrades)

## Referência
- Processo: [docs/processo/manutencao-tecnica.md](../../docs/processo/manutencao-tecnica.md)
- F21: feature one-shot que estabeleceu este processo
