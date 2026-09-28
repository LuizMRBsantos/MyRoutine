import { useState } from 'react'
import { useAuthStore } from '@/store/authStore'
import { useInvites, useCreateInvite, useRevokeInvite } from '@/hooks/useInvites'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { toast, apiErrorMessage } from '@/lib/toast'
import { inviteLink, type CreatedInvite, type Invite, type InviteStatus } from '@/types/invite'
import styles from './InvitesPage.module.css'

const STATUS_LABEL: Record<InviteStatus, string> = {
  pending: '⏳ Aguardando',
  used: '✓ Usado',
  revoked: '⊘ Cancelado',
  expired: '⌛ Expirado',
}

const CREATE_ERRORS: Record<string, string> = {
  'invalid email': 'Esse e-mail não parece válido.',
  'email already registered': 'Essa pessoa já tem conta no MyRoutine.',
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit' })
}

function inviteMessage(invite: CreatedInvite, link: string): string {
  return (
    'Oi! Te convidei para testar o MyRoutine, o app que estou construindo para organizar ' +
    'hábitos, finanças, saúde e estudos.\n\n' +
    `Crie sua conta por este link (vale até ${formatDate(invite.expires_at)} e só funciona uma vez):\n` +
    link
  )
}

async function copy(text: string, what: string) {
  try {
    await navigator.clipboard.writeText(text)
    toast.success(`${what} copiado`)
  } catch {
    toast.error('Não consegui copiar. Selecione o texto e copie manualmente.')
  }
}

export function InvitesPage() {
  const isAdmin = useAuthStore(s => s.user?.is_admin === true)
  const invites = useInvites()
  const createInvite = useCreateInvite()
  const revokeInvite = useRevokeInvite()

  const [email, setEmail] = useState('')
  const [created, setCreated] = useState<CreatedInvite | null>(null)
  const [toRevoke, setToRevoke] = useState<Invite | null>(null)

  if (!isAdmin) {
    return (
      <div className={styles.page}>
        <h1 className={styles.pageTitle}>Convites</h1>
        <p className={styles.hint}>Esta página é só para administradores.</p>
      </div>
    )
  }

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault()
    createInvite.mutate(email, {
      onSuccess: (inv) => {
        setCreated(inv)
        setEmail('')
      },
      onError: (err) => {
        const msg = apiErrorMessage(err, '')
        toast.error(CREATE_ERRORS[msg] || 'Não foi possível criar o convite.')
      },
    })
  }

  const link = created ? inviteLink(created.token) : ''

  return (
    <div className={styles.page}>
      <div>
        <h1 className={styles.pageTitle}>Convites</h1>
        <p className={styles.pageSubtitle}>
          O MyRoutine está em beta fechado. Gere um link para cada pessoa e envie pelo seu
          e-mail ou WhatsApp. Cada link vale 7 dias, só para o e-mail convidado, e funciona uma vez.
        </p>
      </div>

      <form className={`glass-card ${styles.section}`} onSubmit={handleCreate}>
        <h2 className={styles.sectionTitle}>Novo convite</h2>
        <div className={styles.row}>
          <input
            type="email"
            className="input"
            placeholder="email@da-pessoa.com"
            aria-label="E-mail da pessoa convidada"
            value={email}
            onChange={e => setEmail(e.target.value)}
            required
          />
          <button type="submit" className="btn btn-primary" disabled={createInvite.isPending}>
            {createInvite.isPending ? 'Gerando…' : 'Gerar convite'}
          </button>
        </div>
      </form>

      {created && (
        <section className={`glass-card ${styles.section}`} aria-live="polite">
          <h2 className={styles.sectionTitle}>Convite para {created.email}</h2>
          <p className={styles.hint}>
            Copie agora: por segurança o link não fica salvo e não aparece de novo. Se perder,
            cancele este convite e gere outro.
          </p>

          <div className={styles.row}>
            <input className="input" readOnly value={link} aria-label="Link do convite" onFocus={e => e.target.select()} />
            <button type="button" className="btn btn-primary" onClick={() => copy(link, 'Link')}>
              Copiar link
            </button>
          </div>

          <textarea
            className={`input ${styles.message}`}
            readOnly
            value={inviteMessage(created, link)}
            aria-label="Mensagem sugerida"
            onFocus={e => e.target.select()}
          />
          <div>
            <button
              type="button"
              className="btn btn-ghost"
              onClick={() => copy(inviteMessage(created, link), 'Mensagem')}
            >
              Copiar mensagem
            </button>
          </div>
        </section>
      )}

      <section className={`glass-card ${styles.section}`}>
        <h2 className={styles.sectionTitle}>Convites enviados</h2>
        {invites.isLoading && <p className={styles.empty}>Carregando…</p>}
        {invites.isError && <p className={styles.empty}>Não foi possível carregar os convites.</p>}
        {invites.data && invites.data.length === 0 && (
          <p className={styles.empty}>Nenhum convite ainda.</p>
        )}
        {invites.data && invites.data.length > 0 && (
          <ul className={styles.list}>
            {invites.data.map(inv => (
              <li key={inv.id} className={styles.item}>
                <div className={styles.itemMain}>
                  <span className={styles.itemEmail}>{inv.email}</span>
                  <span className={styles.itemMeta}>
                    Criado em {formatDate(inv.created_at)}
                    {inv.status === 'pending' && ` · vale até ${formatDate(inv.expires_at)}`}
                    {inv.used_at && ` · usado em ${formatDate(inv.used_at)}`}
                  </span>
                </div>
                <div className={styles.itemActions}>
                  <span className={`${styles.status} ${styles[`status_${inv.status}`]}`}>
                    {STATUS_LABEL[inv.status]}
                  </span>
                  {inv.status === 'pending' && (
                    <button type="button" className="btn btn-ghost" onClick={() => setToRevoke(inv)}>
                      Cancelar
                    </button>
                  )}
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>

      <ConfirmDialog
        open={toRevoke !== null}
        title="Cancelar convite?"
        message={toRevoke ? `O link enviado para ${toRevoke.email} deixa de funcionar.` : ''}
        confirmLabel="Cancelar convite"
        cancelLabel="Voltar"
        onCancel={() => setToRevoke(null)}
        onConfirm={() => {
          if (!toRevoke) return
          revokeInvite.mutate(toRevoke.id, {
            onSuccess: () => toast.success('Convite cancelado'),
          })
          setToRevoke(null)
        }}
      />
    </div>
  )
}
