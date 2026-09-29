import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { motion } from 'framer-motion'
import api from '@/services/api'
import styles from './AuthPages.module.css'

// Página aberta pelo link de redefinição que o admin gerou (?codigo=...).
// O link vale 1 hora e funciona uma vez; ao trocar a senha, todas as sessões
// da conta são encerradas no servidor.
type LinkState =
  | { kind: 'checking' }
  | { kind: 'valid'; email: string }
  | { kind: 'invalid' }
  | { kind: 'done' }

export function ResetPasswordPage() {
  const [searchParams] = useSearchParams()
  const code = searchParams.get('codigo') ?? ''

  const [state, setState] = useState<LinkState>(code ? { kind: 'checking' } : { kind: 'invalid' })
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!code) return
    let cancelled = false
    api
      .get(`/auth/password-resets/${encodeURIComponent(code)}`)
      .then(({ data }) => {
        if (!cancelled) setState({ kind: 'valid', email: data.email })
      })
      .catch(() => {
        if (!cancelled) setState({ kind: 'invalid' })
      })
    return () => {
      cancelled = true
    }
  }, [code])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (password.length < 8) {
      setError('A senha deve ter pelo menos 8 caracteres')
      return
    }
    if (password !== confirm) {
      setError('As senhas não conferem')
      return
    }
    setLoading(true)
    setError('')
    try {
      await api.post(`/auth/password-resets/${encodeURIComponent(code)}`, { password })
      setState({ kind: 'done' })
    } catch (err: any) {
      if (err.response?.status === 404) {
        setState({ kind: 'invalid' })
      } else if (err.response?.status === 429) {
        setError('Muitas tentativas seguidas. Espere um minuto e tente de novo.')
      } else {
        setError('Não foi possível trocar a senha. Tente novamente.')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className={styles.page}>
      <div className={styles.sphere1} />
      <div className={styles.sphere2} />

      <motion.div
        className={styles.card}
        initial={{ opacity: 0, y: 24, scale: 0.97 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        transition={{ duration: 0.4, ease: [0.34, 1.56, 0.64, 1] }}
      >
        <div className={styles.cardHeader}>
          <div className={styles.logoMark}>M</div>
          <h1 className={styles.title}>Criar nova senha</h1>
          {state.kind === 'valid' && (
            <p className={styles.subtitle}>Para a conta {state.email}</p>
          )}
        </div>

        {state.kind === 'checking' && (
          <p className={styles.notice} role="status">Conferindo o link…</p>
        )}

        {state.kind === 'invalid' && (
          <p className={styles.notice} role="alert">
            Este link não é mais válido — ele vale 1 hora e funciona uma vez. Peça um novo link
            a quem te convidou.
          </p>
        )}

        {state.kind === 'done' && (
          <p className={styles.notice} role="status">
            Senha alterada. Por segurança, você saiu de todos os aparelhos — entre de novo com a
            senha nova.
          </p>
        )}

        {state.kind === 'valid' && (
          <form onSubmit={handleSubmit} className={styles.form}>
            <div className={styles.field}>
              <label htmlFor="password" className={styles.label}>Nova senha</label>
              <input
                id="password"
                type="password"
                className="input"
                placeholder="Mínimo 8 caracteres"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoComplete="new-password"
                required
                minLength={8}
              />
            </div>

            <div className={styles.field}>
              <label htmlFor="confirm" className={styles.label}>Repita a nova senha</label>
              <input
                id="confirm"
                type="password"
                className="input"
                value={confirm}
                onChange={(e) => setConfirm(e.target.value)}
                autoComplete="new-password"
                required
              />
            </div>

            {error && (
              <div className={styles.error} role="alert">{error}</div>
            )}

            <button type="submit" className={`btn btn-primary ${styles.submitBtn}`} disabled={loading}>
              {loading ? <span className={styles.spinner} /> : 'Salvar nova senha'}
            </button>
          </form>
        )}

        <p className={styles.switchLink}>
          <Link to="/login">Ir para o login</Link>
        </p>
      </motion.div>
    </div>
  )
}
