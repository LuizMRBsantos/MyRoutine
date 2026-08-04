import { AnimatePresence, motion } from 'framer-motion'
import styles from './ConfirmDialog.module.css'

interface ConfirmDialogProps {
  open: boolean
  title: string
  message: string
  confirmLabel?: string
  cancelLabel?: string
  onConfirm: () => void
  onCancel: () => void
}

export function ConfirmDialog({
  open,
  title,
  message,
  confirmLabel = 'Confirmar',
  cancelLabel = 'Cancelar',
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  return (
    <AnimatePresence>
      {open && (
        <>
          <motion.div
            className={styles.backdrop}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            onClick={onCancel}
          />
          <div className={styles.wrapper}>
            <motion.div
              className={styles.dialog}
              role="alertdialog"
              aria-modal="true"
              initial={{ opacity: 0, y: 16, scale: 0.96 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0, y: 8, scale: 0.97 }}
              transition={{ type: 'spring', damping: 26, stiffness: 320 }}
            >
              <h3 className={styles.title}>{title}</h3>
              <p className={styles.message}>{message}</p>
              <div className={styles.actions}>
                <button className="btn btn-ghost" onClick={onCancel} autoFocus>
                  {cancelLabel}
                </button>
                <button className="btn btn-primary" onClick={onConfirm}>
                  {confirmLabel}
                </button>
              </div>
            </motion.div>
          </div>
        </>
      )}
    </AnimatePresence>
  )
}
