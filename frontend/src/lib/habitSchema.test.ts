import { describe, it, expect } from 'vitest'
import { normalizeHabitPayload, validateHabit } from './habitSchema'
import type { CreateHabitInput } from '@/types/habit'

function baseHabit(overrides: Partial<CreateHabitInput> = {}): CreateHabitInput {
  return {
    name: 'Correr',
    icon: '🏃',
    color: '#0071E3',
    frequency: 'daily',
    target_days: [1, 2, 3, 4, 5, 6, 7],
    time_of_day: 'morning',
    category: 'health',
    check_type: 'simple',
    ...overrides,
  }
}

describe('validateHabit', () => {
  it('aceita um hábito simples válido', () => {
    expect(validateHabit(baseHabit())).toBeNull()
  })

  it('rejeita nome vazio ou só espaços', () => {
    expect(validateHabit(baseHabit({ name: '' }))).toBe('Dê um nome ao hábito')
    expect(validateHabit(baseHabit({ name: '   ' }))).toBe('Dê um nome ao hábito')
  })

  it('exige ao menos um dia da semana', () => {
    expect(validateHabit(baseHabit({ target_days: [] })))
      .toBe('Escolha ao menos um dia da semana')
  })

  it('exige duração para hábito com timer', () => {
    expect(validateHabit(baseHabit({ check_type: 'timed' })))
      .toBe('Informe a duração do timer')
    expect(validateHabit(baseHabit({ check_type: 'timed', timer_minutes: 30 })))
      .toBeNull()
  })

  it('exige horário para hábito com prazo', () => {
    expect(validateHabit(baseHabit({ check_type: 'deadline' })))
      .toBe('Informe o horário limite')
    expect(validateHabit(baseHabit({ check_type: 'deadline', deadline_time: '05:30' })))
      .toBeNull()
  })

  it('exige ao menos uma métrica para hábito de métrica', () => {
    expect(validateHabit(baseHabit({ check_type: 'metric' })))
      .toBe('Adicione ao menos uma métrica')
    expect(validateHabit(baseHabit({
      check_type: 'metric',
      metric_config: [{ key: 'km', label: 'Distância', unit: 'km' }],
    }))).toBeNull()
  })

  it('rejeita timer com valor não positivo', () => {
    expect(validateHabit(baseHabit({ check_type: 'timed', timer_minutes: 0 })))
      .not.toBeNull()
    expect(validateHabit(baseHabit({ check_type: 'timed', timer_minutes: -5 })))
      .not.toBeNull()
  })
})

describe('normalizeHabitPayload', () => {
  it('remove configuração que não pertence ao check_type', () => {
    const payload = normalizeHabitPayload(baseHabit({
      check_type: 'simple',
      timer_minutes: 30,
      deadline_time: '05:30',
      metric_config: [{ key: 'km', label: 'Distância', unit: 'km' }],
    }))

    expect(payload.timer_minutes).toBeUndefined()
    expect(payload.deadline_time).toBeUndefined()
    expect(payload.metric_config).toBeUndefined()
  })

  it('preserva a configuração do check_type escolhido', () => {
    const payload = normalizeHabitPayload(baseHabit({
      check_type: 'timed',
      timer_minutes: 30,
      deadline_time: '05:30',
    }))

    expect(payload.timer_minutes).toBe(30)
    expect(payload.deadline_time).toBeUndefined()
  })

  it('preserva is_target e target_value das métricas', () => {
    const payload = normalizeHabitPayload(baseHabit({
      check_type: 'metric',
      metric_config: [
        { key: 'km', label: 'Distância', unit: 'km', is_target: true, target_value: 5 },
      ],
    }))

    expect(payload.metric_config).toHaveLength(1)
    expect(payload.metric_config![0].is_target).toBe(true)
    expect(payload.metric_config![0].target_value).toBe(5)
  })

  it('descarta campos de métrica incompletos', () => {
    const payload = normalizeHabitPayload(baseHabit({
      check_type: 'metric',
      metric_config: [
        { key: 'km', label: 'Distância', unit: 'km' },
        { key: '', label: '', unit: '' },
      ],
    }))

    expect(payload.metric_config).toHaveLength(1)
    expect(payload.metric_config![0].key).toBe('km')
  })

  it('remove espaços em volta do nome', () => {
    expect(normalizeHabitPayload(baseHabit({ name: '  Correr  ' })).name).toBe('Correr')
  })

  it('trata deadline vazio como ausente', () => {
    const payload = normalizeHabitPayload(baseHabit({
      check_type: 'deadline',
      deadline_time: '',
    }))
    expect(payload.deadline_time).toBeUndefined()
    // e o schema então acusa a falta
    expect(validateHabit(payload)).toBe('Informe o horário limite')
  })
})
