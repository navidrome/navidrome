import { describe, it, expect, vi } from 'vitest'
import { createNavigationHandler } from './swNavigation'

const OFFLINE = { offlinePage: true }

const setup = (networkResult) => {
  const networkOnly = {
    handle: vi.fn(() =>
      networkResult instanceof Error
        ? Promise.reject(networkResult)
        : Promise.resolve(networkResult),
    ),
  }
  const offlineFallback = vi.fn(() => Promise.resolve(OFFLINE))
  return {
    networkOnly,
    offlineFallback,
    handler: createNavigationHandler(networkOnly, offlineFallback),
  }
}

describe('createNavigationHandler', () => {
  it('serves the response from the network', async () => {
    const response = { status: 200 }
    const { handler, offlineFallback } = setup(response)

    await expect(handler({})).resolves.toBe(response)
    expect(offlineFallback).not.toHaveBeenCalled()
  })

  it('falls back to the offline page when the network fails', async () => {
    const { handler } = setup(new Error('Failed to fetch'))

    await expect(handler({})).resolves.toBe(OFFLINE)
  })

  it.each([500, 502, 503])(
    'falls back to the offline page on %i',
    async (status) => {
      const { handler } = setup({ status })

      await expect(handler({})).resolves.toBe(OFFLINE)
    },
  )

  it.each([304, 401, 404])('passes %i through untouched', async (status) => {
    const response = { status }
    const { handler, offlineFallback } = setup(response)

    await expect(handler({})).resolves.toBe(response)
    expect(offlineFallback).not.toHaveBeenCalled()
  })

  it('passes the navigation params to the network strategy', async () => {
    const params = { request: { url: '/app/' } }
    const { handler, networkOnly } = setup({ status: 200 })

    await handler(params)

    expect(networkOnly.handle).toHaveBeenCalledWith(params)
  })
})
