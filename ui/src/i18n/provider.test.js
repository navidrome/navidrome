import { describe, it, expect, vi } from 'vitest'
import en from './en.json'

vi.mock('../dataProvider', () => ({ default: { getOne: vi.fn() } }))

const countLeaves = (obj) =>
  Object.values(obj).reduce(
    (sum, v) =>
      sum + (typeof v === 'object' && v !== null ? countLeaves(v) : v ? 1 : 0),
    0,
  )

describe('i18n provider', () => {
  it('does not mutate the bundled English translations', async () => {
    const before = countLeaves(en)
    await import('./provider')
    expect(countLeaves(en)).toEqual(before)
  })
})
