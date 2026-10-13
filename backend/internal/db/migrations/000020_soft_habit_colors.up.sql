-- Redesenho "caderno calmo" (12/10/2026): as cores saturadas do tema antigo
-- viram os tons equivalentes da paleta nova (mesmo brilho, pouca saturação).
-- Só dado; nada de schema muda além do DEFAULT de habits.color.

CREATE OR REPLACE FUNCTION pg_temp.soft_color(c text) RETURNS text AS $$
  SELECT CASE upper(c)
    WHEN '#0071E3' THEN '#7EA6D8'  -- azul    → céu
    WHEN '#0A84FF' THEN '#7EA6D8'
    WHEN '#34C759' THEN '#7FB59A'  -- verde   → sálvia
    WHEN '#30D158' THEN '#7FB59A'
    WHEN '#FF9F0A' THEN '#D9A85B'  -- laranja → âmbar
    WHEN '#FF3B30' THEN '#D894A8'  -- vermelho → rosa
    WHEN '#FF2D55' THEN '#D894A8'  -- rosa    → rosa
    WHEN '#AF52DE' THEN '#A99AD6'  -- roxo    → lavanda
    WHEN '#5856D6' THEN '#A99AD6'  -- índigo  → lavanda
    WHEN '#5AC8FA' THEN '#6FB0B5'  -- ciano   → petróleo
    ELSE c
  END
$$ LANGUAGE sql IMMUTABLE;

UPDATE habits        SET color = pg_temp.soft_color(color) WHERE color IS DISTINCT FROM pg_temp.soft_color(color);
UPDATE monthly_goals SET color = pg_temp.soft_color(color) WHERE color IS NOT NULL AND color IS DISTINCT FROM pg_temp.soft_color(color);
UPDATE tasks         SET color = pg_temp.soft_color(color) WHERE color IS NOT NULL AND color IS DISTINCT FROM pg_temp.soft_color(color);

ALTER TABLE habits ALTER COLUMN color SET DEFAULT '#7FB59A';
