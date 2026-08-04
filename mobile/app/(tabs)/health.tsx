import { StyleSheet, ScrollView } from 'react-native';
import { Text, View } from '@/components/Themed';

export default function HealthHub() {
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
          {/* Running */}
          <View style={[styles.statBox, styles.statBoxRun]}>
            <View style={styles.emojiContainer}>
              <Text style={styles.statEmoji}>🏃</Text>
            </View>
            <Text style={styles.statValue}>15 <Text style={styles.unit}>km</Text></Text>
            <Text style={styles.statLabel}>Corrida</Text>
          </View>

          {/* Cycling */}
          <View style={[styles.statBox, styles.statBoxBike]}>
            <View style={styles.emojiContainer}>
              <Text style={styles.statEmoji}>🚴</Text>
            </View>
            <Text style={styles.statValue}>40 <Text style={styles.unit}>km</Text></Text>
            <Text style={styles.statLabel}>Ciclismo</Text>
          </View>

          {/* Swimming */}
          <View style={[styles.statBox, styles.statBoxSwim]}>
            <View style={styles.emojiContainer}>
              <Text style={styles.statEmoji}>🏊</Text>
            </View>
            <Text style={styles.statValue}>2 <Text style={styles.unit}>km</Text></Text>
            <Text style={styles.statLabel}>Natação</Text>
          </View>
        </View>
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
});
