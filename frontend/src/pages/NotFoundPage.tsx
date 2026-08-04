import { Link } from 'react-router-dom'

export function NotFoundPage() {
  return (
    <div style={{
      minHeight: '60vh',
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      gap: 'var(--space-4)',
      textAlign: 'center',
      padding: 'var(--space-8)',
    }}>
      <span style={{ fontSize: '2.5rem' }}>✦</span>
      <h1 style={{ fontSize: 'var(--font-size-2xl)', color: 'var(--color-text-primary)' }}>
        Página não encontrada
      </h1>
      <p style={{ color: 'var(--color-text-secondary)' }}>
        O endereço que você acessou não existe.
      </p>
      <Link to="/" className="btn btn-primary">Voltar ao Dashboard</Link>
    </div>
  )
}
