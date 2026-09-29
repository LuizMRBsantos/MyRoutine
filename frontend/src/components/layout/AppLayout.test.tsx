import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { useAuthStore } from '@/store/authStore'
import { AppLayout } from './AppLayout'

// O jsdom não avalia media queries: a barra do celular fica "display: none"
// aqui. Estes testes cobrem a lógica; o visual é conferido no navegador.
const H = { hidden: true } as const

// Para elementos escondidos, a biblioteca calcula o nome acessível como vazio,
// então buscamos pelo aria-label diretamente.
function byLabel(role: string, label: string): HTMLElement {
  const el = screen.getAllByRole(role, H).find(e => e.getAttribute('aria-label') === label)
  if (!el) throw new Error(`${role} "${label}" não encontrado`)
  return el
}

function renderAt(path: string, isAdmin = false) {
  useAuthStore.setState({
    accessToken: 'a', refreshToken: 'r', isAuthenticated: true,
    user: { id: '1', name: 'Luiz', email: 'l@x.com', createdAt: '', is_admin: isAdmin },
  })
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/" element={<div>página inicial</div>} />
          <Route path="/health" element={<div>página saúde</div>} />
        </Route>
      </Routes>
    </MemoryRouter>
  )
}

beforeEach(() => {
  useAuthStore.setState({ accessToken: null, refreshToken: null, user: null, isAuthenticated: false })
})

describe('AppLayout — navegação no celular', () => {
  it('barra inferior tem os 4 módulos principais e o "Mais"', () => {
    renderAt('/')
    const bar = byLabel('navigation', 'Navegação principal')
    const labels = within(bar).getAllByRole('link', H).map(l => l.textContent)
    expect(labels).toEqual(['⊞Início', '✎Diário', '✦Hábitos', '📅Planner'])
    expect(within(bar).getByRole('button', { name: /Mais/, ...H })).toHaveAttribute('aria-expanded', 'false')
  })

  it('"Mais" abre a folha com o resto e o Sair; admin vê Acessos', async () => {
    const user = userEvent.setup()
    renderAt('/', true)

    await user.click(screen.getByRole('button', { name: /Mais/, ...H }))
    const sheet = byLabel('dialog', 'Mais opções')
    const items = within(sheet).getAllByRole('link', H).map(l => l.textContent)
    expect(items).toEqual(['◈Finanças', '◉Saúde', '◆Estudos', '↺Revisão', '⚙Minha conta', '✉Acessos'])
    expect(within(sheet).getByRole('button', { name: /Sair/, ...H })).toBeInTheDocument()
  })

  it('fecha a folha ao navegar e com Esc', async () => {
    const user = userEvent.setup()
    renderAt('/')

    await user.click(screen.getByRole('button', { name: /Mais/, ...H }))
    await user.click(within(screen.getByRole('dialog', H)).getByRole('link', { name: /Saúde/, ...H }))
    expect(screen.getByText('página saúde')).toBeInTheDocument()
    expect(screen.queryByRole('dialog', H)).not.toBeInTheDocument()
    // Saúde está no "Mais": o botão fica marcado como ativo.
    expect(screen.getByRole('button', { name: /Mais/, ...H }).className).toMatch(/tabActive/)

    await user.click(screen.getByRole('button', { name: /Mais/, ...H }))
    await user.keyboard('{Escape}')
    expect(screen.queryByRole('dialog', H)).not.toBeInTheDocument()
  })
})
