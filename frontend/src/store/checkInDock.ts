import { create } from 'zustand'
import { persist } from 'zustand/middleware'

// Qual hábito está aberto na janelinha de check-in (timer ou medidas) do
// canto inferior direito. Guardado no aparelho para o timer reaparecer
// depois de recarregar a página; o tempo em si fica em useHabitTimer.
interface CheckInDockState {
  habitId: string | null
  open: (habitId: string) => void
  close: () => void
}

export const useCheckInDock = create<CheckInDockState>()(
  persist(
    (set) => ({
      habitId: null,
      open: (habitId) => set({ habitId }),
      close: () => set({ habitId: null }),
    }),
    { name: 'myroutine-checkin-dock' }
  )
)
