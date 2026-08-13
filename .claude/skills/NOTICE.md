# Skills deste projeto

Skills são pastas de instruções que o agente carrega sozinho quando o assunto
aparece. Diferente de documentação solta, elas chegam no momento da decisão.

## Skills do MyRoutine

Escritas para este projeto. São o que faz a constituição do produto
(`.agents/rules/product-constitution.md`) valer na prática:

| Skill | Para quê |
|---|---|
| `habit-review-flow` | Revisão consciente de dias sem registro, sem cobrança |
| `cross-module-data-flow` | Uma fonte de verdade por evento; módulos referenciam |
| `non-binary-habit-modeling` | Métrica é enriquecimento opcional, nunca bloqueia |
| `go-migration-safety` | Migrações incrementais e não-destrutivas |
| `dashboard-widget-design` | Cada bloco responde uma pergunta de decisão |
| `glassmorphism-ui-conventions` | Dark-first, sem vermelho para ausência de dado |

Ficavam em `.agents/skills/`, onde eram apenas documentação — o agente só as
lia se fosse procurar. Aqui elas são carregadas automaticamente.

## Skills de terceiros

Copiadas de https://github.com/anthropics/skills sob **Apache License 2.0**.
Cada pasta traz seu `LICENSE.txt` original.

| Skill | Para quê | Origem |
|---|---|---|
| `webapp-testing` | Dirigir o app no navegador com Playwright: descobrir seletores, capturar console, subir servidor | anthropics/skills |
| `skill-creator` | Criar e melhorar as skills acima | anthropics/skills |

As skills `docx`, `pdf`, `pptx` e `xlsx` daquele repositório **não** foram
copiadas: são source-available (© Anthropic, todos os direitos reservados),
não open source, e este repositório é público.
