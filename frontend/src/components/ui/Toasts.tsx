import { AnimatePresence, motion } from 'framer-motion'
import { useToastStore } from '@/lib/toast'
import styles from './Toasts.module.css'

const KIND_ICON = { success: '✓', error: '!', info: 'ℹ' } as const

export function Toasts() {
  const toasts = useToastStore((s) => s.toasts)
  const dismiss = useToastStore((s) => s.dismiss)

  return (
    <div className={styles.container} role="status" aria-live="polite">
      <AnimatePresence>
        {toasts.map((t) => (
          <motion.div
            key={t.id}
            className={`${styles.toast} ${styles[t.kind]}`}
            initial={{ opacity: 0, y: 12, scale: 0.96 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 8, scale: 0.97 }}
            transition={{ duration: 0.2 }}
            onClick={() => dismiss(t.id)}
          >
            <span className={styles.icon}>{KIND_ICON[t.kind]}</span>
            <span className={styles.message}>{t.message}</span>
          </motion.div>
        ))}
      </AnimatePresence>
    </div>
  )
}
