import { describe, expect, it } from 'vitest'
import { buildSegmentsFromLine } from './lyricsSegments'

describe('buildSegmentsFromLine', () => {
  it('does not reuse lowercase offsets when Unicode case folding expands text', () => {
    const segments = buildSegmentsFromLine({
      value: '\u0130A',
      tokens: [{ value: 'a' }],
    })

    expect(segments).toEqual([
      { text: 'a', token: { value: 'a' }, tokenIndex: 0 },
    ])
  })
})
