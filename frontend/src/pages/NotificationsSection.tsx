import { useEffect, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import {
  usePushConfig, useNotificationSettings, useUpdateNotificationSettings, notificationKeys,
  type NotificationSettings,
} from '@/hooks/useNotifications'
import {
  pushSupport, currentSubscription, subscribeThisDevice, unsubscribeThisDevice, PushPermissionDenied,
} from '@/lib/push'
import { toast } from '@/lib/toast'
import styles from './AccountPage.module.css'

const LEAD_OPTIONS = [5, 10, 15, 30, 60]

// Notificações: ligar neste aparelho e escolher o que receber. Poucas e
// úteis (product-constitution): lembrete de compromisso, resumo da manhã e,
// se a pessoa quiser, resumo da noite. Nunca uma por hábito.
export function NotificationsSection() {
  const config = usePushConfig()
  const view = useNotificationSettings()
  const support = pushSupport()

  return (
    <section className={`glass-card ${styles.section}`}>
      <h2 className={styles.sectionTitle}>Notificações</h2>

      {config.data && !config.data.enabled ? (
        <p className={styles.hint}>As notificações ainda não estão disponíveis.</p>
      ) : (
        <>
          <DeviceToggle support={support} vapidKey={config.data?.vapid_public_key} />
          {view.data && <SettingsForm initial={view.data.settings} devices={view.data.devices} />}
        </>
      )}
    </section>
  )
}

function DeviceToggle({ support, vapidKey }: { support: ReturnType<typeof pushSupport>; vapidKey?: string }) {
  const qc = useQueryClient()
  const [subscribed, setSubscribed] = useState<boolean | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (support !== 'ok') return
    currentSubscription().then(sub => setSubscribed(sub !== null), () => setSubscribed(false))
  }, [support])

  if (support === 'needs-install') {
    return (
      <p className={styles.hint}>
        No iPhone, as notificações só funcionam com o MyRoutine instalado: no Safari, toque em
        Compartilhar → <strong>Adicionar à Tela de Início</strong> e abra o app por lá.
      </p>
    )
  }
  if (support === 'unsupported') {
    return <p className={styles.hint}>Este navegador não mostra notificações.</p>
  }

  const toggle = async () => {
    setError('')
    setBusy(true)
    try {
      if (subscribed) {
        await unsubscribeThisDevice()
        setSubscribed(false)
        toast.info('Notificações desligadas neste aparelho.')
      } else {
        if (!vapidKey) return
        await subscribeThisDevice(vapidKey)
        setSubscribed(true)
        toast.success('Notificações ligadas neste aparelho.')
      }
      void qc.invalidateQueries({ queryKey: notificationKeys.settings })
    } catch (err) {
      setError(err instanceof PushPermissionDenied
        ? 'As notificações estão bloqueadas para o MyRoutine. Libere nos ajustes do aparelho ou do navegador.'
        : 'Não foi possível mudar as notificações deste aparelho.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <div className={styles.inlineRow}>
        <span className={styles.hint}>
          {subscribed ? 'Ligadas neste aparelho.' : 'Desligadas neste aparelho.'}
        </span>
        <button type="button" className={subscribed ? 'btn btn-ghost' : 'btn btn-primary'}
          onClick={toggle} disabled={busy || subscribed === null}>
          {busy ? 'Aguarde…' : subscribed ? 'Desligar neste aparelho' : 'Ligar neste aparelho'}
        </button>
      </div>
      {error && <p className={styles.errorText} role="alert">{error}</p>}
    </>
  )
}

function SettingsForm({ initial, devices }: { initial: NotificationSettings; devices: number }) {
  const update = useUpdateNotificationSettings()
  const [s, setS] = useState(initial)
  useEffect(() => setS(initial), [initial])

  const set = <K extends keyof NotificationSettings>(key: K, value: NotificationSettings[K]) =>
    setS(prev => ({ ...prev, [key]: value }))
  const changed = JSON.stringify(s) !== JSON.stringify(initial)

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    update.mutate(s, {
      onSuccess: () => toast.success('Preferências salvas'),
      onError: () => toast.error('Não foi possível salvar as preferências.'),
    })
  }

  return (
    <form onSubmit={handleSubmit} className={styles.confirmForm}>
      <p className={styles.hint}>
        {devices === 0
          ? 'Nenhum aparelho recebe notificações ainda.'
          : `Recebendo em ${devices} ${devices === 1 ? 'aparelho' : 'aparelhos'}.`}
      </p>

      <div className={styles.inlineRow}>
        <label className={styles.checkRow}>
          <input type="checkbox" checked={s.task_reminders}
            onChange={e => set('task_reminders', e.target.checked)} />
          Lembrete de compromissos do Planner
        </label>
        <select aria-label="Antecedência do lembrete" className={`input ${styles.compactInput}`}
          value={s.task_lead_minutes} disabled={!s.task_reminders}
          onChange={e => set('task_lead_minutes', Number(e.target.value))}>
          {LEAD_OPTIONS.map(m => <option key={m} value={m}>{m} min antes</option>)}
        </select>
      </div>

      <div className={styles.inlineRow}>
        <label className={styles.checkRow}>
          <input type="checkbox" checked={s.morning_digest}
            onChange={e => set('morning_digest', e.target.checked)} />
          Resumo da manhã
        </label>
        <input type="time" aria-label="Horário do resumo da manhã" className={`input ${styles.compactInput}`}
          value={s.morning_time} disabled={!s.morning_digest} required
          onChange={e => set('morning_time', e.target.value)} />
      </div>

      <div className={styles.inlineRow}>
        <label className={styles.checkRow}>
          <input type="checkbox" checked={s.evening_digest}
            onChange={e => set('evening_digest', e.target.checked)} />
          Resumo da noite
        </label>
        <input type="time" aria-label="Horário do resumo da noite" className={`input ${styles.compactInput}`}
          value={s.evening_time} disabled={!s.evening_digest} required
          onChange={e => set('evening_time', e.target.value)} />
      </div>

      <p className={styles.hint}>
        O resumo só chega quando há algo no dia — compromissos ou hábitos. Os horários seguem o
        seu fuso, definido no Perfil.
      </p>

      <div className={styles.actions}>
        <button type="submit" className="btn btn-primary" disabled={!changed || update.isPending}>
          {update.isPending ? 'Salvando…' : 'Salvar preferências'}
        </button>
      </div>
    </form>
  )
}
