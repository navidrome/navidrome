import { describe, it, beforeEach, afterEach, vi, expect } from 'vitest'
import config from './config'
import authProvider from './authProvider'

describe('authProvider', () => {
  beforeEach(() => {
    localStorage.clear()
    sessionStorage.clear()
    localStorage.setItem('is-authenticated', 'true')
    localStorage.setItem('token', 'test-token')
    localStorage.setItem('userId', 'test-user')
    localStorage.setItem('role', 'admin')
    config.extAuthLogoutURL = ''
  })

  afterEach(() => {
    config.extAuthLogoutURL = ''
  })

  describe('checkError', () => {
    it('rejects and clears storage on 401', async () => {
      await expect(authProvider.checkError({ status: 401 })).rejects.toBe(
        undefined,
      )
      expect(localStorage.getItem('is-authenticated')).toBeNull()
    })

    it('resolves on non-401 HTTP errors', async () => {
      await expect(
        authProvider.checkError({ status: 500 }),
      ).resolves.toBeUndefined()
      expect(localStorage.getItem('is-authenticated')).toBe('true')
    })

    it('resolves on network error without extAuth configured', async () => {
      config.extAuthLogoutURL = ''
      await expect(
        authProvider.checkError(new TypeError('Failed to fetch')),
      ).resolves.toBeUndefined()
      expect(localStorage.getItem('is-authenticated')).toBe('true')
    })

    it('clears auth and sets reload guard on TypeError with extAuth', () => {
      config.extAuthLogoutURL = 'https://auth.example.com/logout'
      // window.location.reload throws in jsdom, so we catch it
      try {
        authProvider.checkError(new TypeError('Failed to fetch'))
      } catch {
        // jsdom "Not implemented: navigation" is expected
      }
      expect(localStorage.getItem('is-authenticated')).toBeNull()
      expect(sessionStorage.getItem('ext-auth-reload-ts')).not.toBeNull()
    })

    it('clears auth on Firefox NetworkError with extAuth', () => {
      config.extAuthLogoutURL = 'https://auth.example.com/logout'
      const error = new Error('NetworkError when attempting to fetch resource')
      try {
        authProvider.checkError(error)
      } catch {
        // jsdom "Not implemented: navigation" is expected
      }
      expect(localStorage.getItem('is-authenticated')).toBeNull()
      expect(sessionStorage.getItem('ext-auth-reload-ts')).not.toBeNull()
    })

    it('does not reload-loop within 30 seconds', async () => {
      config.extAuthLogoutURL = 'https://auth.example.com/logout'
      sessionStorage.setItem('ext-auth-reload-ts', String(Date.now() - 10000))
      await expect(
        authProvider.checkError(new TypeError('Failed to fetch')),
      ).resolves.toBeUndefined()
      expect(localStorage.getItem('is-authenticated')).toBe('true')
    })

    it('allows reload again after 30 seconds', () => {
      config.extAuthLogoutURL = 'https://auth.example.com/logout'
      sessionStorage.setItem('ext-auth-reload-ts', String(Date.now() - 31000))
      try {
        authProvider.checkError(new TypeError('Failed to fetch'))
      } catch {
        // jsdom "Not implemented: navigation" is expected
      }
      expect(localStorage.getItem('is-authenticated')).toBeNull()
      const ts = parseInt(sessionStorage.getItem('ext-auth-reload-ts'), 10)
      expect(Date.now() - ts).toBeLessThan(5000)
    })
  })

  describe('checkAuth', () => {
    it('resolves when authenticated', async () => {
      await expect(authProvider.checkAuth()).resolves.toBeUndefined()
    })

    it('rejects when not authenticated', async () => {
      localStorage.removeItem('is-authenticated')
      await expect(authProvider.checkAuth()).rejects.toBe(undefined)
    })
  })
})
