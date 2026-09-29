import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { TodayHabits } from './TodayHabits'

function renderWith(props: { isError?: boolean }) {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter>
        <TodayHabits habits={[]} isLoading={false} {...props} />
      </MemoryRouter>
    </QueryClientProvider>
  )
}

describe('TodayHabits', () => {
  it('lista vazia de verdade convida a criar um hábito (sem recarregar a página)', () => {
    renderWith({})
    expect(screen.getByText(/Nenhum hábito para hoje/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Criar hábito/ })).toHaveAttribute('href', '/habits')
  })

  it('falha ao carregar não finge que os hábitos sumiram', () => {
    renderWith({ isError: true })
    expect(screen.getByRole('status')).toHaveTextContent('Não deu para carregar seus hábitos agora')
    expect(screen.queryByText(/Nenhum hábito para hoje/)).not.toBeInTheDocument()
  })
})
