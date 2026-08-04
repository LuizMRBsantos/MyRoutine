import { useState, useEffect, useCallback } from 'react'

interface TimerState {
  habitId: string
  elapsedSeconds: number
  isRunning: boolean
  lastStartedAt: string | null
  startedAt: string | null
}

const getStorageKey = (habitId: string) => `habit_timer_${habitId}`

export function useHabitTimer(habitId: string, targetMinutes: number) {
  const targetSeconds = targetMinutes * 60

  const [state, setState] = useState<TimerState>(() => {
    const saved = localStorage.getItem(getStorageKey(habitId))
    if (saved) {
      try {
        const parsed: TimerState = JSON.parse(saved)
        // If it was running, calculate elapsed time since we closed the tab
        if (parsed.isRunning && parsed.lastStartedAt) {
          const now = new Date().getTime()
          const lastStarted = new Date(parsed.lastStartedAt).getTime()
          const diffSeconds = Math.floor((now - lastStarted) / 1000)
          parsed.elapsedSeconds += diffSeconds
          parsed.lastStartedAt = new Date().toISOString()
        }
        return parsed
      } catch (e) {
        console.error('Failed to parse timer state', e)
      }
    }
    return {
      habitId,
      elapsedSeconds: 0,
      isRunning: false,
      lastStartedAt: null,
      startedAt: null,
    }
  })

  // Persist state
  useEffect(() => {
    localStorage.setItem(getStorageKey(habitId), JSON.stringify(state))
  }, [state, habitId])

  // Timer interval
  useEffect(() => {
    let interval: ReturnType<typeof setInterval>
    if (state.isRunning) {
      interval = setInterval(() => {
        setState(prev => ({
          ...prev,
          elapsedSeconds: prev.elapsedSeconds + 1
        }))
      }, 1000)
    }
    return () => clearInterval(interval)
  }, [state.isRunning])

  const start = useCallback(() => {
    const now = new Date().toISOString()
    setState(prev => ({
      ...prev,
      isRunning: true,
      lastStartedAt: now,
      startedAt: prev.startedAt || now,
    }))
  }, [])

  const pause = useCallback(() => {
    setState(prev => ({
      ...prev,
      isRunning: false,
      lastStartedAt: null,
    }))
  }, [])

  const stopAndReset = useCallback(() => {
    setState({
      habitId,
      elapsedSeconds: 0,
      isRunning: false,
      lastStartedAt: null,
      startedAt: null,
    })
  }, [habitId])

  const clear = useCallback(() => {
    localStorage.removeItem(getStorageKey(habitId))
  }, [habitId])

  const isCompleted = state.elapsedSeconds >= targetSeconds
  const progressPercent = Math.min(100, (state.elapsedSeconds / targetSeconds) * 100)

  // Format MM:SS or HH:MM:SS
  const formatTime = (totalSeconds: number) => {
    const h = Math.floor(totalSeconds / 3600)
    const m = Math.floor((totalSeconds % 3600) / 60)
    const s = totalSeconds % 60
    if (h > 0) return `${h.toString().padStart(2, '0')}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`
    return `${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`
  }

  return {
    ...state,
    targetSeconds,
    isCompleted,
    progressPercent,
    formattedTime: formatTime(state.elapsedSeconds),
    start,
    pause,
    stopAndReset,
    clear,
  }
}
