import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ConfirmDialog } from './ConfirmDialog'

// Exclusões são irreversíveis; este diálogo é a única barreira entre um
// clique acidental e a perda de dados.
describe('ConfirmDialog', () => {
  const baseProps = {
    open: true,
    title: 'Excluir hábito?',
    message: '"Correr" será arquivado.',
    onConfirm: vi.fn(),
    onCancel: vi.fn(),
  }

  it('não renderiza nada quando fechado', () => {
    render(<ConfirmDialog {...baseProps} open={false} />)
    expect(screen.queryByText('Excluir hábito?')).not.toBeInTheDocument()
  })

  it('mostra título e mensagem quando aberto', () => {
    render(<ConfirmDialog {...baseProps} />)
    expect(screen.getByText('Excluir hábito?')).toBeInTheDocument()
    expect(screen.getByText('"Correr" será arquivado.')).toBeInTheDocument()
  })

  it('chama onConfirm apenas ao confirmar', async () => {
    const onConfirm = vi.fn()
    const onCancel = vi.fn()
    render(<ConfirmDialog {...baseProps} onConfirm={onConfirm} onCancel={onCancel} />)

    await userEvent.click(screen.getByRole('button', { name: 'Confirmar' }))

    expect(onConfirm).toHaveBeenCalledOnce()
    expect(onCancel).not.toHaveBeenCalled()
  })

  it('chama onCancel ao cancelar', async () => {
    const onConfirm = vi.fn()
    const onCancel = vi.fn()
    render(<ConfirmDialog {...baseProps} onConfirm={onConfirm} onCancel={onCancel} />)

    await userEvent.click(screen.getByRole('button', { name: 'Cancelar' }))

    expect(onCancel).toHaveBeenCalledOnce()
    expect(onConfirm).not.toHaveBeenCalled()
  })

  it('usa rótulos customizados', () => {
    render(<ConfirmDialog {...baseProps} confirmLabel="Excluir" cancelLabel="Voltar" />)
    expect(screen.getByRole('button', { name: 'Excluir' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Voltar' })).toBeInTheDocument()
  })

  // O foco começa em Cancelar: a ação destrutiva nunca é acionada por um
  // Enter reflexo.
  it('foca o botão seguro por padrão', () => {
    render(<ConfirmDialog {...baseProps} />)
    expect(screen.getByRole('button', { name: 'Cancelar' })).toHaveFocus()
  })
})
