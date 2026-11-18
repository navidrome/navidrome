import { describe, expect, it } from 'vitest'
import {
  personalOffsetFor,
  personalSelectionLabel,
  personalSelectionUnavailable,
  personalSelectionVariantKey,
} from './lyricsPersonal'

const local = {
  version: 1,
  token: 'local:v1:0123456789abcdef0123456789abcdef',
  kind: 'sidecar',
  format: 'lrc',
  language: 'xxx',
  role: 'main',
}

describe('lyrics personal helpers', () => {
  it('addresses a resolved local or provider selection and ignores unavailable pins', () => {
    expect(
      personalSelectionVariantKey({
        selection: { kind: 'local', local, status: 'pinned' },
      }),
    ).toBe(`local:${local.token}`)
    expect(
      personalSelectionVariantKey({
        selection: { kind: 'provider', selection: 'token', status: 'rebound' },
      }),
    ).toBe('selection:token')
    expect(
      personalSelectionVariantKey({
        selection: { kind: 'local', local, status: 'unavailable' },
      }),
    ).toBeNull()
    expect(
      personalSelectionUnavailable({
        selection: { kind: 'local', local, status: 'unavailable' },
      }),
    ).toBe(true)
  })

  it('finds the offset for the displayed local or provider variant', () => {
    const personal = {
      offset: [
        { kind: 'local', local, offsetMs: 500 },
        { kind: 'provider', selection: 'token', offsetMs: -250 },
      ],
    }
    expect(personalOffsetFor(personal, { local: local.token })).toBe(500)
    expect(personalOffsetFor(personal, { selection: 'token' })).toBe(-250)
    expect(personalOffsetFor(personal, { selection: 'other' })).toBe(0)
  })

  it('labels an unavailable local choice without an unknown language tag', () => {
    const label = personalSelectionLabel(
      { selection: { kind: 'local', local, status: 'unavailable' } },
      { sidecarSource: 'Sidecar file' },
    )
    expect(label).toBe('Sidecar file · LRC')
  })
})
