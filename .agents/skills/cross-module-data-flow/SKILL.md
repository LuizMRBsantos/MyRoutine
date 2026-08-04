---
name: cross-module-data-flow
description: Define como dados fluem entre Track Day, Hábitos, Saúde e
  outros módulos do MyRoutine sem duplicação. Use quando implementar
  parsing de entradas do diário livre (Track Day), criar referências
  entre módulos, sincronizar dados entre abas, ou quando o usuário
  mencionar que uma informação deveria aparecer em mais de um lugar do
  app.
---

# Fluxo de dados entre módulos

## Regra central
Um evento tem uma única fonte de verdade. Outros módulos referenciam o
evento original, nunca copiam o dado. Isso é o espírito Obsidian do
produto: um nó atômico, múltiplos links apontando pra ele.

## Padrão de implementação
1. O módulo de origem (ex: Track Day) recebe a entrada bruta (texto
   livre).
2. Um parser extrai um evento estruturado (ex:
   `{type: 'run', distance_km: 8, rpe: 6}`).
3. Esse evento gera um `habit_log` com:
   - `source_type` = nome do módulo de origem (ex: `'track_day'`,
     `'manual'`, `'import'`)
   - `source_id` = id do registro de origem (ex: id da entrada do Track
     Day)
   - `metric_value` / `rpe` preenchidos automaticamente (ver skill
     `non-binary-habit-modeling`)
4. Módulos consumidores (ex: aba Saúde) **consultam** `habit_logs`
   filtrando por `habits.category` — nunca criam ou mantêm sua própria
   cópia da tabela de eventos.

```sql
ALTER TABLE habit_logs ADD COLUMN source_type VARCHAR(20) DEFAULT 'manual';
ALTER TABLE habit_logs ADD COLUMN source_id UUID;
```

## Ao implementar um parser novo (ex: Track Day → Hábitos)
1. Extrair campos estruturados do texto (tipo de atividade, métrica,
   RPE, duração).
2. Se a confiança do parsing for alta, criar o check-in automaticamente
   e permitir desfazer (undo já existe no fluxo de check-in manual —
   reaproveitar).
3. Se a confiança for baixa ou ambígua, perguntar confirmação ao usuário
   antes de criar o check-in.
4. Nunca duplicar o dado em uma tabela nova específica do módulo
   consumidor — se um módulo "precisa" da própria tabela, isso é sinal
   de que o design está indo contra este princípio; revisar antes de
   prosseguir.

## Checklist antes de adicionar uma feature de "integração" entre módulos
- [ ] Existe uma fonte de verdade clara para este dado?
- [ ] O módulo consumidor está lendo (query) ou copiando (insert)?
- [ ] Se copiando, há uma forma de fazer via referência (`source_id`)
      em vez disso?
