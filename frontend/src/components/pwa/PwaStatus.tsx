import { useRegisterSW } from 'virtual:pwa-register/react'
import { useOnlineStatus } from '@/hooks/useOnlineStatus'
import styles from './PwaStatus.module.css'

// Registra o service worker e mostra os dois avisos do app instalado:
// - sem internet: o app abre, mas nada é salvo até a conexão voltar;
// - nova versão: só troca quando a pessoa aceita, nunca no meio do uso.
export function PwaStatus() {
  const online = useOnlineStatus()
  const {
    needRefresh: [needRefresh, setNeedRefresh],
    updateServiceWorker,
  } = useRegisterSW()

  if (online && !needRefresh) return null

  return (
    <div className={styles.stack}>
      {!online && (
        <div className={styles.banner} role="status">
          <span className={styles.icon} aria-hidden="true">⚡︎</span>
          <span className={styles.text}>
            Você está sem internet. Dá para navegar, mas nada será salvo até a conexão voltar.
          </span>
        </div>
      )}
      {needRefresh && (
        <div className={styles.banner} role="status">
          <span className={styles.icon} aria-hidden="true">↻</span>
          <span className={styles.text}>Nova versão do MyRoutine disponível.</span>
          <button type="button" className="btn btn-primary" onClick={() => updateServiceWorker(true)}>
            Atualizar
          </button>
          <button type="button" className="btn btn-ghost" onClick={() => setNeedRefresh(false)}>
            Depois
          </button>
        </div>
      )}
    </div>
  )
}
