import { renderHook } from '@testing-library/react-hooks'
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useDateLocale } from './useDateLocale'

vi.mock('react-admin', () => ({
  useLocale: vi.fn(),
}))

describe('useDateLocale', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  const renderWith = async (locale, browserLocales) => {
    const { useLocale } = await import('react-admin')
    vi.mocked(useLocale).mockReturnValue(locale)
    vi.spyOn(navigator, 'languages', 'get').mockReturnValue(browserLocales)
    return renderHook(() => useDateLocale()).result
  }

  it('adds the region from the browser when the language has none', async () => {
    const result = await renderWith('en', ['en-GB', 'fr-FR'])
    expect(result.current).toEqual('en-GB')
  })

  it('falls back to the language when no browser entry matches', async () => {
    const result = await renderWith('de', ['en-US', 'fr-FR'])
    expect(result.current).toEqual('de')
  })

  it('keeps a language that already carries a region or script', async () => {
    expect((await renderWith('pt-br', ['pt-PT'])).current).toEqual('pt-br')
    expect((await renderWith('zh-Hans', ['zh-TW'])).current).toEqual('zh-Hans')
  })

  it('matches the browser language case-insensitively', async () => {
    const result = await renderWith('pt', ['PT-PT'])
    expect(result.current).toEqual('PT-PT')
  })

  it('returns undefined when there is no language', async () => {
    const result = await renderWith(undefined, ['en-GB'])
    expect(result.current).toBeUndefined()
  })
})
