---
name: go-migration-safety
description: Convenções para escrever migrações de banco PostgreSQL na
  API Go do MyRoutine. Use quando o usuário pedir para alterar o schema
  do banco, criar uma migração, adicionar colunas ou tabelas, ou mexer em
  arquivos de migration.
---

# Migrações seguras de banco (Go + PostgreSQL)

## Regra central
Migrações neste projeto devem ser incrementais e não-destrutivas por
padrão. Este é um app de produção pessoal em uso contínuo — não há
ambiente de staging separado tolerando downtime ou perda de dado.

## Convenções
1. Preferir `ALTER TABLE ... ADD COLUMN` com `NULL` ou `DEFAULT`
   explícito a criar tabelas novas, quando o dado se encaixa numa
   entidade existente (ex: métricas de hábito viraram colunas em
   `habits`/`habit_logs`, não uma tabela `habit_metrics` separada — ver
   skill `non-binary-habit-modeling`).
2. Nunca fazer `DROP COLUMN` ou `DROP TABLE` na mesma migração que
   adiciona a substituição. Adicionar o novo, migrar dado se necessário,
   e só then, em uma migração separada e explicitamente confirmada pelo
   usuário, remover o antigo.
3. Toda coluna nova deve ser `NULL`able ou ter `DEFAULT`, para não
   quebrar inserts existentes no código Go que ainda não conhece o campo
   novo.
4. Nomear migrações com prefixo numérico sequencial e descrição curta
   (ex: `0007_add_habit_metric_columns.sql`), consistente com o padrão
   já usado no diretório de migrations do projeto — verificar o padrão
   existente antes de criar a próxima.
5. Adicionar índice em colunas usadas em filtro/order by de queries
   frequentes (ex: `habit_logs(habit_id, logged_date)` para as queries
   de heatmap, streak e "hábito em risco").

## Ao propor uma migração
1. Mostrar o SQL da migração antes de rodar.
2. Explicar se é reversível (down migration) e incluir o down quando o
   framework de migração do projeto suportar.
3. Se a migração for potencialmente lenta em tabela grande (ex:
   `ALTER TABLE` com validação de constraint em tabela com muitas
   linhas), sinalizar isso explicitamente antes de aplicar.
