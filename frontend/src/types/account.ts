export interface Profile {
  id: string
  name: string
  email: string
  avatar_url: string | null
  timezone: string
  created_at: string
}

// Fuso detectado no navegador (IANA, ex.: "America/Sao_Paulo").
export function browserTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || 'America/Sao_Paulo'
  } catch {
    return 'America/Sao_Paulo'
  }
}

const FALLBACK_TIMEZONES = [
  'America/Sao_Paulo', 'America/Manaus', 'America/Belem', 'America/Fortaleza',
  'America/Recife', 'America/Cuiaba', 'America/Rio_Branco', 'America/Noronha',
  'America/New_York', 'America/Los_Angeles', 'Europe/Lisbon', 'Europe/London',
  'Europe/Madrid', 'Europe/Paris', 'Asia/Tokyo', 'UTC',
]

// Lista de fusos para escolher; o navegador fornece a lista completa quando
// suporta `Intl.supportedValuesOf`.
export function timezoneOptions(current: string): string[] {
  let all: string[] = FALLBACK_TIMEZONES
  try {
    const intl = Intl as unknown as { supportedValuesOf?: (key: string) => string[] }
    if (intl.supportedValuesOf) all = intl.supportedValuesOf('timeZone')
  } catch {
    // mantém a lista curta
  }
  // A lista do navegador omite "UTC"; garantimos que ele esteja lá.
  return Array.from(new Set([current, browserTimezone(), ...all, 'UTC'])).filter(Boolean)
}
