import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { AddTaskModal } from './AddTaskModal'
import type { CreateTaskInput } from '@/types/task'

function renderModal(onSave = vi.fn<(input: CreateTaskInput) => void>()) {
  render(<AddTaskModal date="2030-03-12" onSave={onSave} onClose={() => {}} isPending={false} />)
  return onSave
}

describe('AddTaskModal — Me avisar antes', () => {
  it('só aparece com horário e segue a categoria até a pessoa escolher', async () => {
    const user = userEvent.setup()
    const onSave = renderModal()

    expect(screen.queryByRole('checkbox', { name: 'Me avisar antes' })).not.toBeInTheDocument()

    await user.type(screen.getByPlaceholderText('O que você vai fazer?'), 'Aula de Cálculo')
    await user.type(screen.getByPlaceholderText('Ex: 05:30 ou 16:00'), '1400')
    const notify = screen.getByRole('checkbox', { name: 'Me avisar antes' })
    expect(notify).not.toBeChecked() // "Outro"

    await user.click(screen.getByRole('button', { name: /Compromisso/ }))
    expect(notify).toBeChecked()
    await user.click(screen.getByRole('button', { name: /Estudo/ }))
    expect(notify).not.toBeChecked()

    await user.click(notify) // escolha da pessoa vale mais que a categoria
    await user.click(screen.getByRole('button', { name: /Prova/ }))
    expect(notify).toBeChecked()

    await user.click(screen.getByRole('button', { name: /Criar|Adicionar|Salvar/ }))
    expect(onSave).toHaveBeenCalledWith(expect.objectContaining({ category: 'exam', notify: true, start_time: '14:00' }))
  })
})
