import { Q } from '@nozbe/watermelondb';
import * as Crypto from 'expo-crypto';
import { database } from '@/src/db/database';
import type Note from '@/src/db/models/Note';
import type Transaction from '@/src/db/models/Transaction';
import type HealthRecord from '@/src/db/models/HealthRecord';
import { parseNoteContent } from '@/src/parser';

export function todayStr(date = new Date()): string {
  const pad = (n: number) => (n < 10 ? `0${n}` : `${n}`);
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
}

const notes = () => database.get<Note>('notes');
const transactions = () => database.get<Transaction>('transactions');
const healthRecords = () => database.get<HealthRecord>('health_records');

export async function getNoteByDate(date: string): Promise<Note | null> {
  const found = await notes().query(Q.where('date', date)).fetch();
  return found[0] ?? null;
}

/**
 * Salva o diário de um dia e regrava os eventos extraídos dele.
 *
 * A nota é a fonte de verdade do texto; transações e registros de saúde são
 * derivados do parser e recriados a cada salvamento (re-save idempotente).
 * Registros já sincronizados são preservados para não gerar duplicata no
 * servidor — o id local é o `source_id` remoto.
 */
export async function saveJournal(date: string, content: string): Promise<void> {
  const parsed = parseNoteContent(content);

  await database.write(async () => {
    let note = await getNoteByDate(date);

    if (note) {
      await note.update((n) => {
        n.content = content;
      });
    } else {
      note = await notes().create((n) => {
        n.date = date;
        n.content = content;
        n.isCollection = false;
      });
    }

    const noteId = note.id;

    // Apaga apenas o que ainda não foi enviado; o que já foi sincronizado
    // permanece como registro do que o servidor recebeu.
    const staleTx = await transactions()
      .query(Q.where('note_id', noteId), Q.where('push_status', 'pending'))
      .fetch();
    const staleHealth = await healthRecords()
      .query(Q.where('note_id', noteId), Q.where('push_status', 'pending'))
      .fetch();

    const syncedTx = await transactions()
      .query(Q.where('note_id', noteId), Q.where('push_status', 'synced'))
      .fetch();
    const syncedHealth = await healthRecords()
      .query(Q.where('note_id', noteId), Q.where('push_status', 'synced'))
      .fetch();

    await Promise.all([
      ...staleTx.map((t) => t.destroyPermanently()),
      ...staleHealth.map((h) => h.destroyPermanently()),
    ]);

    const seenTx = new Set(syncedTx.map((t) => `${t.value}|${t.category}|${t.description ?? ''}`));
    const seenHealth = new Set(syncedHealth.map((h) => `${h.type}|${h.distance ?? ''}|${h.rpe ?? ''}`));

    const creations: unknown[] = [];

    for (const line of parsed) {
      if (line.transaction) {
        const key = `${line.transaction.value}|${line.transaction.category}|${line.transaction.description ?? ''}`;
        if (seenTx.has(key)) continue;
        creations.push(
          transactions().prepareCreate((t) => {
            // ID gerado aqui vira o source_id no servidor → push idempotente
            t._raw.id = Crypto.randomUUID();
            t.value = line.transaction!.value;
            t.description = line.transaction!.description;
            t.category = line.transaction!.category;
            t.method = line.transaction!.method;
            t.date = date;
            t.pushStatus = 'pending';
            (t._raw as any).note_id = noteId;
          })
        );
      }

      if (line.health) {
        const key = `${line.health.type}|${line.health.distance ?? ''}|${line.health.rpe ?? ''}`;
        if (seenHealth.has(key)) continue;
        creations.push(
          healthRecords().prepareCreate((h) => {
            h._raw.id = Crypto.randomUUID();
            h.type = line.health!.type;
            h.distance = line.health!.distance;
            h.time = line.health!.time;
            h.rpe = line.health!.rpe;
            h.date = date;
            h.pushStatus = 'pending';
            (h._raw as any).note_id = noteId;
          })
        );
      }
    }

    if (creations.length > 0) {
      await database.batch(...(creations as never[]));
    }
  });
}
