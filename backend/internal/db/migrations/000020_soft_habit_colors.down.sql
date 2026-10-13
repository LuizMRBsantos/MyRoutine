-- Volta às cores do tema antigo. Aproximado: rosa e lavanda vieram de duas
-- cores cada e voltam para uma só (rosa → #FF2D55, lavanda → #AF52DE).
CREATE OR REPLACE FUNCTION pg_temp.old_color(c text) RETURNS text AS $$
  SELECT CASE upper(c)
    WHEN '#7EA6D8' THEN '#0071E3'
    WHEN '#7FB59A' THEN '#34C759'
    WHEN '#D9A85B' THEN '#FF9F0A'
    WHEN '#D894A8' THEN '#FF2D55'
    WHEN '#A99AD6' THEN '#AF52DE'
    WHEN '#6FB0B5' THEN '#5AC8FA'
    ELSE c
  END
$$ LANGUAGE sql IMMUTABLE;

UPDATE habits        SET color = pg_temp.old_color(color);
UPDATE monthly_goals SET color = pg_temp.old_color(color) WHERE color IS NOT NULL;
UPDATE tasks         SET color = pg_temp.old_color(color) WHERE color IS NOT NULL;

ALTER TABLE habits ALTER COLUMN color SET DEFAULT '#0071E3';
