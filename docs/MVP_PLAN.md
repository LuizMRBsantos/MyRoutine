# MyRoutine — Plano do beta fechado (MVP)

> Documento vivo. Última atualização: 29/09/2026.
> Substitui o roteiro de `docs/PROJECT_CONTEXT.md` (de 28/06, desatualizado).

## Objetivo

Deixar o MyRoutine pronto para **até 10 convidados** (amigos e família) usarem de
verdade, em **web, iPhone e Mac**. Nos dois últimos, ele roda como **PWA**, o site
instalado pela tela inicial ou pelo dock, sem App Store. A hospedagem é na **AWS**.

**Os convites só saem depois da Etapa 6.** Até lá, IA e notificações já precisam
estar prontas.

## Etapas

| # | Etapa | Status |
|---|---|---|
| 0 | Consertar o que quebra com várias pessoas | ✅ concluída |
| 1 | Convites e conta | 🔄 2 de 4 peças |
| 2 | App no celular e no Mac (PWA, layout responsivo, Track Day web, revisão semanal) | ⏳ |
| 3 | IA (Claude): assistente e insights sob demanda | ⏳ |
| 4 | Notificações (Web Push): uma consolidada por horário, no fuso de cada pessoa | ⏳ |
| 5 | Colocar no ar na AWS. **Luiz faz, com o Claude ensinando passo a passo** | ⏳ |
| 6 | Revisão final, teste como convidado e envio dos convites | ⏳ |

### Etapa 0: concluída (27–29/09)

- **Fuso de cada usuário.** "Hoje" passou a ser calculado no fuso da pessoa
  (`users.timezone`), e não no relógio do servidor (UTC). Antes, um check-in às 22h
  no Brasil caía no dia seguinte. Vale para hábitos, revisão, saúde, finanças,
  estudos e tarefas. O banco de fusos vai embutido no binário (`time/tzdata`).
  - Base: `internal/appctx` e o middleware `RequireActiveUser`.
- **Contas isoladas.** Cartão, hábito e tarefa de outro usuário não podem mais ser
  referenciados, e o nome do cartão de outra pessoa não aparece mais nas transações.
  O upload de extrato ficou limitado a 2 MiB.
- **Login mais seguro:**
  - O e-mail é normalizado (maiúsculas não criam uma segunda conta).
  - Um cadastro simultâneo com o mesmo e-mail responde 409, e não mais 500.
  - Trocar a senha derruba as outras sessões.
  - Reusar um refresh token revoga todas as sessões.
  - `JWT_*` inválido impede o boot.
- **Sessão no navegador.** A renovação da sessão é única por aba e coordenada entre
  abas (Web Locks). O cache é limpo no logout.

### Etapa 1: convites e conta

**Decisão (29/09):** o app **não envia e-mail**. Convites e redefinição de senha
são **links que o admin gera no app** e envia por conta própria (Gmail ou WhatsApp).
Por isso não há Mailpit nem SES, nem verificação de e-mail separada: abrir o link
do convite já prova que o e-mail é da pessoa.

| Peça | Status |
|---|---|
| 1. Convites por link | ✅ |
| 2. "Esqueci minha senha" por link | ✅ |
| 3. Exportar e apagar a conta (LGPD), mais uma tela "Minha conta" | ⏳ **próxima** |
| 4. Registro de ações sensíveis (`audit_logs`) | ⏳ |

**Peça 1, convites** (migração `000011`):
- Cadastro só com convite. A exceção são os e-mails em `ADMIN_EMAILS`, que se
  cadastram sem convite, viram admin e são promovidos no boot.
- Cada convite vale 7 dias, é de uso único e fica preso ao e-mail convidado. O
  banco guarda apenas o hash SHA-256 do código.
- O admin gera e cancela convites em **Acessos** (`/acessos`). O convidado abre
  `/register?convite=<código>`.
- API:
  - `POST|GET|DELETE /api/v1/admin/invites`, só admin.
  - `GET /api/v1/auth/invites/{código}`, público.

**Peça 2, redefinição de senha** (migração `000012`):
- O admin gera o link em **Acessos**. Ele vale 1 hora, é de uso único, e gerar um
  link novo cancela o anterior.
- Trocar a senha pelo link derruba todas as sessões da conta.
- O convidado abre `/redefinir-senha?codigo=<código>`.
- API:
  - `POST /api/v1/admin/password-resets`, só admin.
  - `GET|POST /api/v1/auth/password-resets/{código}`, público.
- ⚠️ Como é o admin quem gera o link, ele consegue entrar em qualquer conta. É uma
  relação de confiança, aceitável num beta de família. A peça 4 precisa registrar
  cada link gerado.

**Peça 3, próxima:**
- `GET /me/export`: arquivo JSON com todos os dados do usuário.
- `DELETE /me`: pede a senha e apaga de verdade, via `ON DELETE CASCADE`.
- Tela "Minha conta": nome, senha, baixar dados e apagar conta.

## Pendências anotadas (não esquecer)

**Para a Etapa 5 (AWS e nginx):**
- O rate limit de login no nginx aponta para `/api/auth/`, mas as rotas reais são
  `/api/v1/auth/`, então o limite mais rígido nunca é aplicado. Corrigir ao
  configurar o edge.
- HSTS só é enviado quando `r.TLS != nil`, o que nunca acontece atrás de um proxy
  TLS. Confiar no `X-Forwarded-Proto`.
- Remover as credenciais padrão do compose (`changeme_in_production` e o
  admin/admin do Grafana), fechar as portas expostas do Postgres, Redis e
  Prometheus, e proteger o `/metrics`.
- O `backend/Dockerfile` fixa `GOARCH=amd64`. Buildar para `arm64` se usarmos
  EC2 `t4g`.
- Não há backups do Postgres.
- A topologia recomendada para aprender e gastar pouco é **um único EC2 com o
  `docker compose`**, com backup diário via `pg_dump` para o S3. A alternativa é
  S3 + CloudFront + EC2 + RDS.

**Dívidas menores:**
- Índice único em `lower(email)`: hoje login e cadastro comparam com `lower()`,
  sem índice.
- Uma sessão de estudo com `habit_id` de outro usuário é ignorada em silêncio.
  Linhas antigas que apontam para o cartão de outro usuário só ficam ocultas.
  Cartões desativados ainda são aceitos na importação.
- `HabitService.CheckIn` usa o relógio real, sem um relógio de teste compartilhado.
- A renovação de sessão não tem timeout. Um logout numa aba só é percebido nas
  outras no próximo 401.
- O modelo `ANTHROPIC_MODEL` padrão está desatualizado. Atualizar na Etapa 3.

## Como trabalhamos

- **Commits direto no `main`**, em Conventional Commits e **sem coautoria do
  Claude**. O Claude envia ao GitHub depois de revisar e ver os testes passarem, e
  só chama o Luiz se o CI falhar.
- **Ritmo:** uma coisa por vez, com o Claude explicando cada mudança. Recursos pagos
  na AWS, domínio e envio de convites exigem aprovação do Luiz.
- O CI (GitLeaks, backend, frontend, mobile e Trivy) precisa ficar verde a cada push.
- O plano de governança (`docs/superpowers/`) está **pausado** até o MVP estar no ar.
