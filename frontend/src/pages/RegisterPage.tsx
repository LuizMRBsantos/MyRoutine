import { useEffect, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { motion } from 'framer-motion'
import { useAuthStore } from '@/store/authStore'
import api from '@/services/api'
import { browserTimezone } from '@/types/account'
import styles from './AuthPages.module.css'

// O cadastro é só por convite. O link do convite traz ?convite=<código>:
// - convite válido  → e-mail preenchido e travado (o convite vale só para ele);
// - convite inválido → aviso, sem formulário;
// - sem convite      → aviso de beta fechado, mas o formulário fica disponível,
//   porque é por ele que o admin (ADMIN_EMAILS) cria a primeira conta.
type InviteState =
  | { kind: 'none' }
  | { kind: 'checking' }
  | { kind: 'valid'; email: string }
  | { kind: 'invalid' }

const ERROR_MESSAGES: Record<string, string> = {
  'email already registered': 'Este e-mail já está cadastrado. Tente entrar.',
  'an invite is required to register':
    'O MyRoutine está em beta fechado: para criar uma conta você precisa de um convite.',
  'invite is invalid or expired':
    'Este convite não é mais válido. Peça um novo a quem te convidou.',
}

export function RegisterPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const inviteToken = searchParams.get('convite') ?? ''
  const { setAuth } = useAuthStore()

  const [invite, setInvite] = useState<InviteState>(
    inviteToken ? { kind: 'checking' } : { kind: 'none' }
  )
  const [form, setForm] = useState({ name: '', email: '', password: '' })
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!inviteToken) return
    let cancelled = false
    api
      .get(`/auth/invites/${encodeURIComponent(inviteToken)}`)
      .then(({ data }) => {
        if (cancelled) return
        setInvite({ kind: 'valid', email: data.email })
        setForm(f => ({ ...f, email: data.email }))
      })
      .catch(() => {
        if (!cancelled) setInvite({ kind: 'invalid' })
      })
    return () => {
      cancelled = true
    }
  }, [inviteToken])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (form.password.length < 8) {
      setError('A senha deve ter pelo menos 8 caracteres')
      return
    }
    setLoading(true)
    setError('')

    try {
      const { data } = await api.post('/auth/register', {
        ...form,
        timezone: browserTimezone(),
        invite_token: inviteToken || undefined,
      })
      setAuth(
        { accessToken: data.access_token, refreshToken: data.refresh_token },
        data.user
      )
      navigate('/')
    } catch (err: any) {
      const msg: string | undefined = err.response?.data?.error
      setError((msg && ERROR_MESSAGES[msg]) || 'Erro ao criar conta. Tente novamente.')
    } finally {
      setLoading(false)
    }
  }

  const emailLocked = invite.kind === 'valid'

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
          <h1 className={styles.title}>Criar sua conta</h1>
          <p className={styles.subtitle}>
            {emailLocked ? 'Você foi convidado para o MyRoutine' : 'Comece a controlar sua rotina hoje'}
          </p>
        </div>

        {invite.kind === 'checking' && (
          <p className={styles.notice} role="status">Conferindo seu convite…</p>
        )}

        {invite.kind === 'invalid' && (
          <p className={styles.notice} role="alert">
            Este convite não é mais válido — ele pode ter expirado, sido cancelado ou já usado.
            Peça um novo convite a quem te convidou.
          </p>
        )}

        {(invite.kind === 'none' || invite.kind === 'valid') && (
          <form onSubmit={handleSubmit} className={styles.form}>
            {invite.kind === 'none' && (
              <p className={styles.notice}>
                O MyRoutine está em beta fechado. Para criar uma conta, use o link do convite que
                você recebeu.
              </p>
            )}

            <div className={styles.field}>
              <label htmlFor="name" className={styles.label}>Nome</label>
              <input
                id="name"
                type="text"
                className="input"
                placeholder="Seu nome"
                value={form.name}
                onChange={(e) => setForm(f => ({ ...f, name: e.target.value }))}
                required
              />
            </div>

            <div className={styles.field}>
              <label htmlFor="email" className={styles.label}>E-mail</label>
              <input
                id="email"
                type="email"
                className={`input ${emailLocked ? styles.inputLocked : ''}`}
                placeholder="seu@email.com"
                value={form.email}
                onChange={(e) => setForm(f => ({ ...f, email: e.target.value }))}
                readOnly={emailLocked}
                aria-readonly={emailLocked}
                required
              />
            </div>

            <div className={styles.field}>
              <label htmlFor="password" className={styles.label}>Senha</label>
              <input
                id="password"
                type="password"
                className="input"
                placeholder="Mínimo 8 caracteres"
                value={form.password}
                onChange={(e) => setForm(f => ({ ...f, password: e.target.value }))}
                required
                minLength={8}
              />
            </div>

            {error && (
              <motion.div
                className={styles.error}
                role="alert"
                initial={{ opacity: 0, y: -8 }}
                animate={{ opacity: 1, y: 0 }}
              >
                {error}
              </motion.div>
            )}

            <button
              type="submit"
              className={`btn btn-primary ${styles.submitBtn}`}
              disabled={loading}
            >
              {loading ? <span className={styles.spinner} /> : 'Criar conta'}
            </button>
          </form>
        )}

        <p className={styles.switchLink}>
          Já tem conta? <Link to="/login">Entrar</Link>
        </p>
      </motion.div>
    </div>
  )
}
