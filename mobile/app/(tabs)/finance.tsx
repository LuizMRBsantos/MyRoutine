import { StyleSheet, ScrollView } from 'react-native';
import { Q } from '@nozbe/watermelondb';
import { Text, View } from '@/components/Themed';
import { database } from '@/src/db/database';
import { useObservableQuery } from '@/src/db/hooks';
import type Transaction from '@/src/db/models/Transaction';
import { todayStr } from '@/src/journal/store';

function monthStart(): string {
  const d = new Date();
  const pad = (n: number) => (n < 10 ? `0${n}` : `${n}`);
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-01`;
}

const brl = (value: number) =>
  value.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' });

export default function FinanceHub() {
  const from = monthStart();
  const to = todayStr();

  const transactions = useObservableQuery<Transaction>(
    () =>
      database
        .get<Transaction>('transactions')
        .query(Q.where('date', Q.gte(from)), Q.where('date', Q.lte(to)), Q.sortBy('date', Q.desc)),
    [from, to]
  );

  const total = transactions.reduce((sum, t) => sum + t.value, 0);
  const pending = transactions.filter((t) => t.pushStatus === 'pending').length;

  // Agrega por categoria a partir do que o parser extraiu do diário
  const byCategory = transactions.reduce<Record<string, number>>((acc, t) => {
    acc[t.category] = (acc[t.category] ?? 0) + t.value;
    return acc;
  }, {});
  const categories = Object.entries(byCategory).sort((a, b) => b[1] - a[1]);
  const maxCategory = Math.max(...categories.map(([, v]) => v), 1);

  return (
    <ScrollView style={styles.container}>
      <View style={styles.header}>
        <Text style={styles.title}>Fluxo de Caixa</Text>
        <Text style={styles.subtitle}>Do seu diário, este mês</Text>
      </View>

      <View style={styles.card}>
        <Text style={styles.cardLabel}>Total registrado</Text>
        <Text style={styles.bigValue}>{brl(total)}</Text>
        <Text style={styles.cardMeta}>
          {transactions.length} lançamento(s)
          {pending > 0 ? ` · ${pending} aguardando sincronização` : ' · tudo sincronizado'}
        </Text>
      </View>

      {categories.length > 0 && (
        <View style={styles.card}>
          <Text style={styles.cardTitle}>Por categoria</Text>
          {categories.map(([category, value]) => (
            <View key={category} style={styles.budgetRow}>
              <Text style={styles.budgetText}>
                {category} — {brl(value)}
              </Text>
              <View style={styles.progressBarBg}>
                <View
                  style={[styles.progressBarFill, { width: `${(value / maxCategory) * 100}%` }]}
                />
              </View>
            </View>
          ))}
        </View>
      )}

      <View style={styles.card}>
        <Text style={styles.cardTitle}>Lançamentos</Text>
        {transactions.length === 0 ? (
          <Text style={styles.empty}>
            Nada registrado ainda. Escreva “$ 45 Almoço #alimentacao” no diário.
          </Text>
        ) : (
          transactions.map((t) => (
            <View key={t.id} style={styles.txRow}>
              <View style={styles.txBody}>
                <Text style={styles.txTitle}>{t.description || t.category}</Text>
                <Text style={styles.txMeta}>
                  {t.date} · {t.category}
                  {t.pushStatus === 'pending' ? ' · pendente' : ''}
                </Text>
              </View>
              <Text style={styles.txValue}>{brl(t.value)}</Text>
            </View>
          ))
        )}
      </View>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#0A0A0A',
    padding: 24,
  },
  header: {
    marginBottom: 24,
    backgroundColor: 'transparent',
  },
  title: {
    fontSize: 32,
    fontWeight: '800',
    color: '#FFF',
    letterSpacing: -1,
  },
  subtitle: {
    fontSize: 16,
    color: '#888',
    marginTop: 4,
  },
  card: {
    backgroundColor: 'rgba(255,255,255,0.05)',
    padding: 20,
    borderRadius: 16,
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.1)',
    marginBottom: 16,
  },
  cardTitle: {
    fontSize: 18,
    fontWeight: '700',
    color: '#FFF',
    marginBottom: 16,
  },
  cardLabel: { fontSize: 13, color: '#888' },
  bigValue: {
    fontSize: 30,
    fontWeight: '800',
    color: '#FFF',
    marginTop: 4,
  },
  cardMeta: { fontSize: 13, color: '#888', marginTop: 6 },
  budgetRow: {
    marginBottom: 16,
    backgroundColor: 'transparent',
  },
  budgetText: {
    color: '#CCC',
    fontSize: 14,
    marginBottom: 8,
    textTransform: 'capitalize',
  },
  progressBarBg: {
    height: 8,
    backgroundColor: 'rgba(255,255,255,0.1)',
    borderRadius: 4,
    overflow: 'hidden',
  },
  progressBarFill: {
    height: '100%',
    borderRadius: 4,
    backgroundColor: '#0071E3',
  },
  txRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: 10,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: 'rgba(255,255,255,0.08)',
    backgroundColor: 'transparent',
  },
  txBody: { flex: 1, backgroundColor: 'transparent' },
  txTitle: { color: '#EEE', fontSize: 15 },
  txMeta: { color: '#777', fontSize: 12, marginTop: 2 },
  txValue: { color: '#FFF', fontSize: 15, fontWeight: '600' },
  empty: { color: '#777', fontSize: 14, lineHeight: 20 },
});
