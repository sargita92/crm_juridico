# Changelog

Registro histórico de entregas do projeto.

---

## [2026-04-05] F02 — Autenticação e Multitenancy

- Entidade Tenant (PF/PJ, status, bloqueio) com repositório GORM
- Entidade User com bcrypt, relação N:N com Tenant via user_tenants
- Login com JWT (HS256, expiração configurável), cookie HttpOnly + SameSite Lax
- Middleware Auth (cookie/Bearer), middleware RequireTenant, TenantScope GORM
- Tela de login (HTMX, toggle senha, loading state, erro genérico)
- Tela de seleção de tenant (cards PF/PJ, admin vê todos)
- Dashboard placeholder
- 3 migrations reversíveis (tenants, users, user_tenants)
- 83 testes, cobertura F02 87.6%
- Segurança: 3 vulnerabilidades encontradas e corrigidas (err.Error() exposto, cookie Secure, SameSite)
- Artefatos: stories, wireframes, design técnico, cenários QA, validação QA, review segurança
