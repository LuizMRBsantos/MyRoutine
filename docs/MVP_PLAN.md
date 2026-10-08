# MyRoutine — Plano do beta fechado (MVP)

> Documento vivo. Última atualização: 29/09/2026 (Etapa 1 concluída).
> Substitui o roteiro de `docs/PROJECT_CONTEXT.md` (de 28/06, desatualizado).

## Objetivo

Deixar o MyRoutine pronto para **até 10 convidados** (amigos e família) usarem de
verdade, em **web, iPhone e Mac**. Nos dois últimos, ele roda como **PWA**, o site
instalado pela tela inicial ou pelo dock, sem App Store. A hospedagem é na **AWS**.

**Decisão de 29/09:** a IA foi **adiada para depois do lançamento**, porque pede
decisões de produto com calma. A nova ordem é: hospedagem (5), depois notificações (4),
depois a revisão final e os convites (6). **Os convites saem quando o app estiver
no ar e com notificações.** A IA chega depois, como novidade para os convidados.

## Etapas

| # | Etapa | Status |
|---|---|---|
| 0 | Consertar o que quebra com várias pessoas | ✅ concluída |
| 1 | Convites e conta | ✅ concluída |
| 2 | App no celular e no Mac (PWA, layout responsivo, Track Day web, revisão semanal) | ✅ concluída |
| 3 | IA (Claude): assistente e insights sob demanda | ⏸ adiada (pós-lançamento) |
| 4 | Notificações (Web Push): uma consolidada por horário, no fuso de cada pessoa | ⏳ **próxima** |
| 5 | Colocar no ar com **Vercel + Supabase** | ✅ no ar em `myroutine-eight.vercel.app` (pendente: trocar a senha do banco antes dos convites) |
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
| 4. Revisão semanal (`/revisao`) | ✅ |

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

**Peça 4, revisão semanal** (`/revisao`, "↺ Revisão" no menu; no celular fica no "Mais"):
- A revisão tinha sido desligada porque aparecia sozinha como uma faixa com
  contador de "N dias sem registro", o que viola a skill `habit-review-flow`.
  Voltou como **página por escolha**, sem contador, com texto neutro e sem
  comemoração.
- Os dias aparecem agrupados. Para cada hábito há a escolha **Migrar / Descartar**,
  e dá para mudar de ideia (o backend faz upsert). O componente antigo foi removido.
- Corrigido: dias **anteriores à criação do hábito** contavam como "sem registro".
  Um hábito novo chegava com uma semana de pendências. O dia de criação agora
  entra no fuso do usuário.
- Os testes do frontend garantem as regras do produto: nenhum número de dias e
  nenhum texto de comemoração.

### Etapa 5: colocar no ar com Vercel + Supabase

**Decisão (06/10):** sai a AWS, que ficaria a US$ 10–30 por mês sem créditos. O
app vai para **Vercel** (telas + API em Go) e **Supabase** (Postgres). Custo:
**R$ 0 por mês** nos planos gratuitos, mais o domínio `.com.br` (~R$ 40 por ano).
A conta AWS não tem recursos criados nem custo.

```
 celular / Mac / navegador (PWA)
            │  https://myroutine.com.br
   ┌────────▼─────────────────────────────┐
   │ Vercel                               │
   │  ├─ arquivos do app (React) na CDN   │
   │  └─ /api/*, /health → função em Go   │  o mesmo backend, como "Vercel Function"
   └────────┬─────────────────────────────┘
            │ SSL, via pooler de conexões
   ┌────────▼────────┐        GitHub Actions: CI, migrações antes do deploy,
   │ Supabase        │        backup diário criptografado
   │ Postgres        │
   └─────────────────┘
```

**Adequações no código** (o Claude faz e explica):
1. **A API como função:** um pacote público `backend/server` monta o handler uma
   vez por instância (config, pool e admins). `api/index.go` fica na raiz, com um
   `go.mod` próprio. O `main.go` continua servindo para Docker e para o ambiente
   local.
2. **Banco pelo pooler do Supabase:** `default_query_exec_mode=simple_protocol`
   (o modo transação não aceita prepared statements), pool pequeno configurável
   (`DB_MAX_CONNS`) e SSL.
3. **Migrações fora do boot em produção** (`RUN_MIGRATIONS=false`): o GitHub
   Actions aplica as migrações pela conexão direta, **antes** do deploy de
   produção.
4. **Segurança do Supabase:** ele publica automaticamente uma API REST das
   tabelas do schema `public`, acessível com a chave pública. A proteção tem
   três partes:
   - desligar essa "Data API" no painel;
   - criar uma migração que liga o **RLS em todas as tabelas**, sem políticas, de
     modo que nada passa por ali (o backend conecta como dono e não é afetado);
   - um teste que garante RLS em toda tabela nova.
5. **`vercel.json`:**
   - build do frontend;
   - rotas: `/api/*` e `/health` vão para a função, e o resto cai no app (SPA);
   - cabeçalhos de segurança;
   - `sw.js` e manifesto sem cache;
   - função na região `gru1` (São Paulo), perto do Supabase em `sa-east-1`.
6. **IP do cliente:** a Vercel define `x-real-ip`, que já é suportado
   (`TRUSTED_IP_HEADER=X-Real-IP`).
7. **Limite de tentativas no Postgres:** o limite em memória vale só por
   instância, e no serverless há várias. Ele passa a ser contado numa tabela,
   valendo entre todas as instâncias.
8. **Cold start mais leve:** o hash bcrypt "falso" do login passa a ser gerado
   sob demanda, e não ao carregar (~250 ms a menos na primeira requisição).
9. **Backup:** o plano gratuito do Supabase **não tem backup para baixar**. Um
   GitHub Action diário faz `pg_dump`, **criptografa** e guarda por 30 dias. A
   restauração será testada uma vez.
10. **Notificações (Etapa 4):** o cron da Vercel no plano grátis roda 1 vez por
    dia, pouco para lembretes. A ideia é usar o `pg_cron` + `pg_net` do Supabase
    para chamar um endpoint interno a cada 15 minutos.

**Plataformas** (o Luiz faz, o Claude ensina):
1. ✅ Supabase: conta (login com GitHub), projeto na região **São Paulo** e senha
   forte do banco. Desligar a Data API e copiar as duas *connection strings*:
   pooler 6543 e direta.
2. ✅ Vercel: conta Hobby criada e repositório importado (preset "Other", que
   segue o `vercel.json`). As variáveis estão configuradas (`DATABASE_URL`, `JWT_SECRET`, `ADMIN_EMAILS`, `APP_ENV`...).
3. ✅ **Teste de viabilidade (06/10), em `myroutine-eight.vercel.app`:** passou.
   - A Vercel compila Go 1.26.4.
   - A função na raiz com `replace` para o backend funciona.
   - **A função recebe o caminho original** (`/api/...`).
   - `x-real-ip` presente.
   - Rotas do React, cabeçalhos de segurança e CSP ok, `sw.js` sem cache.
   - Região `gru1` (São Paulo).
   - O endpoint temporário foi removido depois do teste.
4. Domínio `.com.br` no Registro.br, apontado para a Vercel (HTTPS automático).
5. Secrets no GitHub para migrações, backup e deploy.

**Limites dos planos gratuitos (não esconder):**
- Supabase Free:
  - o projeto **pausa depois de 7 dias sem uso**, e é reativado no painel sem
    perder dados;
  - 500 MB de banco;
  - sem backup próprio, por isso o item 9.
- Vercel Hobby:
  - uso pessoal e não comercial, o que serve para um beta com amigos;
  - funções com tempo máximo por requisição;
  - cron 1 vez por dia.
- Go na Vercel: é um runtime menos usado que Node, e a primeira requisição depois
  de um tempo parado leva algumas centenas de milissegundos. Por isso o teste de
  viabilidade vem primeiro.

**✅ No ar (08/10):** `https://myroutine-eight.vercel.app`.
- `/health` responde `healthy`, com o Postgres conectado pelo Transaction pooler
  (porta 6543, `simple_protocol`).
- Migrações 1 a 14 aplicadas pelo Session pooler (porta 5432), via
  `go run ./cmd/migrate`.
- 20 tabelas, todas com RLS; o papel `anon` não lê nada; a Data API está
  desligada.
- 7 variáveis só em Production, com `DATABASE_URL` e `JWT_SECRET` como Secret.
- Conta de admin criada pelo Luiz (`auth.register via admin_email` registrado no
  `audit_logs`).
- Verificado em produção:
  - sem login → 401;
  - cadastro sem convite → 403;
  - `/admin` sem login → 401.
- Integração do agente: Vercel CLI 63.1.0, plugin da Vercel e MCP
  `https://mcp.vercel.com` (escopo local do projeto).

**Falta na Etapa 5:**
- [x] **Esteira de deploy (08/10)**, em `.github/workflows/deploy.yml`:
  - roda **só depois do CI verde** no `main`;
  - ordem: migrações (Session pooler) → `vercel deploy --prod` (com
    `APP_VERSION`) → smoke test do `/health`;
  - um deploy por vez, e um commit antigo nunca sobrepõe um mais novo;
  - a Vercel não publica mais sozinha (`git.deploymentEnabled.main=false`);
  - segredos no GitHub: `MIGRATE_DATABASE_URL`, `VERCEL_TOKEN` (validade de 1
    ano), `VERCEL_ORG_ID` e `VERCEL_PROJECT_ID`;
  - primeira execução: tudo verde, versão `9147154` saudável.
- [x] **Backup diário criptografado (08/10)**, em `.github/workflows/backup.yml`:
  - roda às 03:17 BRT e também sob demanda;
  - `pg_dump -Fc` do schema `public`, com Postgres 17 igual ao do Supabase;
  - criptografado com `age` para a chave **pública** (variável
    `BACKUP_AGE_RECIPIENT`); a chave privada fica só no Mac do Luiz, em
    `~/.myroutine/backup-age-key.txt`;
  - artefato guardado por 30 dias;
  - **teste de restauração** rigoroso (`--exit-on-error`) num Postgres 17
    comum: 20 tabelas, usuários e registro de ações voltaram;
  - o teste encontrou 5 tabelas presas ao `extensions.uuid_generate_v4()` do
    Supabase, que não restauravam fora dele. Corrigido com a **migração 015**
    (`gen_random_uuid()`), a primeira aplicada pela esteira sozinha;
  - passo a passo em `docs/RUNNING.md`.
  ⚠️ **O Luiz precisa guardar uma cópia da chave privada no gerenciador de
  senhas.** Sem ela, nenhum backup abre.
- [x] **Limite de tentativas no Postgres (08/10)**, migração 016
  `auth_rate_limits`:
  - janela fixa de 1 minuto pelo relógio do banco;
  - incrementa e lê num comando só (teste: 20 tentativas simultâneas de 2
    cópias → exatamente 3 passam);
  - guarda só o SHA-256 do IP;
  - limpa linhas com mais de 1 hora;
  - se o contador falhar, deixa passar e registra no log (fail-open).
  Provado em produção: 10 respostas 401, a 11ª é 429 com Retry-After. O hash
  bcrypt "falso" agora é gerado sob demanda (`sync.OnceValue`), economizando ~250
  ms no cold start.
- [ ] **Trocar a senha do banco: adiado pelo Luiz em 08/10, OBRIGATÓRIO antes
  dos convites (Etapa 6).** Ela apareceu no chat e no histórico do terminal.
  Depois atualizar `DATABASE_URL` (Vercel) e `MIGRATE_DATABASE_URL` (segredo do
  GitHub).
- [~] Domínio `.com.br`: **decisão de 08/10: seguir sem domínio**, em
  `myroutine-eight.vercel.app`. Se ele vier depois, quem instalou o PWA precisa
  reinstalar pelo endereço novo.

**Já feito e que continua valendo:**
- versão real no `/health`;
- limite de tentativas nas rotas de auth (vai para o Postgres no item 7);
- IP por cabeçalho confiável;
- logs em JSON em produção;
- a web não desloga por 429 nem por 5xx;
- erros traduzidos.

A imagem Docker multi-arquitetura continua servindo para o ambiente local e para
o CI (Trivy).

### Etapa 4: notificações (quase pronta)

**Decisões de 08/10:**
- **Lembrete de compromisso:** 15 minutos antes de cada tarefa do Planner com
  horário e ainda não concluída. Tarefas que começam juntas viram **uma**
  notificação. Ligado por padrão.
- **Resumo da manhã às 07:00:** compromissos e hábitos do dia. Só é enviado se
  houver algo. Ligado por padrão.
- **Noite às 21:00:** o que ainda dá tempo hoje, em tom neutro. Opcional e
  desligado por padrão.
- Tudo é opt-in em Minha conta → Notificações. Os horários e a antecedência são
  ajustáveis, e sempre no fuso da pessoa.
- Dentro da constituição: nunca uma notificação por hábito, sem cobrança e sem
  vermelho, e nada é enviado quando não há o que dizer.

**Arquitetura:**
- Web Push (VAPID), via service worker do PWA. No iOS, só com o app instalado na
  Tela de Início (16.4+).
- O `pg_cron` + `pg_net` do Supabase chama `POST /api/v1/internal/notifications/dispatch`
  a cada 5 minutos, com um segredo próprio (`CRON_SECRET`). O cron da Vercel
  Hobby roda só 1 vez por dia.
- O registro de entregas (`notification_deliveries`) impede envio duplicado.
- Assinaturas recusadas pelo serviço de push (404/410) são removidas.

**Partes:**
1. ✅ Base (`b227515`): migração 017 (`push_subscriptions`,
   `notification_settings`, `notification_deliveries`, todas com RLS), endpoints
   de inscrição e preferências. Endereços aceitos só dos serviços oficiais de
   push (Apple, Google, Mozilla, Microsoft), para evitar SSRF.
2. ✅ Dispatcher (`1401d83`): seleção por fuso e horário, mensagens
   consolidadas, idempotência (cada envio é "reservado" antes), remoção de
   aparelhos expirados. O envio usa webpush-go e não segue redirecionamentos.
3. ✅ Web (`f717044`): seção Notificações em Minha conta, aviso de instalar no
   iPhone, `push`/`notificationclick` no `sw.ts`. Ao sair da conta o aparelho
   para de receber; ao entrar de novo, volta.
4. ✅ Produção (`83f1ea3`): variáveis `VAPID_*` e `CRON_SECRET` na Vercel
   (Production). Job `myroutine-notifications` do `pg_cron` criado em 08/10 pelo
   [`ops/notifications-cron.sql`](../ops/notifications-cron.sql), com o segredo no
   Vault; a primeira chamada respondeu 200. Cópia das chaves em
   `~/.myroutine/notifications.env` (passar para o gerenciador de senhas).
5. ⏳ Teste no iPhone real (PWA instalado).

### Depois do lançamento (ideias registradas)

- **Widget do Mac (pedido em 08/10).** Mostra os hábitos de hoje, com
  **check-in direto no widget** (AppIntents, macOS 14+), e os próximos
  compromissos do Planner.
  - Exige um app nativo em Swift/SwiftUI com WidgetKit, porque um PWA não cria
    widgets.
  - Usa a mesma API. Para a autenticação, o backend ganha uma **chave de acesso
    própria do widget**, que pode ser revogada em Minha conta.
  - Para uso pessoal, roda grátis via Xcode. Para distribuir aos convidados,
    precisa da Apple Developer (US$ 99 por ano).
- **Apps nas lojas (iPhone, iPad e Mac):**
  - Capacitor sobre o app React, com integrações nativas que justificam a App
    Store: Saúde/Apple Watch → check-in automático de treinos, widgets e Siri.
  - Na mesma base nativa entra o widget acima.
  - Requisitos: Apple Developer, política de privacidade, exclusão de conta no
    app (✅ já existe).
- **IA (Etapa 3):** assistente e insights sob demanda. Começar carregando a
  skill `claude-api`.

## Pendências anotadas (não esquecer)

- ✅ Rota `/health` do app colidia com o `/health` da API (recarregar a tela Saúde
  mostrava o JSON do servidor). Corrigido: a tela agora é `/saude`.

**Para a Etapa 5 (AWS e nginx):**
- ~~O rate limit de login no nginx aponta para o endereço errado.~~ Resolvido: o
  limite agora fica no app, e o nginx local foi corrigido.
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
- O Docker local e o testcontainers usam **Postgres 16**, mas a produção
  (Supabase) é **17**. Alinhar para 17.
- Acompanhar deploys pela **versão no `/health`**, e não pelo `headSha` dos runs:
  um run disparado por `workflow_run` aparece com o SHA do `main` no momento em
  que é disparado.
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
