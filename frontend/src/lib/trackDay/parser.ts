// Leitor do diário (Track Day), trazido do app mobile (mobile/src/parser).
// Lê uma linha em texto livre e reconhece: o tipo de bullet (Bullet Journal),
// [[links]], gastos ("$ 50 Almoço #alimentacao") e treinos ("corrida 5km 30min").
//
// O que ele reconhece só vira gasto/treino quando a pessoa toca "Registrar"
// (a linha muda enquanto é digitada: "$ 5" antes de "$ 50").
import { FINANCE_CATEGORIES, parseAmountToCents } from '@/types/finance'

export type BulletType = 'task_pending' | 'task_done' | 'task_migrated' | 'note' | 'event' | 'none'
export type WorkoutType = 'run' | 'bike' | 'swim'

export interface ParsedLine {
  originalText: string
  bulletType: BulletType
  transaction?: {
    value: number
    amountCents: number
    description: string
    // Como escrito na hashtag (minúsculas); `financeCategory` é a categoria de
    // Finanças correspondente ("other" quando não há equivalente).
    category: string
    financeCategory: string
  }
  health?: {
    type: WorkoutType
    distance?: number // km
    time?: number // minutos
    rpe?: number
  }
  links: string[]
}

// Mesma normalização do backend (JournalLineKey): tira espaços das pontas e
// junta espaços repetidos. É a identidade da linha para "já registrado".
export function lineKey(line: string): string {
  return line.trim().split(/\s+/).filter(Boolean).join(' ')
}

function stripAccents(s: string): string {
  return s.normalize('NFD').replace(/[̀-ͯ]/g, '')
}

// "#Alimentação" → "alimentacao"; hashtag sem categoria equivalente → "other".
export function toFinanceCategory(tag: string | undefined): string {
  if (!tag) return 'other'
  const key = stripAccents(tag).toLowerCase()
  return key in FINANCE_CATEGORIES ? key : 'other'
}

// Valor: "50", "25,90", "25.90" ou "1.234,56".
const FINANCE_RE = /^\$\s*(\d{1,3}(?:\.\d{3})+(?:,\d{1,2})?|\d+(?:[.,]\d{1,2})?)\s+(.*?)(?:\s+#(\S+))?\s*$/i

// \b depois da unidade: o "30m" de "30min" NÃO é distância.
const DISTANCE_RE = /(\d+(?:[.,]\d+)?)\s*(km|m)\b/i
// "1h", "1,5 horas" e "1h30min" (número logo depois do h também encerra a unidade).
const HOURS_RE = /(\d+(?:[.,]\d+)?)\s*h(?:oras?)?(?=\d|\b)/i
const MINUTES_RE = /(\d+)\s*min/i
const RPE_RE = /RPE\s*(\d+)/i

function toNumber(s: string): number {
  return parseFloat(s.replace(',', '.'))
}

export function parseLine(line: string): ParsedLine {
  const trimmed = line.trim()

  let bulletType: BulletType = 'none'
  if (trimmed.startsWith('•')) bulletType = 'task_pending'
  else if (/^[xX]\s/.test(trimmed)) bulletType = 'task_done'
  else if (trimmed.startsWith('>')) bulletType = 'task_migrated'
  else if (trimmed.startsWith('-')) bulletType = 'note'
  else if (trimmed.startsWith('○')) bulletType = 'event'

  const links = (trimmed.match(/\[\[(.*?)\]\]/g) ?? []).map(l => l.slice(2, -2))

  let transaction: ParsedLine['transaction']
  const finance = trimmed.match(FINANCE_RE)
  if (finance) {
    const amountCents = parseAmountToCents(finance[1])
    if (amountCents !== null) {
      const tag = finance[3]?.toLowerCase()
      transaction = {
        value: amountCents / 100,
        amountCents,
        description: finance[2].trim(),
        category: tag ?? 'other',
        financeCategory: toFinanceCategory(tag),
      }
    }
  }

  let health: ParsedLine['health']
  const isRun = trimmed.includes('🏃') || /corrida/i.test(trimmed)
  const isBike = trimmed.includes('🚴') || /ciclismo/i.test(trimmed)
  const isSwim = trimmed.includes('🏊') || /nata[çc][ãa]o/i.test(trimmed)
  if (isRun || isBike || isSwim) {
    const dist = trimmed.match(DISTANCE_RE)
    const hours = trimmed.match(HOURS_RE)
    const minutes = trimmed.match(MINUTES_RE)
    const rpe = trimmed.match(RPE_RE)

    let time: number | undefined
    if (hours || minutes) {
      time = Math.round((hours ? toNumber(hours[1]) * 60 : 0) + (minutes ? toNumber(minutes[1]) : 0))
    }

    health = {
      type: isRun ? 'run' : isBike ? 'bike' : 'swim',
      // Metros viram km: "natação 1500m" é 1,5 km.
      distance: dist ? (dist[2].toLowerCase() === 'm' ? toNumber(dist[1]) / 1000 : toNumber(dist[1])) : undefined,
      time,
      rpe: rpe ? parseInt(rpe[1], 10) : undefined,
    }
  }

  return { originalText: line, bulletType, transaction, health, links }
}

export function parseNoteContent(content: string): ParsedLine[] {
  return content.split('\n').map(parseLine)
}

// ─── Treino → hábito ─────────────────────────────────────────────────────────
// Mapa explícito tipo-de-treino → nome do hábito (do app mobile). Deliberadamente
// explícito: um "match aproximado" erraria em silêncio e marcaria o hábito errado.
export const WORKOUT_HABIT_NAMES: Record<WorkoutType, string[]> = {
  run: ['correr', 'corrida'],
  bike: ['pedalar', 'ciclismo', 'bike'],
  swim: ['nadar', 'natação', 'natacao'],
}

export interface HabitRef {
  id: string
  name: string
  category?: string | null
}

// Hábito de Saúde que corresponde ao treino, ou null quando não há um claro
// (nunca chuta um destino).
export function resolveHabitForWorkout<T extends HabitRef>(type: WorkoutType, habits: T[]): T | null {
  const aliases = WORKOUT_HABIT_NAMES[type]
  return habits
    .filter(h => h.category === 'health')
    .find(h => aliases.some(alias => h.name.toLowerCase().includes(alias))) ?? null
}
