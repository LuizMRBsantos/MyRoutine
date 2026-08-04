import { StyleSheet, ScrollView } from 'react-native';
import { Text, View } from '@/components/Themed';

export default function FinanceHub() {
  return (
    <ScrollView style={styles.container}>
      <View style={styles.header}>
        <Text style={styles.title}>Fluxo de Caixa</Text>
        <Text style={styles.subtitle}>Visão Geral do Mês</Text>
      </View>

      {/* Mock Progress Bars for Budgets */}
      <View style={styles.card}>
        <Text style={styles.cardTitle}>Orçamentos</Text>
        
        <View style={styles.budgetRow}>
          <Text style={styles.budgetText}>Alimentação (R$ 800 / R$ 1000)</Text>
          <View style={styles.progressBarBg}>
            <View style={[styles.progressBarFill, { width: '80%', backgroundColor: '#FF8A65' }]} />
          </View>
        </View>

        <View style={styles.budgetRow}>
          <Text style={styles.budgetText}>Transporte (R$ 150 / R$ 300)</Text>
          <View style={styles.progressBarBg}>
            <View style={[styles.progressBarFill, { width: '50%', backgroundColor: '#64B5F6' }]} />
          </View>
        </View>
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
    marginBottom: 32,
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
  },
  cardTitle: {
    fontSize: 18,
    fontWeight: '700',
    color: '#FFF',
    marginBottom: 20,
  },
  budgetRow: {
    marginBottom: 16,
    backgroundColor: 'transparent',
  },
  budgetText: {
    color: '#CCC',
    fontSize: 14,
    marginBottom: 8,
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
  },
});
