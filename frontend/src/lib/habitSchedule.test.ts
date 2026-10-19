import { describe, it, expect } from 'vitest'
import { scheduleLabel } from './habitSchedule'

describe('scheduleLabel', () => {
  it('descreve os dias do hábito em português', () => {
    expect(scheduleLabel([1, 2, 3, 4, 5, 6, 7])).toBe('todo dia')
    expect(scheduleLabel([5, 4, 3, 2, 1])).toBe('seg a sex')
    expect(scheduleLabel([6, 7])).toBe('fim de semana')
    expect(scheduleLabel([2, 4, 6])).toBe('ter, qui e sáb')
    expect(scheduleLabel([3])).toBe('qua')
  })
})
