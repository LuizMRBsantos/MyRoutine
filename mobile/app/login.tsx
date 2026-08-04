import { useState } from 'react';
import {
  StyleSheet, TextInput, Pressable, KeyboardAvoidingView, Platform, ActivityIndicator,
} from 'react-native';
import { Text, View } from '@/components/Themed';
import { useAuth } from '@/src/api/AuthContext';
import { API_BASE_URL } from '@/src/api/client';

export default function LoginScreen() {
  const { signIn } = useAuth();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  const handleSubmit = async () => {
    setError(null);
    setPending(true);
    try {
      await signIn(email.trim(), password);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Não foi possível entrar');
    } finally {
      setPending(false);
    }
  };

  return (
    <KeyboardAvoidingView
      style={styles.container}
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
    >
      <View style={styles.inner}>
        <Text style={styles.title}>MyRoutine</Text>
        <Text style={styles.subtitle}>Entre para sincronizar seu diário</Text>

        <TextInput
          style={styles.input}
          placeholder="E-mail"
          placeholderTextColor="#666"
          autoCapitalize="none"
          keyboardType="email-address"
          autoComplete="email"
          value={email}
          onChangeText={setEmail}
        />
        <TextInput
          style={styles.input}
          placeholder="Senha"
          placeholderTextColor="#666"
          secureTextEntry
          value={password}
          onChangeText={setPassword}
        />

        {error && <Text style={styles.error}>{error}</Text>}

        <Pressable
          style={[styles.button, pending && styles.buttonDisabled]}
          onPress={handleSubmit}
          disabled={pending}
        >
          {pending
            ? <ActivityIndicator color="#FFF" />
            : <Text style={styles.buttonText}>Entrar</Text>}
        </Pressable>

        <Text style={styles.hint}>Servidor: {API_BASE_URL}</Text>
      </View>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: '#0A0A0A' },
  inner: {
    flex: 1,
    justifyContent: 'center',
    padding: 24,
    backgroundColor: 'transparent',
    gap: 12,
  },
  title: { fontSize: 34, fontWeight: '800', color: '#FFF', letterSpacing: -1 },
  subtitle: { fontSize: 16, color: '#888', marginBottom: 20 },
  input: {
    backgroundColor: 'rgba(255,255,255,0.05)',
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.1)',
    borderRadius: 12,
    padding: 16,
    fontSize: 16,
    color: '#E0E0E0',
  },
  button: {
    backgroundColor: '#0071E3',
    borderRadius: 12,
    padding: 16,
    alignItems: 'center',
    marginTop: 8,
  },
  buttonDisabled: { opacity: 0.6 },
  buttonText: { color: '#FFF', fontSize: 16, fontWeight: '600' },
  error: { color: '#FF9F0A', fontSize: 14 },
  hint: { color: '#555', fontSize: 12, marginTop: 16, textAlign: 'center' },
});
