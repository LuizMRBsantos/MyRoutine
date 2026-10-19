-- Revisão: dias sem registro e sem decisão por mais de 2 dias viram
-- "descartado" automaticamente (decisão do Luiz em 19/10/2026). A coluna
-- separa o descarte feito pelo app do feito pela pessoa, para o histórico
-- continuar honesto.
ALTER TABLE habit_day_reviews ADD COLUMN IF NOT EXISTS auto BOOLEAN NOT NULL DEFAULT false;
