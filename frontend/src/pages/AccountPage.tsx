import { useEffect, useState } from 'react'
import { useAuthStore } from '@/store/authStore'
import {
  useProfile, useUpdateProfile, useChangePassword, useDeleteAccount, downloadMyData,
} from '@/hooks/useAccount'
import { browserTimezone, timezoneOptions } from '@/types/account'
import { toast, apiErrorMessage } from '@/lib/toast'
import { NotificationsSection } from './NotificationsSection'
import styles from './AccountPage.module.css'

// Minha conta: perfil (nome, fuso), notificações, senha, e os direitos da LGPD —
// levar os próprios dados e apagar a conta de verdade.
export function AccountPage() {
  const logout = useAuthStore(s => s.logout)
  const profile = useProfile()

  return (
    <div className={styles.page}>
      <div>
        <h1 className={styles.pageTitle}>Minha conta</h1>
        {profile.data && <p className={styles.pageSubtitle}>{profile.data.email}</p>}
      </div>

      {profile.isLoading && <p className={styles.hint}>Carregando…</p>}
      {profile.isError && <p className={styles.hint}>Não foi possível carregar seu perfil.</p>}
      {profile.data && (
        <ProfileSection name={profile.data.name} timezone={profile.data.timezone} />
      )}

      <NotificationsSection />
      <PasswordSection onChanged={logout} />
      <DataSection />
      <DeleteSection onDeleted={logout} />
    </div>
  )
}

function ProfileSection({ name: initialName, timezone: initialTz }: { name: string; timezone: string }) {
  const update = useUpdateProfile()
  const [name, setName] = useState(initialName)
  const [timezone, setTimezone] = useState(initialTz)
  const detected = browserTimezone()

  useEffect(() => {
    setName(initialName)
    setTimezone(initialTz)
  }, [initialName, initialTz])

  const changed = name.trim() !== initialName || timezone !== initialTz

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    update.mutate(
      { name: name.trim(), timezone },
      {
        onSuccess: () => toast.success('Perfil atualizado'),
        onError: (err) => {
          const msg = apiErrorMessage(err, '')
          toast.error(msg === 'invalid timezone'
            ? 'Esse fuso horário não é válido.'
            : 'Não foi possível salvar o perfil.')
        },
      }
    )
  }

  return (
    <form className={`glass-card ${styles.section}`} onSubmit={handleSubmit}>
      <h2 className={styles.sectionTitle}>Perfil</h2>

      <div className={styles.field}>
        <label htmlFor="acc-name" className={styles.label}>Nome</label>
        <input id="acc-name" className="input" value={name} onChange={e => setName(e.target.value)} required />
      </div>

      <div className={styles.field}>
        <label htmlFor="acc-tz" className={styles.label}>Fuso horário</label>
        <select id="acc-tz" className="input" value={timezone} onChange={e => setTimezone(e.target.value)}>
          {timezoneOptions(initialTz).map(tz => (
            <option key={tz} value={tz}>{tz.replace(/_/g, ' ')}</option>
          ))}
        </select>
        <p className={styles.hint}>
          Define quando o seu dia começa e termina nos hábitos, finanças e estudos.
        </p>
        {detected !== timezone && (
          <div className={styles.inlineRow}>
            <span className={styles.hint}>Seu aparelho está em {detected.replace(/_/g, ' ')}.</span>
            <button type="button" className="btn btn-ghost" onClick={() => setTimezone(detected)}>
              Usar este
            </button>
          </div>
        )}
      </div>

      <div className={styles.actions}>
        <button type="submit" className="btn btn-primary" disabled={!changed || update.isPending}>
          {update.isPending ? 'Salvando…' : 'Salvar perfil'}
        </button>
      </div>
    </form>
  )
}

function PasswordSection({ onChanged }: { onChanged: () => void }) {
  const change = useChangePassword()
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState('')

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    if (next.length < 8) return setError('A nova senha deve ter pelo menos 8 caracteres')
    if (next !== confirm) return setError('As senhas novas não conferem')

    change.mutate(
      { current_password: current, new_password: next },
      {
        onSuccess: () => {
          // O servidor encerra todas as sessões ao trocar a senha.
          toast.success('Senha alterada. Entre de novo com a senha nova.')
          onChanged()
        },
        onError: (err) => {
          const msg = apiErrorMessage(err, '')
          setError(msg === 'current password is incorrect'
            ? 'A senha atual está incorreta.'
            : 'Não foi possível trocar a senha.')
        },
      }
    )
  }

  return (
    <form className={`glass-card ${styles.section}`} onSubmit={handleSubmit}>
      <h2 className={styles.sectionTitle}>Senha</h2>
      <p className={styles.hint}>Ao trocar a senha, você sai de todos os aparelhos.</p>

      <div className={styles.field}>
        <label htmlFor="acc-current" className={styles.label}>Senha atual</label>
        <input id="acc-current" type="password" className="input" autoComplete="current-password"
          value={current} onChange={e => setCurrent(e.target.value)} required />
      </div>
      <div className={styles.field}>
        <label htmlFor="acc-new" className={styles.label}>Nova senha</label>
        <input id="acc-new" type="password" className="input" autoComplete="new-password"
          placeholder="Mínimo 8 caracteres" value={next} onChange={e => setNext(e.target.value)} required />
      </div>
      <div className={styles.field}>
        <label htmlFor="acc-confirm" className={styles.label}>Repita a nova senha</label>
        <input id="acc-confirm" type="password" className="input" autoComplete="new-password"
          value={confirm} onChange={e => setConfirm(e.target.value)} required />
      </div>

      {error && <p className={styles.errorText} role="alert">{error}</p>}

      <div className={styles.actions}>
        <button type="submit" className="btn btn-primary" disabled={change.isPending}>
          {change.isPending ? 'Salvando…' : 'Trocar senha'}
        </button>
      </div>
    </form>
  )
}

function DataSection() {
  const [downloading, setDownloading] = useState(false)

  const handleDownload = async () => {
    setDownloading(true)
    try {
      await downloadMyData()
    } catch {
      toast.error('Não foi possível gerar o arquivo. Tente novamente.')
    } finally {
      setDownloading(false)
    }
  }

  return (
    <section className={`glass-card ${styles.section}`}>
      <h2 className={styles.sectionTitle}>Seus dados</h2>
      <p className={styles.hint}>
        Baixe tudo o que o MyRoutine guarda sobre você — hábitos, tarefas, finanças, saúde e
        estudos — num arquivo JSON, que qualquer programa consegue abrir.
      </p>
      <div className={styles.actions}>
        <button type="button" className="btn btn-ghost" onClick={handleDownload} disabled={downloading}>
          {downloading ? 'Preparando…' : 'Baixar meus dados'}
        </button>
      </div>
    </section>
  )
}

function DeleteSection({ onDeleted }: { onDeleted: () => void }) {
  const del = useDeleteAccount()
  const [confirming, setConfirming] = useState(false)
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')

  const handleDelete = (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    del.mutate(password, {
      onSuccess: () => {
        toast.info('Sua conta e todos os seus dados foram apagados.')
        onDeleted()
      },
      onError: (err) => {
        const msg = apiErrorMessage(err, '')
        setError(msg === 'password is incorrect'
          ? 'Senha incorreta — nada foi apagado.'
          : 'Não foi possível apagar a conta. Nada foi apagado.')
      },
    })
  }

  return (
    <section className={`glass-card ${styles.section} ${styles.danger}`}>
      <h2 className={styles.sectionTitle}>Apagar conta</h2>
      <p className={styles.hint}>
        Apaga sua conta e <strong>todos</strong> os seus dados, de verdade e para sempre — não dá
        para desfazer. Se quiser guardar uma cópia, baixe seus dados antes.
      </p>

      {!confirming ? (
        <div className={styles.actions}>
          <button type="button" className="btn btn-danger" onClick={() => setConfirming(true)}>
            Quero apagar minha conta
          </button>
        </div>
      ) : (
        <form onSubmit={handleDelete} className={styles.confirmForm}>
          <div className={styles.field}>
            <label htmlFor="acc-delete-pw" className={styles.label}>Digite sua senha para confirmar</label>
            <input id="acc-delete-pw" type="password" className="input" autoComplete="current-password"
              value={password} onChange={e => setPassword(e.target.value)} required />
          </div>
          {error && <p className={styles.errorText} role="alert">{error}</p>}
          <div className={styles.actions}>
            <button type="submit" className="btn btn-danger" disabled={del.isPending}>
              {del.isPending ? 'Apagando…' : 'Apagar definitivamente'}
            </button>
            <button type="button" className="btn btn-ghost" onClick={() => {
              setConfirming(false)
              setPassword('')
              setError('')
            }}>
              Cancelar
            </button>
          </div>
        </form>
      )}
    </section>
  )
}
