# MyRoutine — Plano do beta fechado (MVP)

> Documento vivo. Última atualização: 29/09/2026 (Etapa 1 concluída).
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
| 1 | Convites e conta | ✅ concluída |
| 2 | App no celular e no Mac (PWA, layout responsivo, Track Day web, revisão semanal) | 🔄 3 de 4 peças |
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
| 3. Exportar e apagar a conta (LGPD), mais uma tela "Minha conta" | ✅ |
| 4. Registro de ações sensíveis (`audit_logs`) | ✅ |

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

**Peça 3, LGPD e "Minha conta"** (`/conta`, no menu para todos):
- `GET /api/v1/me/export`: baixa `myroutine-dados-AAAA-MM-DD.json` com o perfil e
  uma lista por tabela. Nunca inclui a senha nem as chaves de sessão.
  - O teste `TestExportCoversEveryUserTable` consulta o banco e **falha se surgir
    uma tabela com `user_id` que ficou fora da exportação**.
- `DELETE /api/v1/me` com `{"password"}`: apaga de verdade (cascade). O registro
  de ações é mantido sem dono e **sem IP e sem navegador**.
- A tela tem perfil (nome e fuso horário, com sugestão do fuso do aparelho),
  troca de senha (desloga, porque as sessões caem), baixar dados e apagar conta
  (em dois passos, com senha).
- O cadastro passou a enviar o fuso do navegador. Um fuso inválido no cadastro
  vira São Paulo, e no `PATCH /me` responde 400.
- Correções encontradas no caminho:
  - Senha atual errada agora responde `403`, e não `401`, que o frontend lia como
    sessão expirada.
  - O aviso global de erro dava toast duplicado, com a mensagem crua do servidor
    em inglês. Agora uma mutation pode marcar `meta: { handlesError: true }`.
  - A lista de fusos do navegador não trazia "UTC".

**Peça 4, registro de ações** (`audit_logs`, `internal/service/audit.go`):
- O que é registrado:
  - `auth.login`, `auth.login_failed`, `auth.register`, `auth.password_changed`,
    `auth.password_reset` e `auth.session_reuse_detected`;
  - `admin.invite_created`, `admin.invite_revoked` e
    `admin.password_reset_link_created`;
  - `account.exported` e `account.deleted`.
- O IP (o `X-Real-IP` confiável, com fallback para o peer) e o navegador vêm do
  middleware `RequestMeta`.
- As ações de admin e de conta gravam **na mesma transação** (tudo ou nada). O
  login é "melhor esforço": se a gravação falhar, a pessoa entra mesmo assim e o
  erro vai para o log do servidor.
- O link de nova senha é gravado **no histórico da pessoa afetada**, com
  `by_admin`, e aparece na exportação LGPD dela.
- `account.deleted` fica sem dono, sem IP e sem navegador.
- Corrigido no caminho: a proteção contra ataque de tempo no login usava um hash
  inválido (`$2a$12$dummy`), então nunca funcionou. Agora usa um hash bcrypt real.
- Ainda não há tela para ler o registro. Por enquanto a consulta é via SQL, e
  cada pessoa vê o seu na exportação.

### Etapa 2: app no celular e no Mac

| Peça | Status |
|---|---|
| 1. Tela que cabe no celular | ✅ |
| 2. App instalável (PWA): manifest, ícones, service worker | ✅ |
| 3. Diário (Track Day) na web, portando o parser do app mobile | ✅ |
| 4. Ligar a revisão semanal (`WeeklyReview`, hoje "estacionada") | ⏳ **próxima** |

**Peça 1, layout responsivo** (`AppLayout`):
- Acima de 1024px: menu lateral completo.
- De 769px a 1024px: menu lateral só com ícones (72px). Os nomes ficam escondidos
  só visualmente, para leitores de tela, e aparecem como dica ao passar o mouse.
- Até 768px: barra inferior opaca (Início, Hábitos, Planner, Finanças e Mais), e o
  "Mais" abre uma folha com o resto e o Sair. As áreas seguras do iPhone são
  respeitadas (`viewport-fit=cover` + `env(safe-area-inset-*)`).
- Verificado no navegador (Playwright com API simulada) em 393, 900 e 1440px:
  nada vaza para os lados e o console não mostrou erros.
- Corrigido: o script `type-check` rodava `tsc --noEmit` na raiz, que tem
  `"files": []`, e **não checava nada**. Agora é `tsc -b`, e o erro que ele deixava
  passar foi provado.

**Peça 2, PWA** (`vite-plugin-pwa`, estratégia `injectManifest`, `src/sw/sw.ts`):
- O cache guarda **só arquivos estáticos**. Nenhuma resposta da API é guardada, e
  as navegações para `/api`, `/health` e `/metrics` nunca usam o cache.
- Manifesto em modo standalone, com ícones 192/512, maskable e o
  `apple-touch-icon` (180). A fonte do ícone é `public/icons/icon.svg`. Há metas
  de iOS: nome, barra de status translúcida e `theme-color` para claro e escuro.
- `PwaStatus` mostra a faixa "sem internet" e o aviso "Nova versão ·
  Atualizar/Depois" (`registerType: 'prompt'`). A troca de versão nunca acontece
  sozinha.
- `frontend/nginx.conf`: `sw.js` e `manifest.webmanifest` sem cache.
- Verificado num Chromium real (build de produção): o SW fica ativo, o manifesto
  e os ícones respondem 200, e **sem internet o app abre logado**.
- Corrigido graças a essa verificação:
  - Abrir o app sem rede **deslogava a pessoa**. A renovação da sessão só desloga
    agora quando o servidor recusa, e nunca por falta de rede.
  - O Dashboard mostrava "Nenhum hábito" quando a busca falhava. Agora diz que
    não deu para carregar. O link "Criar hábito" não recarrega mais o app.
- ⚠️ No iPhone de verdade, o service worker exige **HTTPS**. O teste real de
  "Adicionar à Tela de Início" fica para depois da Etapa 5 (AWS com domínio).
- `npm audit`: 0 vulnerabilidades. O `undici` do jsdom, usado só nos testes, foi
  atualizado.

**Peça 3, Diário / Track Day** (`/diario`, 2º item do menu):
- **Backend** (migração `000013`, `internal/service/journal.go`):
  - Há um texto por pessoa por dia, com no máximo 20 mil caracteres. Ele entra
    na exportação LGPD, e o teste de cobertura pegou a tabela nova antes de ela
    ser incluída.
  - O `source_id` de um item é derivado de forma determinística: UUIDv5 do id do
    diário com a linha normalizada. A mesma linha sempre gera a mesma etiqueta,
    então registrar de novo não duplica.
  - O gasto vira uma transação do dia, e o treino vira o check-in do hábito, os
    dois com `source_type='track_day'`.
  - O diário **nunca sobrescreve** um check-in feito à mão ou vindo de outro
    lugar: nesse caso responde 409 e explica.
  - API:
    - `GET` e `PUT /journal/{data}`;
    - `POST /journal/{data}/items`;
    - `DELETE /journal/{data}/items/{source_id}`.
- **Web:**
  - O parser foi trazido do app mobile (`src/lib/trackDay/parser.ts`), com os 23
    testes herdados.
  - Corrigidos 3 bugs do parser, comprovados no original:
    - "30min" era lido como distância;
    - "1h" virava 1 minuto;
    - "1500m" virava 1500 km.
  - Novidades do parser:
    - valor em centavos, inclusive `1.234,56`;
    - `#tag` convertida para uma categoria de Finanças;
    - identidade da linha (`lineKey`) igual à do backend.
  - A tela salva sozinha 0,8s depois da última digitação, e também ao trocar de
    dia e ao sair do campo.
  - O painel "Reconhecido no texto" traz Registrar, "✓ Registrado" e Desfazer.
    Mostra também o que foi registrado de uma linha que saiu do texto.
  - Um treino só vai para um hábito de Saúde com nome correspondente. Sem esse
    hábito, a tela orienta a criar um.
- **Corrigido no caminho:**
  - **Segurança:** o `ON CONFLICT (source_type, source_id)` das transações era
    global. Um `source_id` de outra pessoa **sobrescrevia o gasto dela**. Agora
    só atualiza a linha do próprio dono, e o caso responde 409.
  - **Visual:** não havia estilo de botão desativado no app inteiro. Agora existe
    `.btn:disabled`.
- Menu: Início, Diário, Hábitos e Planner ficam na barra do celular, e Finanças
  passou para o "Mais". A ordem ainda pode mudar se o Luiz preferir.

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
- O calendário do Dashboard mostra um texto de desenvolvimento como subtítulo
  ("Visão Google Calendar com ponteiro de tempo ao vivo…"). Trocar no polimento.
- GitLeaks: rodar sempre no projeto inteiro antes do push. Já houve dois alarmes
  falsos em testes, e os achados revisados ficam no `.gitleaksignore`.

## Como trabalhamos

- **Commits direto no `main`**, em Conventional Commits e **sem coautoria do
  Claude**. O Claude envia ao GitHub depois de revisar e ver os testes passarem, e
  só chama o Luiz se o CI falhar.
- **Ritmo:** uma coisa por vez, com o Claude explicando cada mudança. Recursos pagos
  na AWS, domínio e envio de convites exigem aprovação do Luiz.
- O CI (GitLeaks, backend, frontend, mobile e Trivy) precisa ficar verde a cada push.
- O plano de governança (`docs/superpowers/`) está **pausado** até o MVP estar no ar.
