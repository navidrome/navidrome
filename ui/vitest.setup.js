import '@testing-library/jest-dom/vitest'
import { vi } from 'vitest'

// Set up localStorage mock
const localStorageMock = (function () {
  let store = {}

  return {
    getItem: function (key) {
      return store[key] || null
    },
    setItem: function (key, value) {
      store[key] = value.toString()
    },
    clear: function () {
      store = {}
    },
    removeItem: function(key) {
      delete store[key]
    }
  }
})()

Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
})

localStorage.setItem('username', 'admin')
localStorage.setItem('token', 'fake-token')

// Mock EventSource
global.EventSource = class EventSource {
  constructor() {
    this.close = vi.fn()
    this.onmessage = vi.fn()
    this.onerror = vi.fn()
  }
}

// Suppress React 18 warnings about act()
const originalError = console.error
console.error = (...args) => {
  if (/Warning.*not wrapped in act/.test(args[0])) {
    return
  }
  originalError.call(console, ...args)
}