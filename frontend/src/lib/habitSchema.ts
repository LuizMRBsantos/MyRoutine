import { z } from 'zod'
import type { CreateHabitInput } from '@/types/habit'

// Validação do formulário de hábito. Vive fora do componente para ser
// testável isoladamente — é o portão que impede payloads inválidos de
// chegarem ao backend.
export const habitSchema = z
  .object({
    name: z.string().trim().min(1, 'Dê um nome ao hábito'),
    check_type: z.enum(['simple', 'timed', 'deadline', 'metric']),
    timer_minutes: z.number().int().positive().optional(),
    deadline_time: z.string().optional(),
    metric_config: z
      .array(z.object({
        key: z.string().min(1),
        label: z.string().min(1),
        unit: z.string(),
        is_target: z.boolean().optional(),
        target_value: z.number().optional(),
      }))
      .optional(),
    target_days: z.array(z.number()).min(1, 'Escolha ao menos um dia da semana'),
  })
  .superRefine((data, ctx) => {
    if (data.check_type === 'timed' && !data.timer_minutes) {
      ctx.addIssue({ code: 'custom', message: 'Informe a duração do timer' })
    }
    if (data.check_type === 'deadline' && !data.deadline_time) {
      ctx.addIssue({ code: 'custom', message: 'Informe o horário limite' })
    }
    if (data.check_type === 'metric' && !data.metric_config?.length) {
      ctx.addIssue({ code: 'custom', message: 'Adicione ao menos uma métrica' })
    }
  })

/**
 * Remove os campos que não pertencem ao check_type escolhido antes de validar.
 * Sem isso, trocar o tipo de hábito deixaria configuração órfã no payload —
 * e o backend rejeitaria pelas CHECK constraints da migration 004.
 */
export function normalizeHabitPayload(form: CreateHabitInput): CreateHabitInput {
  const payload: CreateHabitInput = { ...form, name: form.name.trim() }

  if (payload.check_type !== 'timed') delete payload.timer_minutes
  if (payload.check_type !== 'deadline' || !payload.deadline_time) delete payload.deadline_time
  if (payload.check_type !== 'metric' || !payload.metric_config?.length) delete payload.metric_config
  if (payload.check_type === 'metric') {
    payload.metric_config = payload.metric_config?.filter((m) => m.key && m.label)
  }

  return payload
}

/** Retorna a primeira mensagem de erro, ou null se o payload é válido. */
export function validateHabit(payload: CreateHabitInput): string | null {
  const result = habitSchema.safeParse(payload)
  if (result.success) return null
  return result.error.issues[0]?.message ?? 'Verifique os campos do formulário'
}
