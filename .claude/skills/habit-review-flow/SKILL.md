---
name: habit-review-flow
description: Define o mecanismo de revisão consciente de dias sem
  check-in de hábito, inspirado na migração do Bullet Journal. Use
  quando implementar telas ou lógica relacionada a streaks quebradas,
  dias perdidos, revisão semanal de hábitos, ou qualquer feature que
  lide com ausência de dado em habit_logs.
---

# Revisão de dias perdidos sem culpa

## Regra central
Um dia sem log não é "falha" — é estado neutro não revisado. O sistema
nunca julga automaticamente (sem vermelho, sem "X", sem streak quebrada
tratada como punição visual). A decisão sobre o que fazer com um dia
perdido é sempre consciente e do usuário, nunca automática.

## Mecanismo: migração / descarte
Uma vez por semana (nunca diariamente — cadência diária de "você
falhou" gera ansiedade, não reflexão), o sistema mostra os buracos da
semana e oferece, por dia perdido, duas ações conscientes:

- **Migrar** — "não fiz, mas ainda quero, mover intenção pra próxima
  semana". Não cria um `habit_log` — só um marcador de intenção.
- **Descartar** — "esse dia não vai ser recuperado, e tudo bem". Marca
  como `intentionally_skipped`, visualmente neutro (nunca vermelho).

Se o usuário ignorar a revisão, o dia permanece como estado neutro —
não vira falha por omissão.

## Schema
```sql
CREATE TABLE habit_day_reviews (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  habit_id UUID REFERENCES habits(id) ON DELETE CASCADE,
  review_date DATE NOT NULL,
  status VARCHAR(20) NOT NULL, -- 'migrated' | 'discarded'
  reviewed_at TIMESTAMP DEFAULT now(),
  UNIQUE(habit_id, review_date)
);
```

## Regras de UI
- Dias sem log = cor neutra (cinza), nunca vermelho.
- Dias `discarded` = cinza com indicador sutil (ícone pequeno), nunca
  "X" ou traço vermelho.
- Nunca mostrar contador de "dias falhados" como métrica proeminente —
  isso reintroduz culpa pela porta dos fundos.
- A tela de revisão semanal é opt-in na navegação, não um pop-up
  interruptivo ou notificação obrigatória.
- Streak (se exibida) deve ser tratada como informação neutra, não como
  troféu — não usar animação de "quebra" quando some.

## Anti-padrões a evitar explicitamente
- Notificação diária tipo "você não fez X hoje!"
- Cor vermelha ou ícone de erro em dias sem log.
- Copy que atribui intenção negativa ("você desistiu", "você falhou").
- Qualquer mecanismo de "recuperar streak" gamificado (ex: pagar para
  restaurar) — viola também a regra de não-gamificação da constitution.
