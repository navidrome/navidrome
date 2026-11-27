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
