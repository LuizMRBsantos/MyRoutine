import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach, beforeEach } from 'vitest'

// O authStore usa o middleware persist do zustand, que exige localStorage já
// na importação do módulo. Garantimos uma implementação em memória para o
// ambiente de teste não depender do que o jsdom expõe.
class MemoryStorage implements Storage {
  private data = new Map<string, string>()

  get length() { return this.data.size }
  clear() { this.data.clear() }
  getItem(key: string) { return this.data.get(key) ?? null }
  key(index: number) { return Array.from(this.data.keys())[index] ?? null }
  removeItem(key: string) { this.data.delete(key) }
  setItem(key: string, value: string) { this.data.set(key, String(value)) }
}

const storage = new MemoryStorage()
Object.defineProperty(globalThis, 'localStorage', { value: storage, writable: true })
Object.defineProperty(globalThis, 'sessionStorage', { value: new MemoryStorage(), writable: true })

beforeEach(() => {
  storage.clear()
})

afterEach(() => {
  cleanup()
})
