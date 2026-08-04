import { useEffect, useRef, useState } from 'react';
import {
  StyleSheet, TextInput, ScrollView, KeyboardAvoidingView, Platform, Pressable,
  ActivityIndicator,
} from 'react-native';
import { Text, View } from '@/components/Themed';
import { parseNoteContent } from '@/src/parser';
import { getNoteByDate, saveJournal, todayStr } from '@/src/journal/store';
import { pushPendingRecords } from '@/src/sync/pushSync';

type SaveState = 'idle' | 'saving' | 'saved';

export default function DashboardHoje() {
  const date = todayStr();
  const [content, setContent] = useState('');
  const [loaded, setLoaded] = useState(false);
  const [saveState, setSaveState] = useState<SaveState>('idle');
  const [syncing, setSyncing] = useState(false);
  const [syncMessage, setSyncMessage] = useState<string | null>(null);
  const saveTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Carrega o diário do dia gravado no aparelho
  useEffect(() => {
    let cancelled = false;
    (async () => {
      const note = await getNoteByDate(date);
      if (cancelled) return;
      setContent(note?.content ?? '');
      setLoaded(true);
    })();
    return () => { cancelled = true; };
  }, [date]);

  // Autosave com debounce — escrever no diário nunca deve exigir um botão
  useEffect(() => {
    if (!loaded) return;
    if (saveTimer.current) clearTimeout(saveTimer.current);

    setSaveState('saving');
    saveTimer.current = setTimeout(async () => {
      await saveJournal(date, content);
      setSaveState('saved');
    }, 800);

    return () => { if (saveTimer.current) clearTimeout(saveTimer.current); };
  }, [content, loaded, date]);

  const handleSync = async () => {
    setSyncing(true);
    setSyncMessage(null);
    try {
      if (saveTimer.current) clearTimeout(saveTimer.current);
      await saveJournal(date, content);
      const result = await pushPendingRecords();

      const parts: string[] = [];
      if (result.transactionsSynced) parts.push(`${result.transactionsSynced} transação(ões)`);
      if (result.workoutsSynced) parts.push(`${result.workoutsSynced} treino(s)`);
      if (parts.length === 0) {
        setSyncMessage(
          result.skipped > 0
            ? `${result.skipped} treino(s) sem hábito correspondente no servidor`
            : 'Tudo já estava sincronizado'
        );
      } else {
        setSyncMessage(`Enviado: ${parts.join(' e ')}`);
      }
    } catch (e) {
      setSyncMessage(e instanceof Error ? e.message : 'Falha ao sincronizar');
    } finally {
      setSyncing(false);
    }
  };

  const parsedLines = parseNoteContent(content);
  const finances = parsedLines.filter((l) => l.transaction);
  const health = parsedLines.filter((l) => l.health);
  const links = parsedLines.flatMap((l) => l.links);

  return (
    <KeyboardAvoidingView
      style={styles.container}
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
    >
      <ScrollView contentContainerStyle={styles.scroll}>
        <View style={styles.header}>
          <Text style={styles.title}>Diário de Bordo</Text>
          <Text style={styles.subtitle}>
            {new Date().toLocaleDateString('pt-BR', { weekday: 'long', day: 'numeric', month: 'long' })}
          </Text>
        </View>

        <TextInput
          style={styles.editor}
          multiline
          placeholder="O que está na sua mente hoje? •, $, 🏃..."
          placeholderTextColor="#666"
          value={content}
          onChangeText={setContent}
          textAlignVertical="top"
          editable={loaded}
        />

        <View style={styles.actionRow}>
          <Text style={styles.saveState}>
            {saveState === 'saving' ? 'Salvando…' : saveState === 'saved' ? 'Salvo no aparelho' : ''}
          </Text>
          <Pressable
            style={[styles.syncButton, syncing && styles.syncButtonDisabled]}
            onPress={handleSync}
            disabled={syncing}
          >
            {syncing
              ? <ActivityIndicator color="#FFF" size="small" />
              : <Text style={styles.syncButtonText}>Sincronizar</Text>}
          </Pressable>
        </View>

        {syncMessage && <Text style={styles.syncMessage}>{syncMessage}</Text>}

        {/* Prévia do que o parser extraiu deste texto */}
        <View style={styles.previewContainer}>
          <Text style={styles.previewTitle}>✨ Extrações Automáticas em Segundo Plano</Text>

          {finances.length > 0 && (
            <View style={styles.card}>
              <Text style={styles.cardTitle}>💰 Finanças</Text>
              {finances.map((line, i) => (
                <Text key={i} style={styles.cardText}>
                  R$ {line.transaction?.value.toFixed(2)} — {line.transaction?.description} ({line.transaction?.category})
                </Text>
              ))}
            </View>
          )}

          {health.length > 0 && (
            <View style={styles.card}>
              <Text style={styles.cardTitle}>💪 Saúde & Performance</Text>
              {health.map((line, i) => (
                <Text key={i} style={styles.cardText}>
                  {line.health?.type === 'run' ? 'Corrida' : line.health?.type === 'bike' ? 'Ciclismo' : 'Natação'}
                  {line.health?.distance != null ? ` — ${line.health.distance}km` : ''}
                  {line.health?.rpe != null ? ` (RPE: ${line.health.rpe})` : ''}
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
    backgroundColor: '#0A0A0A',
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
  },
  actionRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginTop: 12,
    marginBottom: 8,
    backgroundColor: 'transparent',
  },
  saveState: {
    color: '#666',
    fontSize: 13,
  },
  syncButton: {
    backgroundColor: '#0071E3',
    paddingHorizontal: 20,
    paddingVertical: 10,
    borderRadius: 10,
    minWidth: 120,
    alignItems: 'center',
  },
  syncButtonDisabled: { opacity: 0.6 },
  syncButtonText: { color: '#FFF', fontWeight: '600', fontSize: 15 },
  syncMessage: {
    color: '#A0A0A0',
    fontSize: 13,
    marginBottom: 16,
  },
  previewContainer: {
    backgroundColor: 'transparent',
    gap: 16,
    marginTop: 16,
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
