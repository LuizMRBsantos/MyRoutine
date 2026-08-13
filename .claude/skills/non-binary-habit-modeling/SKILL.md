---
name: non-binary-habit-modeling
description: Guia como modelar hábitos com métricas (distância, duração,
  quantidade, rpe) no schema e na UI do MyRoutine. Use quando o usuário
  pedir para adicionar campos de métrica a um hábito, criar tipos de
  hábito não-binários, mexer nas tabelas habits ou habit_logs relacionadas
  a métricas, ou implementar telas de check-in com valores numéricos.
---

# Modelagem de hábitos não-binários

## Regra central
Todo hábito, mesmo com métrica, sempre tem um check-in booleano de base.
A métrica é enriquecimento opcional do check-in — nunca bloqueia o
registro. Se o usuário correu 3km de uma meta de 5km, ainda é um
check-in válido, com dado parcial.

## Schema
```sql
-- habits
metric_type   VARCHAR(20)  -- NULL | 'duration_min' | 'distance_km' | 'quantity' | 'rpe_scale'
metric_unit   VARCHAR(20)  -- 'km', 'min', 'L', 'copos', etc.
target_value  NUMERIC      -- meta opcional, ex: 5 (km), 30 (min)

-- habit_logs
metric_value  NUMERIC      -- valor real registrado, nullable
rpe           SMALLINT     -- 1-10, nullable, específico para hábitos físicos
```

Não normalizar `metric_type` em tabela separada (`habit_metric_types`)
enquanto os tipos conhecidos forem só os 4 acima — seria
over-engineering. Só revisitar isso se o produto passar a precisar de
unidades customizadas ou múltiplas métricas por hábito.

## Regras de UI
- O check-in continua sendo 1 toque para o caso comum (a maioria dos
  hábitos do dia a dia é binária).
- Campo de métrica só aparece na tela de check-in se `metric_type` do
  hábito não for NULL — e mesmo assim é opcional, permitir salvar sem
  preencher.
- Na tela de criação de hábito, não usar um select genérico "tipo do
  hábito" com N opções. Usar um toggle simples ("esse hábito tem uma
  métrica?") e só então perguntar qual tipo.
- Ao exibir progresso de hábitos com meta, mostrar "completou X% da
  meta" ao longo do tempo, não só um check binário — streak binário
  esconde degradação de qualidade (ex: correr cada vez menos km mas
  ainda "bater o check-in").

## Ao adicionar um novo tipo de métrica
1. Adicionar o valor em `metric_type` (string, não enum de banco — mais
   barato de estender).
2. Definir a unidade padrão esperada em `metric_unit`.
3. Atualizar o componente de check-in no frontend para renderizar o
   input certo (numérico simples, slider de 1-10 para rpe, etc.) — nunca
   criar formulário genérico "preencha os campos" para isso.
