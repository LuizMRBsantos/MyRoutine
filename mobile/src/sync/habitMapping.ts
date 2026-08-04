// Mapa explícito tipo-de-treino → nome do hábito no servidor.
// Deliberadamente explícito em vez de heurística: um "match aproximado"
// erraria em silêncio e criaria check-in no hábito errado.
export const WORKOUT_HABIT_NAMES: Record<string, string[]> = {
  run: ['correr', 'corrida'],
  bike: ['pedalar', 'ciclismo', 'bike'],
  swim: ['nadar', 'natação', 'natacao'],
};

export interface RemoteHabit {
  id: string;
  name: string;
  category: string;
  metric_config?: { key: string }[];
}

/**
 * Escolhe o hábito de saúde correspondente ao tipo de treino.
 * Retorna null quando não há hábito claro — nesse caso o registro fica
 * pendente em vez de ir para o hábito errado.
 */
export function resolveHabitForWorkout(
  workoutType: string,
  habits: RemoteHabit[]
): RemoteHabit | null {
  const aliases = WORKOUT_HABIT_NAMES[workoutType];
  if (!aliases) return null;

  const healthHabits = habits.filter((h) => h.category === 'health');
  const match = healthHabits.find((h) => {
    const name = h.name.toLowerCase();
    return aliases.some((alias) => name.includes(alias));
  });

  return match ?? null;
}
