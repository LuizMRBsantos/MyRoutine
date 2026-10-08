import api from '@/services/api'

// Web Push no navegador: pedir permissão, inscrever ESTE aparelho e avisar o
// servidor. O servidor só guarda o endereço do aparelho e as chaves para
// cifrar a mensagem; quem mostra a notificação é o service worker (sw/sw.ts).

export type PushSupport = 'ok' | 'needs-install' | 'unsupported'

// No iPhone/iPad, notificação só existe com o app instalado na Tela de Início
// (iOS 16.4+). No Safari comum, o PushManager nem aparece.
export function pushSupport(): PushSupport {
  const ios = /iPhone|iPad|iPod/.test(navigator.userAgent) ||
    (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1)
  const standalone = window.matchMedia?.('(display-mode: standalone)').matches ||
    (navigator as Navigator & { standalone?: boolean }).standalone === true
  const apis = 'serviceWorker' in navigator && 'PushManager' in window && 'Notification' in window

  if (ios && !standalone) return 'needs-install'
  return apis ? 'ok' : 'unsupported'
}

// A chave pública VAPID vem em base64url; o navegador quer os bytes.
export function urlBase64ToUint8Array(base64: string): Uint8Array<ArrayBuffer> {
  const padded = (base64 + '='.repeat((4 - (base64.length % 4)) % 4)).replace(/-/g, '+').replace(/_/g, '/')
  const raw = atob(padded)
  const bytes = new Uint8Array(new ArrayBuffer(raw.length))
  for (let i = 0; i < raw.length; i++) bytes[i] = raw.charCodeAt(i)
  return bytes
}

export async function currentSubscription(): Promise<PushSubscription | null> {
  if (pushSupport() !== 'ok') return null
  const reg = await navigator.serviceWorker.getRegistration()
  return (await reg?.pushManager.getSubscription()) ?? null
}

export class PushPermissionDenied extends Error {}

// Pede permissão (precisa vir de um toque da pessoa), inscreve o aparelho e
// registra no servidor.
export async function subscribeThisDevice(vapidPublicKey: string): Promise<void> {
  const permission = await Notification.requestPermission()
  if (permission !== 'granted') throw new PushPermissionDenied()

  const reg = await navigator.serviceWorker.ready
  const sub = (await reg.pushManager.getSubscription()) ?? await reg.pushManager.subscribe({
    userVisibleOnly: true,
    applicationServerKey: urlBase64ToUint8Array(vapidPublicKey),
  })
  await api.post('/notifications/subscriptions', sub.toJSON())
}

export async function unsubscribeThisDevice(): Promise<void> {
  const sub = await currentSubscription()
  if (!sub) return
  await api.delete('/notifications/subscriptions', { data: { endpoint: sub.endpoint } })
  await sub.unsubscribe()
}

// Ao sair da conta: o servidor esquece este aparelho (para não mandar os
// compromissos de alguém para um aparelho onde ninguém está logado). A
// inscrição do navegador fica, e syncThisDevice a devolve no próximo login.
export async function forgetThisDeviceOnServer(): Promise<void> {
  const sub = await currentSubscription()
  if (sub) await api.delete('/notifications/subscriptions', { data: { endpoint: sub.endpoint } })
}

// Se este aparelho já tem permissão e inscrição, garante que ela está com a
// pessoa logada agora (o servidor move o aparelho para quem entrou).
export async function syncThisDevice(): Promise<void> {
  if (pushSupport() !== 'ok' || Notification.permission !== 'granted') return
  const sub = await currentSubscription()
  if (sub) await api.post('/notifications/subscriptions', sub.toJSON())
}
