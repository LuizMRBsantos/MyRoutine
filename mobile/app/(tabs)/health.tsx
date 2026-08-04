import { StyleSheet, ScrollView } from 'react-native';
import { Q } from '@nozbe/watermelondb';
import { Text, View } from '@/components/Themed';
import { database } from '@/src/db/database';
import { useObservableQuery } from '@/src/db/hooks';
import type HealthRecord from '@/src/db/models/HealthRecord';
import { todayStr } from '@/src/journal/store';

const MODALITIES = [
  { type: 'run', emoji: '🏃', label: 'Corrida', boxStyle: 'statBoxRun' as const },
  { type: 'bike', emoji: '🚴', label: 'Ciclismo', boxStyle: 'statBoxBike' as const },
  { type: 'swim', emoji: '🏊', label: 'Natação', boxStyle: 'statBoxSwim' as const },
];

function daysAgo(n: number): string {
  const d = new Date();
  d.setDate(d.getDate() - n);
  return todayStr(d);
}

export default function HealthHub() {
  const from = daysAgo(6);
  const to = todayStr();

  const records = useObservableQuery<HealthRecord>(
    () =>
      database
        .get<HealthRecord>('health_records')
        .query(Q.where('date', Q.gte(from)), Q.where('date', Q.lte(to)), Q.sortBy('date', Q.desc)),
    [from, to]
  );

  const volumeByType = records.reduce<Record<string, number>>((acc, r) => {
    acc[r.type] = (acc[r.type] ?? 0) + (r.distance ?? 0);
    return acc;
  }, {});

  const pending = records.filter((r) => r.pushStatus === 'pending').length;

  return (
    <ScrollView style={styles.container}>
      <View style={styles.header}>
        <Text style={styles.title}>Saúde & Corpo</Text>
        <Text style={styles.subtitle}>Estatísticas de alta performance</Text>
      </View>

      <View style={styles.card}>
        <View style={styles.cardHeader}>
          <Text style={styles.cardTitle}>Volume Semanal</Text>
          <Text style={styles.cardSubtitle}>Triathlon Training</Text>
        </View>

        <View style={styles.statsGrid}>
          {MODALITIES.map((m) => (
            <View key={m.type} style={[styles.statBox, styles[m.boxStyle]]}>
              <View style={styles.emojiContainer}>
                <Text style={styles.statEmoji}>{m.emoji}</Text>
              </View>
              <Text style={styles.statValue}>
                {(volumeByType[m.type] ?? 0).toFixed(volumeByType[m.type] ? 1 : 0)}{' '}
                <Text style={styles.unit}>km</Text>
              </Text>
              <Text style={styles.statLabel}>{m.label}</Text>
            </View>
          ))}
        </View>
      </View>

      <View style={styles.listCard}>
        <Text style={styles.listTitle}>Registros da semana</Text>
        {records.length === 0 ? (
          <Text style={styles.empty}>
            Nenhum treino esta semana. Escreva “🏃 Corrida 8km RPE 6” no diário.
          </Text>
        ) : (
          records.map((r) => {
            const modality = MODALITIES.find((m) => m.type === r.type);
            return (
              <View key={r.id} style={styles.recordRow}>
                <Text style={styles.recordTitle}>
                  {modality?.emoji ?? '•'} {modality?.label ?? r.type}
                </Text>
                <Text style={styles.recordMeta}>
                  {r.date}
                  {r.distance != null ? ` · ${r.distance} km` : ''}
                  {r.time != null ? ` · ${r.time} min` : ''}
                  {r.rpe != null ? ` · RPE ${r.rpe}` : ''}
                  {r.pushStatus === 'pending' ? ' · pendente' : ''}
                </Text>
              </View>
            );
          })
        )}
        {pending > 0 && (
          <Text style={styles.pendingNote}>
            {pending} registro(s) aguardando sincronização com o servidor.
          </Text>
        )}
      </View>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#050505', // Deep premium dark mode
    padding: 24,
  },
  header: {
    marginTop: 20,
    marginBottom: 40,
    backgroundColor: 'transparent',
  },
  title: {
    fontSize: 36,
    fontWeight: '900',
    color: '#FFFFFF',
    letterSpacing: -1.5,
  },
  subtitle: {
    fontSize: 16,
    fontWeight: '500',
    color: '#666',
    marginTop: 6,
    textTransform: 'uppercase',
    letterSpacing: 1.5,
  },
  card: {
    backgroundColor: '#111111',
    borderRadius: 24,
    padding: 24,
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.06)',
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 10 },
    shadowOpacity: 0.3,
    shadowRadius: 20,
  },
  cardHeader: {
    backgroundColor: 'transparent',
    marginBottom: 24,
    alignItems: 'center',
  },
  cardTitle: {
    fontSize: 22,
    fontWeight: '800',
    color: '#FFF',
    letterSpacing: -0.5,
  },
  cardSubtitle: {
    fontSize: 14,
    color: '#888',
    marginTop: 4,
  },
  statsGrid: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    backgroundColor: 'transparent',
    gap: 12,
  },
  statBox: {
    flex: 1,
    alignItems: 'center',
    paddingVertical: 20,
    paddingHorizontal: 10,
    borderRadius: 16,
    backgroundColor: 'rgba(255,255,255,0.03)',
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.05)',
  },
  statBoxRun: {
    backgroundColor: 'rgba(255, 100, 100, 0.05)',
    borderColor: 'rgba(255, 100, 100, 0.15)',
  },
  statBoxBike: {
    backgroundColor: 'rgba(100, 200, 255, 0.05)',
    borderColor: 'rgba(100, 200, 255, 0.15)',
  },
  statBoxSwim: {
    backgroundColor: 'rgba(100, 255, 200, 0.05)',
    borderColor: 'rgba(100, 255, 200, 0.15)',
  },
  emojiContainer: {
    width: 48,
    height: 48,
    borderRadius: 24,
    backgroundColor: 'rgba(255,255,255,0.1)',
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 12,
  },
  statEmoji: {
    fontSize: 24,
  },
  statValue: {
    fontSize: 24,
    fontWeight: '900',
    color: '#FFF',
    letterSpacing: -1,
  },
  unit: {
    fontSize: 14,
    fontWeight: '600',
    color: '#888',
  },
  statLabel: {
    fontSize: 13,
    fontWeight: '600',
    color: '#AAA',
    marginTop: 6,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
  },
  listCard: {
    backgroundColor: '#111111',
    borderRadius: 24,
    padding: 24,
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.06)',
    marginTop: 16,
    marginBottom: 24,
  },
  listTitle: {
    fontSize: 18,
    fontWeight: '700',
    color: '#FFF',
    marginBottom: 12,
  },
  recordRow: {
    paddingVertical: 10,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: 'rgba(255,255,255,0.08)',
    backgroundColor: 'transparent',
  },
  recordTitle: { color: '#EEE', fontSize: 15 },
  recordMeta: { color: '#777', fontSize: 12, marginTop: 2 },
  empty: { color: '#777', fontSize: 14, lineHeight: 20 },
  pendingNote: { color: '#777', fontSize: 12, marginTop: 12 },
});
