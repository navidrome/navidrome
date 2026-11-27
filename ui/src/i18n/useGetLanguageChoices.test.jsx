import { describe, it, expect, vi } from 'vitest'
import { renderHook } from '@testing-library/react-hooks'
import { useGetList } from 'react-admin'
import en from './en.json'
import useGetLanguageChoices from './useGetLanguageChoices'

vi.mock('react-admin', () => ({
  useGetList: vi.fn(),
}))

const countLeaves = (obj) =>
  Object.values(obj).reduce(
    (sum, v) =>
      sum + (typeof v === 'object' && v !== null ? countLeaves(v) : v ? 1 : 0),
    0,
  )
const enTermCount = countLeaves(en)

// The percentage is wrapped in a left-to-right isolate, so it reads the same
// next to right-to-left language names
const label = (pct) => `⁦(${pct}%)⁩`

const mockLanguages = (languages) => {
  const data = {}
  languages.forEach((l) => (data[l.id] = l))
  useGetList.mockReturnValue({
    ids: languages.map((l) => l.id),
    data,
    loaded: true,
    loading: false,
  })
}

const choiceFor = (id) => {
  const { result } = renderHook(() => useGetLanguageChoices())
  return result.current.choices.find((c) => c.id === id)
}

describe('useGetLanguageChoices', () => {
  it('appends the completion percentage to incomplete languages', () => {
    const termCount = Math.round(enTermCount * 0.62)
    mockLanguages([{ id: 'cs', name: 'Čeština', termCount }])

    const pct = Math.round((100 * termCount) / enTermCount)
    expect(choiceFor('cs').name).toEqual(`Čeština ${label(pct)}`)
  })

  it('shows 100% for a complete language', () => {
    mockLanguages([{ id: 'de', name: 'Deutsch', termCount: enTermCount }])

    expect(choiceFor('de').name).toEqual(`Deutsch ${label(100)}`)
  })

  it('caps the percentage at 100 when a language has extra terms', () => {
    mockLanguages([
      { id: 'pt', name: 'Português', termCount: enTermCount + 20 },
    ])

    expect(choiceFor('pt').name).toEqual(`Português ${label(100)}`)
  })

  it('isolates the percentage next to a right-to-left name', () => {
    const termCount = Math.round(enTermCount * 0.61)
    mockLanguages([{ id: 'ar', name: 'العربية', termCount }])

    const pct = Math.round((100 * termCount) / enTermCount)
    expect(choiceFor('ar').name).toEqual(`العربية ⁦(${pct}%)⁩`)
  })

  it('omits the percentage when the server does not send a term count', () => {
    mockLanguages([{ id: 'fr', name: 'Français' }])

    expect(choiceFor('fr').name).toEqual('Français')
  })

  it('shows 100% for the bundled English', () => {
    mockLanguages([])

    expect(choiceFor('en').name).toEqual(`English ${label(100)}`)
  })

  it('sorts by language name, ignoring the percentage', () => {
    mockLanguages([
      { id: 'no', name: 'Norsk', termCount: enTermCount },
      { id: 'da', name: 'Dansk', termCount: 1 },
    ])

    const { result } = renderHook(() => useGetLanguageChoices())

    expect(result.current.choices.map((c) => c.id)).toEqual(['da', 'en', 'no'])
  })
})
