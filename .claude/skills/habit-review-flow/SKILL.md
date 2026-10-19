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
perdido é da pessoa enquanto o dia está na janela de revisão.

**Mudança de regra (decisão do Luiz em 19/10/2026):** a janela de revisão
é de **2 dias** (ontem e anteontem). Dias que passam disso sem registro e
sem decisão são **descartados pelo app**, gravados com `auto = true` em
`habit_day_reviews` para separar do descarte feito pela pessoa. Uma
decisão da pessoa nunca é sobrescrita. O descarte automático continua
neutro: sem vermelho, sem contador, sem aviso.

## Mecanismo: migração / descarte
A tela de Revisão (opt-in no menu, nunca notificação) mostra os dias da
janela ainda sem decisão e oferece, por dia perdido, duas ações; a pessoa
escolhe e toca em "Salvar revisão", e os dias decididos saem da lista:

- **Migrar** — "não fiz, mas ainda quero, mover intenção pra próxima
  semana". Não cria um `habit_log` — só um marcador de intenção.
- **Descartar** — "esse dia não vai ser recuperado, e tudo bem". Marca
  como `intentionally_skipped`, visualmente neutro (nunca vermelho).

Se a pessoa ignorar a revisão, depois da janela o dia vira "descartado"
automático (`auto = true`) — neutro, nunca "falha".

## Schema
```sql
CREATE TABLE habit_day_reviews (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  habit_id UUID REFERENCES habits(id) ON DELETE CASCADE,
  review_date DATE NOT NULL,
  status VARCHAR(20) NOT NULL, -- 'migrated' | 'discarded'
  reviewed_at TIMESTAMP DEFAULT now(),
  auto BOOLEAN NOT NULL DEFAULT false, -- migração 021: descartado pelo app
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
