# MyRoutine — Regras do Projeto

## Sobre o Projeto
MyRoutine é um sistema fullstack de gestão de vida pessoal, desenvolvido por Luiz (graduando em Ciência da Computação) para uso próprio, aprendizado e portfólio profissional.

## Perfil do Usuário
- Estudante de Ciência da Computação
- Quer aprender fazendo — NÃO escreva código completo sozinho
- Guie passo a passo, explique o porquê das decisões técnicas
- Proponha desafios e exercícios para o usuário implementar
- Quando houver erro, explique o que causou e como debugar

## Stack Técnica
- **Backend**: Go 1.26 + Chi (router) + pgx (PostgreSQL driver) + sqlc (code gen)
- **Frontend**: React 18 + Vite + TypeScript + Zustand + TanStack Query + Framer Motion
- **Banco de Dados**: PostgreSQL 16 + Redis 7
- **Infra**: Docker Compose, Nginx (reverse proxy)
- **Observabilidade**: Prometheus + Grafana
- **IA**: Claude API (Anthropic) — integração futura
- **DevSecOps**: GitLeaks, Semgrep, govulncheck, Trivy, OWASP ZAP, GitHub Actions

## Convenções de Código

### Go (Backend)
- Seguir o padrão `internal/` do Go para pacotes privados
- Handlers apenas fazem parse de request e escrevem response — lógica fica em `service/`
- Queries SQL puras (sem ORM) — usar sqlc para gerar código tipado
- Logging estruturado com zap
- Erros sempre wrapped com `fmt.Errorf("contexto: %w", err)`
- Secrets NUNCA hardcoded — sempre via variáveis de ambiente

### TypeScript (Frontend)
- CSS Modules para estilos (não Tailwind)
- Design Apple-inspired: glassmorphism, espaçamento generoso, Inter font
- Zustand para state management global
- TanStack Query para server state
- Alias `@/` aponta para `src/`

### Segurança (DevSecOps)
- JWT com refresh token rotation (nunca armazenar raw token, só hash SHA-256)
- bcrypt cost 12 para senhas
- Security headers OWASP em todo response
- Rate limiting por zona (API geral vs auth)
- Proteção contra timing attacks no login
- Audit logs para ações sensíveis

## Idioma
- Código e commits em inglês
- Comunicação com o usuário em português brasileiro
