import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import { useAuthStore } from '@/store/authStore'
import api from '@/services/api'
import styles from './AuthPages.module.css'

export function RegisterPage() {
  const navigate = useNavigate()
  const { setAuth } = useAuthStore()
  const [form, setForm] = useState({ name: '', email: '', password: '' })
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (form.password.length < 8) {
      setError('A senha deve ter pelo menos 8 caracteres')
      return
    }
    setLoading(true)
    setError('')

    try {
      const { data } = await api.post('/auth/register', form)
      setAuth(
        { accessToken: data.access_token, refreshToken: data.refresh_token },
        data.user
      )
      navigate('/')
    } catch (err: any) {
      const msg = err.response?.data?.error
      setError(msg === 'email already registered'
        ? 'Este e-mail já está cadastrado.'
        : 'Erro ao criar conta. Tente novamente.'
      )
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
          <h1 className={styles.title}>Criar sua conta</h1>
          <p className={styles.subtitle}>Comece a controlar sua rotina hoje</p>
        </div>

        <form onSubmit={handleSubmit} className={styles.form}>
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
              className="input"
              placeholder="seu@email.com"
              value={form.email}
              onChange={(e) => setForm(f => ({ ...f, email: e.target.value }))}
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

        <p className={styles.switchLink}>
          Já tem conta? <Link to="/login">Entrar</Link>
        </p>
      </motion.div>
    </div>
  )
}
