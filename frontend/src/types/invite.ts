export type InviteStatus = 'pending' | 'used' | 'revoked' | 'expired'

export interface Invite {
  id: string
  email: string
  status: InviteStatus
  expires_at: string
  created_at: string
  used_at: string | null
}

// Devolvido só na criação: é a única vez que o código do convite existe
// fora do link. O backend guarda apenas o hash.
export interface CreatedInvite {
  id: string
  email: string
  token: string
  expires_at: string
}

// O link que o admin copia e envia. Aponta para o cadastro deste mesmo site.
export function inviteLink(token: string, origin = window.location.origin): string {
  return `${origin}/register?convite=${encodeURIComponent(token)}`
}

// Link de redefinição de senha gerado pelo admin (vale 1 hora, uma vez).
export interface CreatedReset {
  email: string
  token: string
  expires_at: string
}

export function resetLink(token: string, origin = window.location.origin): string {
  return `${origin}/redefinir-senha?codigo=${encodeURIComponent(token)}`
}
