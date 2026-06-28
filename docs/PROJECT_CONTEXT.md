# MyRoutine — Contexto Completo do Projeto

> Este arquivo documenta tudo que foi feito, decisões tomadas, o que funciona
> e o que ainda precisa ser implementado. Criado em 28/06/2026.

---

## 🎯 Visão Geral

**MyRoutine** é uma plataforma fullstack de gestão de vida pessoal com 8 módulos
planejados: Hábitos, Finanças, Saúde/Treino, Estudos, Espiritualidade, Diário,
Metas/OKRs e IA Central.

**Objetivo triplo:**
1. Uso real pessoal do Luiz
2. Projeto de portfólio profissional (Ciência da Computação)
3. Plataforma de estudo de Go, DevSecOps e IA

---

## 🏗️ Arquitetura

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  React+Vite  │────▶│   Nginx      │────▶│   Go + Chi   │
│  :5173       │     │   :80        │     │   :8080      │
└──────────────┘     └──────────────┘     └──────┬───────┘
                                                  │
                                    ┌─────────────┼─────────────┐
                                    ▼             ▼             ▼
                              ┌──────────┐ ┌──────────┐ ┌──────────┐
                              │ Postgres │ │  Redis   │ │ Claude   │
                              │  :5432   │ │  :6379   │ │   API    │
                              └──────────┘ └──────────┘ └──────────┘
```

---

## ✅ O que já foi feito (esqueleto — compila mas não foi testado end-to-end)

### Backend Go (`backend/`)

| Arquivo | Status | Descrição |
|---|---|---|
| `cmd/api/main.go` | ✅ Compila | Entry point, graceful shutdown, injeção de dependência |
| `internal/config/config.go` | ✅ Compila | Carrega env vars, valida JWT_SECRET |
| `internal/db/db.go` | ✅ Compila | Pool pgx com tuning de conexões |
| `internal/db/migrations/001_init.sql` | ✅ Criado | Schema: users, habits, habit_logs, refresh_tokens, audit_logs |
| `internal/db/queries/habits.sql` | ✅ Criado | Queries SQL para sqlc (CRUD, checkin, streak, heatmap) |
| `internal/db/queries/users.sql` | ✅ Criado | Queries SQL para sqlc (auth, tokens, audit) |
| `sqlc.yml` | ✅ Criado | Config do sqlc — MAS o `sqlc generate` nunca foi executado |
| `internal/api/router.go` | ✅ Compila | Chi router, middleware chain, todas as rotas |
| `internal/api/middleware/middleware.go` | ✅ Compila | Logger (zap), Security Headers, JWT Auth |
| `internal/api/middleware/context.go` | ✅ Compila | Helper para extrair user_id do context |
| `internal/api/handlers/health.go` | ✅ Compila | Health check com ping no PostgreSQL |
| `internal/api/handlers/auth.go` | ✅ Compila | Register, Login, Refresh, Logout |
| `internal/api/handlers/habits.go` | ✅ Compila | CRUD, CheckIn, UndoCheckIn, Logs, Stats, Heatmap |
| `internal/service/auth.go` | ✅ Compila | bcrypt, JWT, refresh token rotation, timing attack protection |
| `internal/service/habits.go` | ✅ Compila | Business logic: CRUD, streak, stats, heatmap |
| `Dockerfile` | ✅ Criado | Multi-stage build, scratch image (~5MB), non-root user |

**Dependências Go instaladas**: chi, cors, pgx, jwt, uuid, godotenv, prometheus, zap, crypto

### Frontend React (`frontend/`)

| Arquivo | Status | Descrição |
|---|---|---|
| `vite.config.ts` | ✅ Funciona | Alias @, proxy /api → backend |
| `tsconfig.app.json` | ✅ Funciona | Path aliases configurados |
| `index.html` | ✅ Funciona | SEO, lang pt-BR, meta tags |
| `src/index.css` | ✅ Funciona | Design system completo Apple-inspired (tokens, dark mode, glass) |
| `src/main.tsx` | ✅ Funciona | Entry point limpo |
| `src/App.tsx` | ✅ Type-check OK | Router + QueryClientProvider |
| `src/store/authStore.ts` | ✅ Type-check OK | Zustand com persist seletivo |
| `src/services/api.ts` | ✅ Type-check OK | Axios com interceptors (auto Bearer, auto refresh) |
| `src/types/habit.ts` | ✅ Type-check OK | Interfaces espelhando DTOs do backend |
| `src/hooks/useHabits.ts` | ✅ Type-check OK | React Query hooks (list, create, delete, checkin, stats, heatmap) |
| `src/components/layout/AppLayout.tsx` | ✅ Type-check OK | Sidebar glassmorphism com navegação |
| `src/components/layout/AppLayout.module.css` | ✅ Criado | CSS module do layout |
| `src/components/auth/ProtectedRoute.tsx` | ✅ Type-check OK | Guard de rota |
| `src/pages/LoginPage.tsx` | ✅ Type-check OK | Login com animações e error state |
| `src/pages/RegisterPage.tsx` | ✅ Type-check OK | Registro com validação |
| `src/pages/AuthPages.module.css` | ✅ Criado | Glassmorphism card, blur spheres |
| `src/pages/HabitsPage.tsx` | ✅ Type-check OK | Lista, stats, modal de criação, check-in animado |
| `src/pages/HabitsPage.module.css` | ✅ Criado | Cards, heatmap, modal, skeleton loading |

**Dependências npm instaladas**: react-router-dom, zustand, @tanstack/react-query, framer-motion, recharts, zod, axios, clsx

### Infraestrutura

| Arquivo | Status | Descrição |
|---|---|---|
| `docker-compose.yml` | ✅ Rodando | PostgreSQL 16, Redis 7, Nginx, Prometheus, Grafana |
| `infra/nginx/nginx.conf` | ✅ Criado | Security headers, rate limiting por zona, reverse proxy |
| `infra/prometheus/prometheus.yml` | ✅ Criado | Scraping config para backend, postgres, redis |
| `.env.example` | ✅ Criado | Template de todas as variáveis |
| `.env` | ✅ Criado (local) | Cópia com JWT_SECRET de desenvolvimento |

### DevSecOps

| Arquivo | Status | Descrição |
|---|---|---|
| `.github/workflows/ci.yml` | ✅ Criado | GitLeaks → Semgrep → govulncheck → npm audit → Trivy |
| `backend/Dockerfile` | ✅ Criado | Multi-stage, scratch, non-root |

### Git

- Repositório inicializado com 53 arquivos no primeiro commit
- **Repositório GitHub ainda não foi criado pelo Luiz**

---

## ❌ O que NÃO foi feito (próximos passos)

### Prioridade Alta — Fazer o MVP funcionar
1. **Rodar o backend** — nunca foi executado com `go run`, provavelmente vai ter erros de runtime
2. **Testar migration** — o schema SQL precisa rodar no PostgreSQL pela primeira vez
3. **Testar endpoints** — fazer requests reais com curl/Postman
4. **Rodar o frontend** — verificar se a UI renderiza e conecta com o backend
5. **Fluxo completo** — register → login → criar hábito → check-in → ver stats

### Prioridade Média — Completar o módulo de Hábitos
6. **Componente Heatmap** — o componente visual de consistência (GitHub-like) não foi implementado
7. **Cálculo de streak no frontend** — exibir corretamente a sequência de dias
8. **Testes unitários Go** — zero testes escritos
9. **Testes frontend** — zero testes escritos (Vitest)
10. **Edição de hábitos** — modal de edição não existe ainda
11. **Reordenação** — drag-and-drop para reordenar hábitos

### Prioridade Baixa — Completar infraestrutura
12. **sqlc generate** — gerar código Go tipado (atualmente os services usam SQL raw)
13. **Métricas customizadas** — Prometheus scraping funciona mas não tem métricas do app
14. **Dashboards Grafana** — configurar dashboards de monitoramento
15. **Pipeline CI rodando** — criar repo GitHub e ver pipeline executar de verdade
16. **Testes de segurança** — rodar Semgrep e Trivy localmente

### Futuro — Módulos novos e IA
17. **Integração Claude API** — assistente pessoal, análise de sentimento
18. **Módulo Financeiro** — receitas, despesas, categorização automática
19. **Módulo Saúde/Treino** — planos, progressão, nutrição
20. **Módulo Estudos** — pomodoro, flashcards, planos
21. **Módulo Espiritualidade** — orações, leitura, gratidão
22. **Módulo Metas/OKRs** — objetivos, breakdown, progress
23. **Módulo Diário** — entradas, análise de sentimento, busca semântica
24. **Deploy em cloud** — Railway, Render, ou VPS
25. **App Mobile** — React Native

---

## 🔑 Decisões Técnicas e Por quê

| Decisão | Motivo |
|---|---|
| Go no backend (não Node/Python) | Linguagem do ecossistema cloud/DevSecOps (Docker, K8s, Terraform). Ótimo para portfólio. |
| Chi (não Gin/Echo) | Mais idiomático, usa `net/http` padrão. Menos magia. |
| pgx (não GORM) | Driver nativo, mais performático. Aprende SQL de verdade. |
| sqlc (não ORM) | Gera Go tipado a partir de SQL — sem magia, sem N+1, sem runtime errors de query. |
| Vite (não Next.js) | Mais controle sobre bundling, sem magia de SSR. Mais educacional. |
| Zustand (não Redux) | State management simples, sem boilerplate excessivo. |
| CSS Modules (não Tailwind) | Aprender CSS de verdade, design system próprio. |
| JWT + Refresh Rotation | Padrão de segurança moderno. Refresh tokens com hash SHA-256, nunca raw. |
| Docker multi-stage scratch | Imagem final ~5MB, sem shell = mais seguro. Padrão de mercado. |
| GitHub Actions CI | Pipeline DevSecOps real com 5 ferramentas de segurança. |

---

## 📁 Estrutura de Pastas

```
myroutine/
├── .agents/AGENTS.md           ← Regras auto-descobertas pelo Antigravity
├── .env / .env.example         ← Variáveis de ambiente
├── .github/workflows/ci.yml   ← Pipeline DevSecOps
├── docker-compose.yml          ← PostgreSQL, Redis, Nginx, Prometheus, Grafana
│
├── backend/                    ← Go API
│   ├── cmd/api/main.go        ← Entry point
│   ├── internal/
│   │   ├── api/
│   │   │   ├── handlers/      ← HTTP handlers (auth, habits, health)
│   │   │   ├── middleware/    ← JWT, Logger, Security Headers
│   │   │   └── router.go     ← Chi router com todas as rotas
│   │   ├── config/            ← Env vars loader
│   │   ├── db/
│   │   │   ├── migrations/    ← SQL schemas
│   │   │   ├── queries/       ← SQL para sqlc
│   │   │   └── db.go          ← Pool pgx
│   │   └── service/           ← Business logic (auth, habits)
│   ├── Dockerfile             ← Multi-stage scratch
│   ├── go.mod / go.sum
│   └── sqlc.yml
│
├── frontend/                   ← React + Vite + TypeScript
│   ├── src/
│   │   ├── components/        ← Layout, Auth, Habits, UI
│   │   ├── hooks/             ← React Query hooks
│   │   ├── pages/             ← Login, Register, Habits
│   │   ├── services/          ← Axios API client
│   │   ├── store/             ← Zustand auth store
│   │   ├── types/             ← TypeScript interfaces
│   │   ├── index.css          ← Design system Apple-inspired
│   │   ├── App.tsx            ← Router + Query Provider
│   │   └── main.tsx           ← Entry point
│   └── vite.config.ts
│
├── infra/
│   ├── nginx/nginx.conf       ← Security headers, rate limiting
│   └── prometheus/prometheus.yml
│
└── docs/
    └── PROJECT_CONTEXT.md     ← ESTE ARQUIVO
```

---

## 🐳 Como rodar o ambiente de desenvolvimento

```bash
# 1. Subir banco e cache
docker compose up -d postgres redis

# 2. Backend (terminal 1)
cd backend
DATABASE_URL="postgres://myroutine:changeme_in_production@localhost:5432/myroutine_db?sslmode=disable" \
JWT_SECRET="dev_myroutine_super_secret_key_2026_abcdef" \
go run ./cmd/api/

# 3. Frontend (terminal 2)
cd frontend
npm run dev

# 4. Acessar
# Frontend: http://localhost:5173
# Backend:  http://localhost:8080/health
```

---

## ⚠️ Aviso Importante

O código compila (`go build ./...` e `tsc --noEmit` passam sem erros), mas
**nunca foi testado em runtime**. O primeiro `go run` provavelmente vai revelar
erros que precisam ser corrigidos — e esse é o ponto de partida do aprendizado.
