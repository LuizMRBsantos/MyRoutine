import { useState } from 'react';
import { StyleSheet, TextInput, ScrollView, KeyboardAvoidingView, Platform } from 'react-native';
import { Text, View } from '@/components/Themed';
import { parseNoteContent } from '@/src/parser';

export default function DashboardHoje() {
  const [content, setContent] = useState('• Tarefa pendente\n$ 45 Almoço #alimentação\n🏃 Corrida 5km RPE 7\n- Reunião de alinhamento\n[[Projeto Alpha]]');

  const parsedLines = parseNoteContent(content);
  
  const finances = parsedLines.filter(l => l.transaction);
  const health = parsedLines.filter(l => l.health);
  const links = parsedLines.flatMap(l => l.links);

  return (
    <KeyboardAvoidingView 
      style={styles.container} 
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
    >
      <ScrollView contentContainerStyle={styles.scroll}>
        <View style={styles.header}>
          <Text style={styles.title}>Diário de Bordo</Text>
          <Text style={styles.subtitle}>{new Date().toLocaleDateString('pt-BR', { weekday: 'long', day: 'numeric', month: 'long' })}</Text>
        </View>

        <TextInput
          style={styles.editor}
          multiline
          placeholder="O que está na sua mente hoje? •, $, 🏃..."
          placeholderTextColor="#666"
          value={content}
          onChangeText={setContent}
          textAlignVertical="top"
        />

        {/* Preview Panel for Extractions (For Demonstration) */}
        <View style={styles.previewContainer}>
          <Text style={styles.previewTitle}>✨ Extrações Automáticas em Segundo Plano</Text>
          
          {finances.length > 0 && (
            <View style={styles.card}>
              <Text style={styles.cardTitle}>💰 Finanças</Text>
              {finances.map((line, i) => (
                <Text key={i} style={styles.cardText}>
                  R$ {line.transaction?.value.toFixed(2)} - {line.transaction?.category}
                </Text>
              ))}
            </View>
          )}

          {health.length > 0 && (
            <View style={styles.card}>
              <Text style={styles.cardTitle}>💪 Saúde & Performance</Text>
              {health.map((line, i) => (
                <Text key={i} style={styles.cardText}>
                  {line.health?.type === 'run' ? 'Corrida' : line.health?.type} - {line.health?.distance}km (RPE: {line.health?.rpe})
                </Text>
              ))}
            </View>
          )}

          {links.length > 0 && (
            <View style={styles.card}>
              <Text style={styles.cardTitle}>🔗 Conexões (Coleções)</Text>
              {links.map((link, i) => (
                <Text key={i} style={styles.linkText}>[[{link}]]</Text>
              ))}
            </View>
          )}
        </View>
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#0A0A0A', // Dark mode premium
  },
  scroll: {
    padding: 24,
  },
  header: {
    marginBottom: 32,
    marginTop: 20,
    backgroundColor: 'transparent',
  },
  title: {
    fontSize: 34,
    fontWeight: '800',
    color: '#FFF',
    letterSpacing: -1,
  },
  subtitle: {
    fontSize: 16,
    color: '#888',
    marginTop: 4,
    textTransform: 'capitalize',
  },
  editor: {
    fontSize: 18,
    color: '#E0E0E0',
    lineHeight: 28,
    minHeight: 250,
    backgroundColor: 'rgba(255,255,255,0.03)',
    borderRadius: 16,
    padding: 20,
    marginBottom: 24,
  },
  previewContainer: {
    backgroundColor: 'transparent',
    gap: 16,
  },
  previewTitle: {
    fontSize: 14,
    fontWeight: '600',
    color: '#A0A0A0',
    textTransform: 'uppercase',
    letterSpacing: 1,
    marginBottom: 8,
  },
  card: {
    backgroundColor: 'rgba(255,255,255,0.05)',
    padding: 16,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.1)',
  },
  cardTitle: {
    fontSize: 16,
    fontWeight: 'bold',
    color: '#FFF',
    marginBottom: 8,
  },
  cardText: {
    color: '#CCC',
    fontSize: 15,
    marginBottom: 4,
  },
  linkText: {
    color: '#4DA6FF',
    fontSize: 15,
    fontWeight: '600',
    marginBottom: 4,
  },
});
