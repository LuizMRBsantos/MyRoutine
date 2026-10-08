/// <reference lib="webworker" />
// Service worker do MyRoutine (estratégia injectManifest do vite-plugin-pwa).
//
// Guarda no aparelho SÓ os arquivos do app (JS, CSS, HTML, ícones), para ele
// abrir rápido e mesmo sem internet. Nenhuma resposta da API é guardada: os
// dados de uma pessoa nunca ficam num cache que outra pessoa do mesmo aparelho
// veria, e o app nunca mostra saldo ou hábito velho como se fosse atual.
//
// Também mostra as notificações (Web Push) que o servidor manda.
import { cleanupOutdatedCaches, createHandlerBoundToURL, precacheAndRoute } from 'workbox-precaching'
import { NavigationRoute, registerRoute } from 'workbox-routing'

declare const self: ServiceWorkerGlobalScope

// Lista de arquivos do build, injetada pelo plugin na hora do build.
precacheAndRoute(self.__WB_MANIFEST)
cleanupOutdatedCaches()

// Rotas do app (/habits, /conta...) abrem a casca guardada (index.html).
// API, health e métricas nunca: essas vão sempre para a rede.
registerRoute(
  new NavigationRoute(createHandlerBoundToURL('/index.html'), {
    denylist: [/^\/api\//, /^\/health/, /^\/metrics/],
  })
)

// A versão nova só assume quando a pessoa aceita ("Atualizar"), para não
// trocar o app no meio do uso.
self.addEventListener('message', (event) => {
  if (event.data?.type === 'SKIP_WAITING') void self.skipWaiting()
})

// ─── Notificações ────────────────────────────────────────────
// O servidor manda {title, body, url, tag}; a mensagem chega cifrada e só
// este aparelho consegue abrir.
interface PushPayload {
  title?: string
  body?: string
  url?: string
  tag?: string
}

self.addEventListener('push', (event) => {
  let data: PushPayload = {}
  try {
    data = event.data?.json() ?? {}
  } catch {
    data = { body: event.data?.text() }
  }
  event.waitUntil(
    self.registration.showNotification(data.title || 'MyRoutine', {
      body: data.body,
      tag: data.tag,
      icon: '/icons/pwa-192.png',
      badge: '/icons/pwa-192.png',
      data: { url: data.url },
    })
  )
})

// Só caminhos do próprio app ("/planner"), nunca outro site.
function safePath(url: unknown): string {
  return typeof url === 'string' && url.startsWith('/') && !url.startsWith('//') ? url : '/'
}

// Toque na notificação: abre (ou traz para frente) o app na tela certa.
self.addEventListener('notificationclick', (event) => {
  event.notification.close()
  const path = safePath(event.notification.data?.url)
  event.waitUntil((async () => {
    const windows = await self.clients.matchAll({ type: 'window', includeUncontrolled: true })
    const open = windows[0]
    if (open) {
      await open.focus()
      await open.navigate(path).catch(() => undefined)
      return
    }
    await self.clients.openWindow(path)
  })())
})

