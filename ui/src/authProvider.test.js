import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import config from './config'
import authProvider from './authProvider'

vi.mock('./config', () => ({ default: {} }))

describe('authProvider.logout', () => {
  const logoutURL = 'https://auth.example.com/signout'

  beforeEach(() => {
    vi.stubGlobal('location', { href: '' })
    localStorage.setItem('is-authenticated', 'true')
    localStorage.setItem('token', 'abc')
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    localStorage.clear()
    delete config.extAuthLogoutURL
    delete config.auth
  })

  it('clears the stored session', async () => {
    await authProvider.logout()
    expect(localStorage.getItem('is-authenticated')).toBeNull()
    expect(localStorage.getItem('token')).toBeNull()
  })

  it('does not redirect when no logout URL is configured', async () => {
    config.auth = { id: '1' }
    await expect(authProvider.logout()).resolves.toBeUndefined()
    expect(window.location.href).toBe('')
  })

  it('redirects to the logout URL when the page was authenticated by the proxy', async () => {
    config.extAuthLogoutURL = logoutURL
    config.auth = { id: '1' }
    await expect(authProvider.logout()).resolves.toBe(false)
    expect(window.location.href).toBe(logoutURL)
  })

  it('does not redirect when the page was not authenticated by the proxy', async () => {
    config.extAuthLogoutURL = logoutURL
    await expect(authProvider.logout()).resolves.toBeUndefined()
    expect(window.location.href).toBe('')
  })
})

describe('authProvider.checkError', () => {
  const logoutURL = 'https://auth.example.com/signout'

  beforeEach(() => {
    sessionStorage.clear()
    localStorage.setItem('is-authenticated', 'true')
    localStorage.setItem('token', 'abc')
  })

  afterEach(() => {
    localStorage.clear()
    sessionStorage.clear()
    delete config.extAuthLogoutURL
    delete config.auth
  })

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
    await expect(
      authProvider.checkError(new TypeError('Failed to fetch')),
    ).resolves.toBeUndefined()
    expect(localStorage.getItem('is-authenticated')).toBe('true')
  })

  it('clears auth and sets reload guard on TypeError with extAuth', () => {
    config.extAuthLogoutURL = logoutURL
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
    config.extAuthLogoutURL = logoutURL
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
    config.extAuthLogoutURL = logoutURL
    sessionStorage.setItem('ext-auth-reload-ts', String(Date.now() - 10000))
    await expect(
      authProvider.checkError(new TypeError('Failed to fetch')),
    ).resolves.toBeUndefined()
    expect(localStorage.getItem('is-authenticated')).toBe('true')
  })

  it('allows reload again after 30 seconds', () => {
    config.extAuthLogoutURL = logoutURL
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

describe('authProvider.checkAuth', () => {
  afterEach(() => {
    localStorage.clear()
  })

  it('resolves when authenticated', async () => {
    localStorage.setItem('is-authenticated', 'true')
    await expect(authProvider.checkAuth()).resolves.toBeUndefined()
  })

  it('rejects when not authenticated', async () => {
    await expect(authProvider.checkAuth()).rejects.toBe(undefined)
  })
})
