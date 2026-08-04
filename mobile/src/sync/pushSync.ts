import { Q } from '@nozbe/watermelondb';
import { database } from '@/src/db/database';
import { apiRequest } from '@/src/api/client';
import type Transaction from '@/src/db/models/Transaction';
import type HealthRecord from '@/src/db/models/HealthRecord';
import { resolveHabitForWorkout, type RemoteHabit } from './habitMapping';

export interface SyncResult {
  transactionsSynced: number;
  workoutsSynced: number;
  skipped: number;
}

/**
 * Push unidirecional: o celular escreve, o servidor consolida.
 *
 * Não usamos o protocolo de sync completo do WatermelonDB de propósito — ele
 * exigiria endpoints de pull/push com versionamento por tabela, custo alto
 * para um app de um usuário só. A idempotência vem dos índices únicos
 * (source_type, source_id) do servidor: reenviar o mesmo registro atualiza,
 * nunca duplica.
 */
export async function pushPendingRecords(): Promise<SyncResult> {
  const result: SyncResult = { transactionsSynced: 0, workoutsSynced: 0, skipped: 0 };

  const pendingTx = await database
    .get<Transaction>('transactions')
    .query(Q.where('push_status', 'pending'))
    .fetch();

  const pendingHealth = await database
    .get<HealthRecord>('health_records')
    .query(Q.where('push_status', 'pending'))
    .fetch();

  if (pendingTx.length === 0 && pendingHealth.length === 0) {
    return result;
  }

  // Só o que o servidor confirmou entra nestas listas.
  const syncedTx: Transaction[] = [];
  const syncedHealth: HealthRecord[] = [];

  // ─── Transações → módulo Finanças ───────────────────────────────────────
  for (const tx of pendingTx) {
    await apiRequest('/finance/transactions', {
      method: 'POST',
      body: JSON.stringify({
        amount_cents: Math.round(tx.value * 100),
        kind: 'expense',
        category: tx.category,
        description: tx.description || 'Registro do diário',
        method: tx.method,
        occurred_on: tx.date,
        source_type: 'track_day',
        source_id: tx.id,
      }),
    });
    syncedTx.push(tx);
    result.transactionsSynced++;
  }

  // ─── Treinos → check-in do hábito de saúde correspondente ───────────────
  if (pendingHealth.length > 0) {
    const { habits } = await apiRequest<{ habits: RemoteHabit[] }>('/habits');

    for (const record of pendingHealth) {
      const habit = resolveHabitForWorkout(record.type, habits);
      if (!habit) {
        // Sem hábito correspondente: fica pendente para uma próxima tentativa
        // (o usuário pode criar o hábito depois). Nunca chuta um destino.
        result.skipped++;
        continue;
      }

      const metrics: Record<string, number> = {};
      if (record.distance != null) metrics.km = record.distance;
      if (record.time != null) metrics.time_min = record.time;
      if (record.rpe != null) metrics.rpe = record.rpe;

      await apiRequest(`/habits/${habit.id}/checkin`, {
        method: 'POST',
        body: JSON.stringify({
          date: record.date,
          metrics,
          source_type: 'track_day',
          source_id: record.id,
        }),
      });
      syncedHealth.push(record);
      result.workoutsSynced++;
    }
  }

  const now = Date.now();
  await database.write(async () => {
    await database.batch(
      ...([
        ...syncedTx.map((tx) =>
          tx.prepareUpdate((t) => {
            t.pushStatus = 'synced';
            t.pushedAt = now;
          })
        ),
        ...syncedHealth.map((record) =>
          record.prepareUpdate((h) => {
            h.pushStatus = 'synced';
            h.pushedAt = now;
          })
        ),
      ] as never[])
    );
  });

  return result;
}
