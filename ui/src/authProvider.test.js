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
  const networkError = () => new TypeError('Failed to fetch')
  const PENDING = Symbol('pending')
  const settled = (promise) => Promise.race([promise, Promise.resolve(PENDING)])

  beforeEach(() => {
    vi.stubGlobal('location', { href: '', reload: vi.fn() })
    sessionStorage.clear()
    localStorage.setItem('is-authenticated', 'true')
    localStorage.setItem('token', 'abc')
  })

  afterEach(() => {
    vi.unstubAllGlobals()
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
    config.auth = { id: '1' }
    await expect(
      authProvider.checkError(networkError()),
    ).resolves.toBeUndefined()
    expect(localStorage.getItem('is-authenticated')).toBe('true')
    expect(window.location.reload).not.toHaveBeenCalled()
  })

  it('resolves on network error when the page was not authenticated by the proxy', async () => {
    config.extAuthLogoutURL = logoutURL
    expect(await settled(authProvider.checkError(networkError()))).toBe(
      undefined,
    )
    expect(localStorage.getItem('is-authenticated')).toBe('true')
    expect(sessionStorage.getItem('ext-auth-reload-ts')).toBeNull()
    expect(window.location.reload).not.toHaveBeenCalled()
  })

  it.each([
    ['Chrome', networkError()],
    [
      'Firefox',
      new TypeError('NetworkError when attempting to fetch resource'),
    ],
    ['Safari', new TypeError('Load failed')],
  ])(
    'clears auth and reloads on %s network error with proxy auth',
    async (_, error) => {
      config.extAuthLogoutURL = logoutURL
      config.auth = { id: '1' }
      expect(await settled(authProvider.checkError(error))).toBe(PENDING)
      expect(localStorage.getItem('is-authenticated')).toBeNull()
      expect(sessionStorage.getItem('ext-auth-reload-ts')).not.toBeNull()
      expect(window.location.reload).toHaveBeenCalledOnce()
    },
  )

  it('does not reload-loop within 30 seconds', async () => {
    config.extAuthLogoutURL = logoutURL
    config.auth = { id: '1' }
    sessionStorage.setItem('ext-auth-reload-ts', String(Date.now() - 10000))
    expect(await settled(authProvider.checkError(networkError()))).toBe(
      undefined,
    )
    expect(localStorage.getItem('is-authenticated')).toBe('true')
    expect(window.location.reload).not.toHaveBeenCalled()
  })

  it('allows reload again after 30 seconds', async () => {
    config.extAuthLogoutURL = logoutURL
    config.auth = { id: '1' }
    sessionStorage.setItem('ext-auth-reload-ts', String(Date.now() - 31000))
    expect(await settled(authProvider.checkError(networkError()))).toBe(PENDING)
    expect(localStorage.getItem('is-authenticated')).toBeNull()
    const ts = Number(sessionStorage.getItem('ext-auth-reload-ts'))
    expect(Date.now() - ts).toBeLessThan(5000)
    expect(window.location.reload).toHaveBeenCalledOnce()
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
