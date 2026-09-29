/// <reference lib="webworker" />
// Service worker do MyRoutine (estratégia injectManifest do vite-plugin-pwa).
//
// Guarda no aparelho SÓ os arquivos do app (JS, CSS, HTML, ícones), para ele
// abrir rápido e mesmo sem internet. Nenhuma resposta da API é guardada: os
// dados de uma pessoa nunca ficam num cache que outra pessoa do mesmo aparelho
// veria, e o app nunca mostra saldo ou hábito velho como se fosse atual.
//
// Na Etapa 4 (notificações) os handlers de "push" entram aqui.
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
