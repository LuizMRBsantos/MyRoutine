import { useState } from 'react'
import { useAuthStore } from '@/store/authStore'
import { useInvites, useCreateInvite, useRevokeInvite, useCreatePasswordReset } from '@/hooks/useInvites'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { toast, apiErrorMessage } from '@/lib/toast'
import {
  inviteLink, resetLink,
  type CreatedInvite, type CreatedReset, type Invite, type InviteStatus,
} from '@/types/invite'
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

function formatTime(iso: string): string {
  return new Date(iso).toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' })
}

function resetMessage(reset: CreatedReset, link: string): string {
  return (
    'Oi! Aqui está o link para você criar uma nova senha no MyRoutine. ' +
    `Ele vale até ${formatTime(reset.expires_at)} (1 hora) e só funciona uma vez:\n` +
    link
  )
}

// Link + mensagem pronta, cada um com botão de copiar. O código só existe
// neste momento: o servidor guarda apenas o hash.
function ShareLink({ title, link, message, linkLabel }: {
  title: string
  link: string
  message: string
  linkLabel: string
}) {
  return (
    <section className={`glass-card ${styles.section}`} aria-live="polite">
      <h2 className={styles.sectionTitle}>{title}</h2>
      <p className={styles.hint}>
        Copie agora: por segurança o link não fica salvo e não aparece de novo. Se perder,
        gere outro.
      </p>

      <div className={styles.row}>
        <input className="input" readOnly value={link} aria-label={linkLabel} onFocus={e => e.target.select()} />
        <button type="button" className="btn btn-primary" onClick={() => copy(link, 'Link')}>
          Copiar link
        </button>
      </div>

      <textarea
        className={`input ${styles.message}`}
        readOnly
        value={message}
        aria-label="Mensagem sugerida"
        onFocus={e => e.target.select()}
      />
      <div>
        <button type="button" className="btn btn-ghost" onClick={() => copy(message, 'Mensagem')}>
          Copiar mensagem
        </button>
      </div>
    </section>
  )
}

export function InvitesPage() {
  const isAdmin = useAuthStore(s => s.user?.is_admin === true)
  const invites = useInvites()
  const createInvite = useCreateInvite()
  const revokeInvite = useRevokeInvite()
  const createReset = useCreatePasswordReset()

  const [email, setEmail] = useState('')
  const [created, setCreated] = useState<CreatedInvite | null>(null)
  const [toRevoke, setToRevoke] = useState<Invite | null>(null)
  const [resetEmail, setResetEmail] = useState('')
  const [createdReset, setCreatedReset] = useState<CreatedReset | null>(null)

  if (!isAdmin) {
    return (
      <div className={styles.page}>
        <h1 className={styles.pageTitle}>Acessos</h1>
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

  const handleCreateReset = (e: React.FormEvent) => {
    e.preventDefault()
    createReset.mutate(resetEmail, {
      onSuccess: (reset) => {
        setCreatedReset(reset)
        setResetEmail('')
      },
      onError: (err) => {
        const msg = apiErrorMessage(err, '')
        toast.error(msg === 'no account with this email'
          ? 'Não existe conta com esse e-mail.'
          : 'Não foi possível gerar o link.')
      },
    })
  }

  const link = created ? inviteLink(created.token) : ''
  const rLink = createdReset ? resetLink(createdReset.token) : ''

  return (
    <div className={styles.page}>
      <div>
        <h1 className={styles.pageTitle}>Acessos</h1>
        <p className={styles.pageSubtitle}>
          O MyRoutine está em beta fechado e não envia e-mails: você gera os links aqui e envia
          pelo seu e-mail ou WhatsApp. Convites valem 7 dias; links de nova senha, 1 hora.
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
        <ShareLink
          title={`Convite para ${created.email}`}
          link={link}
          message={inviteMessage(created, link)}
          linkLabel="Link do convite"
        />
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

      <form className={`glass-card ${styles.section}`} onSubmit={handleCreateReset}>
        <h2 className={styles.sectionTitle}>Redefinir senha de alguém</h2>
        <p className={styles.hint}>
          Quando um convidado esquecer a senha, gere aqui um link para ele criar uma nova. Gerar um
          link novo cancela o anterior, e trocar a senha desconecta a pessoa de todos os aparelhos.
        </p>
        <div className={styles.row}>
          <input
            type="email"
            className="input"
            placeholder="email@da-pessoa.com"
            aria-label="E-mail da conta para redefinir"
            value={resetEmail}
            onChange={e => setResetEmail(e.target.value)}
            required
          />
          <button type="submit" className="btn btn-primary" disabled={createReset.isPending}>
            {createReset.isPending ? 'Gerando…' : 'Gerar link de nova senha'}
          </button>
        </div>
      </form>

      {createdReset && (
        <ShareLink
          title={`Nova senha para ${createdReset.email}`}
          link={rLink}
          message={resetMessage(createdReset, rLink)}
          linkLabel="Link de nova senha"
        />
      )}

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
